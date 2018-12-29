// Copyright 2014 The go-ethereum Authors
// This file is part of the go-vapory library.
//
// The go-vapory library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-vapory library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-vapory library. If not, see <http://www.gnu.org/licenses/>.

// Package vap implements the Vapory protocol.
package vap

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/vaporyco/go-vapory/accounts"
	"github.com/vaporyco/go-vapory/common"
	"github.com/vaporyco/go-vapory/common/hexutil"
	"github.com/vaporyco/go-vapory/consensus"
	"github.com/vaporyco/go-vapory/consensus/clique"
	"github.com/vaporyco/go-vapory/consensus/vapash"
	"github.com/vaporyco/go-vapory/core"
	"github.com/vaporyco/go-vapory/core/bloombits"
	"github.com/vaporyco/go-vapory/core/types"
	"github.com/vaporyco/go-vapory/core/vm"
	"github.com/vaporyco/go-vapory/vap/downloader"
	"github.com/vaporyco/go-vapory/vap/filters"
	"github.com/vaporyco/go-vapory/vap/gasprice"
	"github.com/vaporyco/go-vapory/vapdb"
	"github.com/vaporyco/go-vapory/event"
	"github.com/vaporyco/go-vapory/internal/vapapi"
	"github.com/vaporyco/go-vapory/log"
	"github.com/vaporyco/go-vapory/miner"
	"github.com/vaporyco/go-vapory/node"
	"github.com/vaporyco/go-vapory/p2p"
	"github.com/vaporyco/go-vapory/params"
	"github.com/vaporyco/go-vapory/rlp"
	"github.com/vaporyco/go-vapory/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// Vapory implements the Vapory full node service.
type Vapory struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the vapory
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb vapdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *VapApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	vaporbase common.Address

	networkId     uint64
	netRPCService *vapapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and vaporbase)
}

func (s *Vapory) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new Vapory object (including the
// initialisation of the common Vapory object)
func New(ctx *node.ServiceContext, config *Config) (*Vapory, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run vap.Vapory in light sync mode, use les.LightVapory")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	vap := &Vapory{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.Vapash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		vaporbase:      config.Vaporbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising Vapory protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gvap upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}

	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	vap.blockchain, err = core.NewBlockChain(chainDb, vap.chainConfig, vap.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		vap.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	vap.bloomIndexer.Start(vap.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	vap.txPool = core.NewTxPool(config.TxPool, vap.chainConfig, vap.blockchain)

	if vap.protocolManager, err = NewProtocolManager(vap.chainConfig, config.SyncMode, config.NetworkId, vap.eventMux, vap.txPool, vap.engine, vap.blockchain, chainDb); err != nil {
		return nil, err
	}
	vap.miner = miner.New(vap, vap.chainConfig, vap.EventMux(), vap.engine)
	vap.miner.SetExtra(makeExtraData(config.ExtraData))

	vap.ApiBackend = &EthApiBackend{vap, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	vap.ApiBackend.gpo = gasprice.NewOracle(vap.ApiBackend, gpoParams)

	return vap, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gvap",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (vapdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*vapdb.LDBDatabase); ok {
		db.Meter("vap/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Vapory service
func CreateConsensusEngine(ctx *node.ServiceContext, config *vapash.Config, chainConfig *params.ChainConfig, db vapdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowMode == vapash.ModeFake:
		log.Warn("Vapash used in fake mode")
		return vapash.NewFaker()
	case config.PowMode == vapash.ModeTest:
		log.Warn("Vapash used in test mode")
		return vapash.NewTester()
	case config.PowMode == vapash.ModeShared:
		log.Warn("Vapash used in shared mode")
		return vapash.NewShared()
	default:
		engine := vapash.New(vapash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the vapory package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Vapory) APIs() []rpc.API {
	apis := vapapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "vap",
			Version:   "1.0",
			Service:   NewPublicVaporyAPI(s),
			Public:    true,
		}, {
			Namespace: "vap",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "vap",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "vap",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Vapory) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Vapory) Vaporbase() (eb common.Address, err error) {
	s.lock.RLock()
	vaporbase := s.vaporbase
	s.lock.RUnlock()

	if vaporbase != (common.Address{}) {
		return vaporbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			vaporbase := accounts[0].Address

			s.lock.Lock()
			s.vaporbase = vaporbase
			s.lock.Unlock()

			log.Info("Vaporbase automatically configured", "address", vaporbase)
			return vaporbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("vaporbase must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *Vapory) SetVaporbase(vaporbase common.Address) {
	self.lock.Lock()
	self.vaporbase = vaporbase
	self.lock.Unlock()

	self.miner.SetVaporbase(vaporbase)
}

func (s *Vapory) StartMining(local bool) error {
	eb, err := s.Vaporbase()
	if err != nil {
		log.Error("Cannot start mining without vaporbase", "err", err)
		return fmt.Errorf("vaporbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Vaporbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *Vapory) StopMining()         { s.miner.Stop() }
func (s *Vapory) IsMining() bool      { return s.miner.Mining() }
func (s *Vapory) Miner() *miner.Miner { return s.miner }

func (s *Vapory) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Vapory) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Vapory) TxPool() *core.TxPool               { return s.txPool }
func (s *Vapory) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Vapory) Engine() consensus.Engine           { return s.engine }
func (s *Vapory) ChainDb() vapdb.Database            { return s.chainDb }
func (s *Vapory) IsListening() bool                  { return true } // Always listening
func (s *Vapory) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Vapory) NetVersion() uint64                 { return s.networkId }
func (s *Vapory) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Vapory) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Vapory protocol implementation.
func (s *Vapory) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = vapapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		maxPeers -= s.config.LightPeers
		if maxPeers < srvr.MaxPeers/2 {
			maxPeers = srvr.MaxPeers / 2
		}
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Vapory protocol.
func (s *Vapory) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
