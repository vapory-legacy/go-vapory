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

package vm

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/vaporyco/go-vapory/common"
	"github.com/vaporyco/go-vapory/common/math"
	"github.com/vaporyco/go-vapory/core/types"
	"github.com/vaporyco/go-vapory/crypto"
	"github.com/vaporyco/go-vapory/params"
)

var (
	bigZero                  = new(big.Int)
	errWriteProtection       = errors.New("vvm: write protection")
	errReturnDataOutOfBounds = errors.New("vvm: return data out of bounds")
	errExecutionReverted     = errors.New("vvm: execution reverted")
	errMaxCodeSizeExceeded   = errors.New("vvm: max code size exceeded")
)

func opAdd(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(math.U256(x.Add(x, y)))

	vvm.interpreter.intPool.put(y)

	return nil, nil
}

func opSub(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(math.U256(x.Sub(x, y)))

	vvm.interpreter.intPool.put(y)

	return nil, nil
}

func opMul(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(math.U256(x.Mul(x, y)))

	vvm.interpreter.intPool.put(y)

	return nil, nil
}

func opDiv(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if y.Sign() != 0 {
		stack.push(math.U256(x.Div(x, y)))
	} else {
		stack.push(new(big.Int))
	}

	vvm.interpreter.intPool.put(y)

	return nil, nil
}

func opSdiv(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := math.S256(stack.pop()), math.S256(stack.pop())
	if y.Sign() == 0 {
		stack.push(new(big.Int))
		return nil, nil
	} else {
		n := new(big.Int)
		if vvm.interpreter.intPool.get().Mul(x, y).Sign() < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Div(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.push(math.U256(res))
	}
	vvm.interpreter.intPool.put(y)
	return nil, nil
}

func opMod(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if y.Sign() == 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(math.U256(x.Mod(x, y)))
	}
	vvm.interpreter.intPool.put(y)
	return nil, nil
}

