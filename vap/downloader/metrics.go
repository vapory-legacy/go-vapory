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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/vaporyco/go-vapory/metrics"
)

var (
	headerInMeter      = metrics.NewMeter("vap/downloader/headers/in")
	headerReqTimer     = metrics.NewTimer("vap/downloader/headers/req")
	headerDropMeter    = metrics.NewMeter("vap/downloader/headers/drop")
	headerTimeoutMeter = metrics.NewMeter("vap/downloader/headers/timeout")

	bodyInMeter      = metrics.NewMeter("vap/downloader/bodies/in")
	bodyReqTimer     = metrics.NewTimer("vap/downloader/bodies/req")
	bodyDropMeter    = metrics.NewMeter("vap/downloader/bodies/drop")
	bodyTimeoutMeter = metrics.NewMeter("vap/downloader/bodies/timeout")

	receiptInMeter      = metrics.NewMeter("vap/downloader/receipts/in")
	receiptReqTimer     = metrics.NewTimer("vap/downloader/receipts/req")
	receiptDropMeter    = metrics.NewMeter("vap/downloader/receipts/drop")
	receiptTimeoutMeter = metrics.NewMeter("vap/downloader/receipts/timeout")

	stateInMeter   = metrics.NewMeter("vap/downloader/states/in")
	stateDropMeter = metrics.NewMeter("vap/downloader/states/drop")
)
