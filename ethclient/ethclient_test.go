// Copyright 2016 The go-ethereum Authors
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

package ethclient

import "github.com/vaporyco/go-vapory"

// Verify that Client implements the vapory interfaces.
var (
	_ = vapory.ChainReader(&Client{})
	_ = vapory.TransactionReader(&Client{})
	_ = vapory.ChainStateReader(&Client{})
	_ = vapory.ChainSyncReader(&Client{})
	_ = vapory.ContractCaller(&Client{})
	_ = vapory.GasEstimator(&Client{})
	_ = vapory.GasPricer(&Client{})
	_ = vapory.LogFilterer(&Client{})
	_ = vapory.PendingStateReader(&Client{})
	// _ = vapory.PendingStateEventer(&Client{})
	_ = vapory.PendingContractCaller(&Client{})
)