func opSmod(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := math.S256(stack.pop()), math.S256(stack.pop())

	if y.Sign() == 0 {
		stack.push(new(big.Int))
	} else {
		n := new(big.Int)
		if x.Sign() < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Mod(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.push(math.U256(res))
	}
	vvm.interpreter.intPool.put(y)
	return nil, nil
}

func opExp(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	base, exponent := stack.pop(), stack.pop()
	stack.push(math.Exp(base, exponent))

	vvm.interpreter.intPool.put(base, exponent)

	return nil, nil
}

func opSignExtend(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	back := stack.pop()
	if back.Cmp(big.NewInt(31)) < 0 {
		bit := uint(back.Uint64()*8 + 7)
		num := stack.pop()
		mask := back.Lsh(common.Big1, bit)
		mask.Sub(mask, common.Big1)
		if num.Bit(int(bit)) > 0 {
			num.Or(num, mask.Not(mask))
		} else {
			num.And(num, mask)
		}

		stack.push(math.U256(num))
	}

	vvm.interpreter.intPool.put(back)
	return nil, nil
}

func opNot(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x := stack.pop()
	stack.push(math.U256(x.Not(x)))
	return nil, nil
}

func opLt(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) < 0 {
		stack.push(vvm.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	vvm.interpreter.intPool.put(x, y)
	return nil, nil
}

func opGt(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) > 0 {
		stack.push(vvm.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	vvm.interpreter.intPool.put(x, y)
	return nil, nil
}

func opSlt(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := math.S256(stack.pop()), math.S256(stack.pop())
	if x.Cmp(math.S256(y)) < 0 {
		stack.push(vvm.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	vvm.interpreter.intPool.put(x, y)
	return nil, nil
}

func opSgt(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := math.S256(stack.pop()), math.S256(stack.pop())
	if x.Cmp(y) > 0 {
		stack.push(vvm.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	vvm.interpreter.intPool.put(x, y)
	return nil, nil
}

func opEq(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) == 0 {
		stack.push(vvm.interpreter.intPool.get().SetUint64(1))
	} else {
		stack.push(new(big.Int))
	}

	vvm.interpreter.intPool.put(x, y)
	return nil, nil
}

func opIszero(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x := stack.pop()
	if x.Sign() > 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(vvm.interpreter.intPool.get().SetUint64(1))
	}

	vvm.interpreter.intPool.put(x)
	return nil, nil
}

func opAnd(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.And(x, y))

	vvm.interpreter.intPool.put(y)
	return nil, nil
}

func opOr(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Or(x, y))

	vvm.interpreter.intPool.put(y)
	return nil, nil
}

func opXor(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Xor(x, y))

	vvm.interpreter.intPool.put(y)
	return nil, nil
}

func opByte(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	th, val := stack.pop(), stack.peek()
	if th.Cmp(common.Big32) < 0 {
		b := math.Byte(val, 32, int(th.Int64()))
		val.SetUint64(uint64(b))
	} else {
		val.SetUint64(0)
	}
	vvm.interpreter.intPool.put(th)
	return nil, nil
}

func opAddmod(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(bigZero) > 0 {
		add := x.Add(x, y)
		add.Mod(add, z)
		stack.push(math.U256(add))
	} else {
		stack.push(new(big.Int))
	}

	vvm.interpreter.intPool.put(y, z)
	return nil, nil
}

func opMulmod(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Cmp(bigZero) > 0 {
		mul := x.Mul(x, y)
		mul.Mod(mul, z)
		stack.push(math.U256(mul))
	} else {
		stack.push(new(big.Int))
	}

	vvm.interpreter.intPool.put(y, z)
	return nil, nil
}

func opSha3(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	data := memory.Get(offset.Int64(), size.Int64())
	hash := crypto.Keccak256(data)

	if vvm.vmConfig.EnablePreimageRecording {
		vvm.StateDB.AddPreimage(common.BytesToHash(hash), data)
	}

	stack.push(new(big.Int).SetBytes(hash))

	vvm.interpreter.intPool.put(offset, size)
	return nil, nil
}

func opAddress(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(contract.Address().Big())
	return nil, nil
}

func opBalance(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	addr := common.BigToAddress(stack.pop())
	balance := vvm.StateDB.GetBalance(addr)

	stack.push(new(big.Int).Set(balance))
	return nil, nil
}

func opOrigin(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(vvm.Origin.Big())
	return nil, nil
}

func opCaller(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(contract.Caller().Big())
	return nil, nil
}

func opCallValue(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(vvm.interpreter.intPool.get().Set(contract.value))
	return nil, nil
}

func opCallDataLoad(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(new(big.Int).SetBytes(getDataBig(contract.Input, stack.pop(), big32)))
	return nil, nil
}

func opCallDataSize(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(vvm.interpreter.intPool.get().SetInt64(int64(len(contract.Input))))
	return nil, nil
}

func opCallDataCopy(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		dataOffset = stack.pop()
		length     = stack.pop()
	)
	memory.Set(memOffset.Uint64(), length.Uint64(), getDataBig(contract.Input, dataOffset, length))

	vvm.interpreter.intPool.put(memOffset, dataOffset, length)
	return nil, nil
}

func opReturnDataSize(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(vvm.interpreter.intPool.get().SetUint64(uint64(len(vvm.interpreter.returnData))))
	return nil, nil
}

func opReturnDataCopy(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		dataOffset = stack.pop()
		length     = stack.pop()
	)
	defer vvm.interpreter.intPool.put(memOffset, dataOffset, length)

	end := new(big.Int).Add(dataOffset, length)
	if end.BitLen() > 64 || uint64(len(vvm.interpreter.returnData)) < end.Uint64() {
		return nil, errReturnDataOutOfBounds
	}
	memory.Set(memOffset.Uint64(), length.Uint64(), vvm.interpreter.returnData[dataOffset.Uint64():end.Uint64()])

	return nil, nil
}

func opExtCodeSize(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	a := stack.pop()

	addr := common.BigToAddress(a)
	a.SetInt64(int64(vvm.StateDB.GetCodeSize(addr)))
	stack.push(a)

	return nil, nil
}

func opCodeSize(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	l := vvm.interpreter.intPool.get().SetInt64(int64(len(contract.Code)))
	stack.push(l)
	return nil, nil
}

func opCodeCopy(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	codeCopy := getDataBig(contract.Code, codeOffset, length)
	memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy)

	vvm.interpreter.intPool.put(memOffset, codeOffset, length)
	return nil, nil
}

func opExtCodeCopy(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		addr       = common.BigToAddress(stack.pop())
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	codeCopy := getDataBig(vvm.StateDB.GetCode(addr), codeOffset, length)
	memory.Set(memOffset.Uint64(), length.Uint64(), codeCopy)

	vvm.interpreter.intPool.put(memOffset, codeOffset, length)
	return nil, nil
}

func opGasprice(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(vvm.interpreter.intPool.get().Set(vvm.GasPrice))
	return nil, nil
}

