//  Copyright 2015 The go-ethereum Authors
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

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/vaporyco/go-vapory/metrics"
)

var (
	propAnnounceInMeter   = metrics.NewMeter("vap/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("vap/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("vap/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("vap/fetcher/prop/announces/dos")

	propBroadcastInMeter   = metrics.NewMeter("vap/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("vap/fetcher/prop/broadcasts/out")
	propBroadcastDropMeter = metrics.NewMeter("vap/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("vap/fetcher/prop/broadcasts/dos")

	headerFetchMeter = metrics.NewMeter("vap/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("vap/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("vap/fetcher/filter/headers/in")
	headerFilterOutMeter = metrics.NewMeter("vap/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("vap/fetcher/filter/bodies/in")
	bodyFilterOutMeter   = metrics.NewMeter("vap/fetcher/filter/bodies/out")
)
