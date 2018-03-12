// Copyright 2015 The go-ethereum Authors
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

package vap

import (
	"context"
	"math/big"

	"github.com/vaporyco/go-vapory/accounts"
	"github.com/vaporyco/go-vapory/common"
	"github.com/vaporyco/go-vapory/common/math"
	"github.com/vaporyco/go-vapory/core"
	"github.com/vaporyco/go-vapory/core/bloombits"
	"github.com/vaporyco/go-vapory/core/state"
	"github.com/vaporyco/go-vapory/core/types"
	"github.com/vaporyco/go-vapory/core/vm"
	"github.com/vaporyco/go-vapory/vap/downloader"
	"github.com/vaporyco/go-vapory/vap/gasprice"
	"github.com/vaporyco/go-vapory/vapdb"
	"github.com/vaporyco/go-vapory/event"
	"github.com/vaporyco/go-vapory/params"
	"github.com/vaporyco/go-vapory/rpc"
)

// EthApiBackend implements vapapi.Backend for full nodes
type EthApiBackend struct {
	vap *Vapory
	gpo *gasprice.Oracle
}

func (b *EthApiBackend) ChainConfig() *params.ChainConfig {
	return b.vap.chainConfig
}

func (b *EthApiBackend) CurrentBlock() *types.Block {
	return b.vap.blockchain.CurrentBlock()
}

func (b *EthApiBackend) SetHead(number uint64) {
	b.vap.protocolManager.downloader.Cancel()
	b.vap.blockchain.SetHead(number)
}

func (b *EthApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.vap.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.vap.blockchain.CurrentBlock().Header(), nil
	}
	return b.vap.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *EthApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.vap.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.vap.blockchain.CurrentBlock(), nil
	}
	return b.vap.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *EthApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.vap.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.vap.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *EthApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.vap.blockchain.GetBlockByHash(blockHash), nil
}

func (b *EthApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.vap.chainDb, blockHash, core.GetBlockNumber(b.vap.chainDb, blockHash)), nil
}

func (b *EthApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.vap.blockchain.GetTdByHash(blockHash)
}

func (b *EthApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.vap.BlockChain(), nil)
	return vm.NewEVM(context, state, b.vap.chainConfig, vmCfg), vmError, nil
}

func (b *EthApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.vap.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *EthApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.vap.BlockChain().SubscribeChainEvent(ch)
}

func (b *EthApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.vap.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *EthApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.vap.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *EthApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.vap.BlockChain().SubscribeLogsEvent(ch)
}

func (b *EthApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.vap.txPool.AddLocal(signedTx)
}

func (b *EthApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.vap.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *EthApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.vap.txPool.Get(hash)
}

func (b *EthApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.vap.txPool.State().GetNonce(addr), nil
}

func (b *EthApiBackend) Stats() (pending int, queued int) {
	return b.vap.txPool.Stats()
}

func (b *EthApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.vap.TxPool().Content()
}

func (b *EthApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.vap.TxPool().SubscribeTxPreEvent(ch)
}

func (b *EthApiBackend) Downloader() *downloader.Downloader {
	return b.vap.Downloader()
}

func (b *EthApiBackend) ProtocolVersion() int {
	return b.vap.EthVersion()
}

func (b *EthApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *EthApiBackend) ChainDb() vapdb.Database {
	return b.vap.ChainDb()
}

func (b *EthApiBackend) EventMux() *event.TypeMux {
	return b.vap.EventMux()
}

func (b *EthApiBackend) AccountManager() *accounts.Manager {
	return b.vap.AccountManager()
}

func (b *EthApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.vap.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *EthApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.vap.bloomRequests)
	}
}