func opBlockhash(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	num := stack.pop()

	n := vvm.interpreter.intPool.get().Sub(vvm.BlockNumber, common.Big257)
	if num.Cmp(n) > 0 && num.Cmp(vvm.BlockNumber) < 0 {
		stack.push(vvm.GetHash(num.Uint64()).Big())
	} else {
		stack.push(new(big.Int))
	}

	vvm.interpreter.intPool.put(num, n)
	return nil, nil
}

func opCoinbase(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(vvm.Coinbase.Big())
	return nil, nil
}

func opTimestamp(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(math.U256(new(big.Int).Set(vvm.Time)))
	return nil, nil
}

func opNumber(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(math.U256(new(big.Int).Set(vvm.BlockNumber)))
	return nil, nil
}

func opDifficulty(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(math.U256(new(big.Int).Set(vvm.Difficulty)))
	return nil, nil
}

func opGasLimit(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(math.U256(new(big.Int).SetUint64(vvm.GasLimit)))
	return nil, nil
}

func opPop(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	vvm.interpreter.intPool.put(stack.pop())
	return nil, nil
}

func opMload(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset := stack.pop()
	val := new(big.Int).SetBytes(memory.Get(offset.Int64(), 32))
	stack.push(val)

	vvm.interpreter.intPool.put(offset)
	return nil, nil
}

func opMstore(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// pop value of the stack
	mStart, val := stack.pop(), stack.pop()
	memory.Set(mStart.Uint64(), 32, math.PaddedBigBytes(val, 32))

	vvm.interpreter.intPool.put(mStart, val)
	return nil, nil
}

func opMstore8(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	off, val := stack.pop().Int64(), stack.pop().Int64()
	memory.store[off] = byte(val & 0xff)

	return nil, nil
}

func opSload(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	loc := common.BigToHash(stack.pop())
	val := vvm.StateDB.GetState(contract.Address(), loc).Big()
	stack.push(val)
	return nil, nil
}

func opSstore(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	loc := common.BigToHash(stack.pop())
	val := stack.pop()
	vvm.StateDB.SetState(contract.Address(), loc, common.BigToHash(val))

	vvm.interpreter.intPool.put(val)
	return nil, nil
}

func opJump(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	pos := stack.pop()
	if !contract.jumpdests.has(contract.CodeHash, contract.Code, pos) {
		nop := contract.GetOp(pos.Uint64())
		return nil, fmt.Errorf("invalid jump destination (%v) %v", nop, pos)
	}
	*pc = pos.Uint64()

	vvm.interpreter.intPool.put(pos)
	return nil, nil
}

func opJumpi(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	pos, cond := stack.pop(), stack.pop()
	if cond.Sign() != 0 {
		if !contract.jumpdests.has(contract.CodeHash, contract.Code, pos) {
			nop := contract.GetOp(pos.Uint64())
			return nil, fmt.Errorf("invalid jump destination (%v) %v", nop, pos)
		}
		*pc = pos.Uint64()
	} else {
		*pc++
	}

	vvm.interpreter.intPool.put(pos, cond)
	return nil, nil
}

func opJumpdest(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	return nil, nil
}

func opPc(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(vvm.interpreter.intPool.get().SetUint64(*pc))
	return nil, nil
}

func opMsize(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(vvm.interpreter.intPool.get().SetInt64(int64(memory.Len())))
	return nil, nil
}

func opGas(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(vvm.interpreter.intPool.get().SetUint64(contract.Gas))
	return nil, nil
}

func opCreate(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		value        = stack.pop()
		offset, size = stack.pop(), stack.pop()
		input        = memory.Get(offset.Int64(), size.Int64())
		gas          = contract.Gas
	)
	if vvm.ChainConfig().IsEIP150(vvm.BlockNumber) {
		gas -= gas / 64
	}

	contract.UseGas(gas)
	res, addr, returnGas, suberr := vvm.Create(contract, input, gas, value)
	// Push item on the stack based on the returned error. If the ruleset is
	// homestead we must check for CodeStoreOutOfGasError (homestead only
	// rule) and treat as an error, if the ruleset is frontier we must
	// ignore this error and pretend the operation was successful.
	if vvm.ChainConfig().IsHomestead(vvm.BlockNumber) && suberr == ErrCodeStoreOutOfGas {
		stack.push(new(big.Int))
	} else if suberr != nil && suberr != ErrCodeStoreOutOfGas {
		stack.push(new(big.Int))
	} else {
		stack.push(addr.Big())
	}
	contract.Gas += returnGas
	vvm.interpreter.intPool.put(value, offset, size)

	if suberr == errExecutionReverted {
		return res, nil
	}
	return nil, nil
}

