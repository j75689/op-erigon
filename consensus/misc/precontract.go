// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package misc

import (
	"github.com/holiman/uint256"

	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/state"
)

var (
	// WBNBContract WBNB preDeploy contract address
	WBNBContract = libcommon.HexToAddress("0x4200000000000000000000000000000000000006")
	// GovernanceToken contract address
	governanceToken = libcommon.HexToAddress("4200000000000000000000000000000000000042")
	nameSlot        = libcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	symbolSlot      = libcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	// Wrapped BNB
	nameValue, _ = uint256.FromHex("0x5772617070656420424e42000000000000000000000000000000000000000016")
	// WBNB
	symbolValue, _ = uint256.FromHex("0x57424e4200000000000000000000000000000000000000000000000000000008")
)

// ApplyPreContractHardFork modifies the state database according to the hard-fork rules
func ApplyPreContractHardFork(statedb *state.IntraBlockState) {
	statedb.SetState(WBNBContract, &nameSlot, *nameValue)
	statedb.SetState(WBNBContract, &symbolSlot, *symbolValue)
	statedb.Selfdestruct(governanceToken)
}