func opCall(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// Pop gas. The actual gas in in vvm.callGasTemp.
	vvm.interpreter.intPool.put(stack.pop())
	gas := vvm.callGasTemp
	// Pop other call parameters.
	addr, value, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()
	toAddr := common.BigToAddress(addr)
	value = math.U256(value)
	// Get the arguments from the memory.
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if value.Sign() != 0 {
		gas += params.CallStipend
	}
	ret, returnGas, err := vvm.Call(contract, toAddr, args, gas, value)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == errExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	vvm.interpreter.intPool.put(addr, value, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opCallCode(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// Pop gas. The actual gas is in vvm.callGasTemp.
	vvm.interpreter.intPool.put(stack.pop())
	gas := vvm.callGasTemp
	// Pop other call parameters.
	addr, value, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()
	toAddr := common.BigToAddress(addr)
	value = math.U256(value)
	// Get arguments from the memory.
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if value.Sign() != 0 {
		gas += params.CallStipend
	}
	ret, returnGas, err := vvm.CallCode(contract, toAddr, args, gas, value)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == errExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	vvm.interpreter.intPool.put(addr, value, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opDelegateCall(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// Pop gas. The actual gas is in vvm.callGasTemp.
	vvm.interpreter.intPool.put(stack.pop())
	gas := vvm.callGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()
	toAddr := common.BigToAddress(addr)
	// Get arguments from the memory.
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	ret, returnGas, err := vvm.DelegateCall(contract, toAddr, args, gas)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == errExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	vvm.interpreter.intPool.put(addr, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opStaticCall(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// Pop gas. The actual gas is in vvm.callGasTemp.
	vvm.interpreter.intPool.put(stack.pop())
	gas := vvm.callGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()
	toAddr := common.BigToAddress(addr)
	// Get arguments from the memory.
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	ret, returnGas, err := vvm.StaticCall(contract, toAddr, args, gas)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == errExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	vvm.interpreter.intPool.put(addr, inOffset, inSize, retOffset, retSize)
	return ret, nil
}

func opReturn(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	vvm.interpreter.intPool.put(offset, size)
	return ret, nil
}

func opRevert(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	vvm.interpreter.intPool.put(offset, size)
	return ret, nil
}

func opStop(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	return nil, nil
}

func opSuicide(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	balance := vvm.StateDB.GetBalance(contract.Address())
	vvm.StateDB.AddBalance(common.BigToAddress(stack.pop()), balance)

	vvm.StateDB.Suicide(contract.Address())
	return nil, nil
}

// following functions are used by the instruction jump  table

// make log instruction function
func makeLog(size int) executionFunc {
	return func(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		topics := make([]common.Hash, size)
		mStart, mSize := stack.pop(), stack.pop()
		for i := 0; i < size; i++ {
			topics[i] = common.BigToHash(stack.pop())
		}

		d := memory.Get(mStart.Int64(), mSize.Int64())
		vvm.StateDB.AddLog(&types.Log{
			Address: contract.Address(),
			Topics:  topics,
			Data:    d,
			// This is a non-consensus field, but assigned here because
			// core/state doesn't know the current block number.
			BlockNumber: vvm.BlockNumber.Uint64(),
		})

		vvm.interpreter.intPool.put(mStart, mSize)
		return nil, nil
	}
}

// make push instruction function
func makePush(size uint64, pushByteSize int) executionFunc {
	return func(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		codeLen := len(contract.Code)

		startMin := codeLen
		if int(*pc+1) < startMin {
			startMin = int(*pc + 1)
		}

		endMin := codeLen
		if startMin+pushByteSize < endMin {
			endMin = startMin + pushByteSize
		}

		integer := vvm.interpreter.intPool.get()
		stack.push(integer.SetBytes(common.RightPadBytes(contract.Code[startMin:endMin], pushByteSize)))

		*pc += size
		return nil, nil
	}
}

// make push instruction function
func makeDup(size int64) executionFunc {
	return func(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		stack.dup(vvm.interpreter.intPool, int(size))
		return nil, nil
	}
}

// make swap instruction function
func makeSwap(size int64) executionFunc {
	// switch n + 1 otherwise n would be swapped with n
	size += 1
	return func(pc *uint64, vvm *VVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
		stack.swap(int(size))
		return nil, nil
	}
}
