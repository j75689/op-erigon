package opstack

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/types"
	"github.com/stretchr/testify/require"
)

// This file is based on op-geth
// https://github.com/ethereum-optimism/op-geth/commit/a290ca164a36c80a8d106d88bd482b6f82220bef

var (
	basefee  = uint256.NewInt(1000 * 1e6)
	overhead = uint256.NewInt(50)
	scalar   = uint256.NewInt(7 * 1e6)

	blobBasefee       = uint256.NewInt(10 * 1e6)
	basefeeScalar     = uint256.NewInt(2)
	blobBasefeeScalar = uint256.NewInt(3)

	// below are the expected cost func outcomes for the above parameter settings on the emptyTx
	// which is defined in transaction_test.go
	bedrockFee  = uint256.NewInt(11326000000000)
	regolithFee = uint256.NewInt(3710000000000)
	ecotoneFee  = uint256.NewInt(960900) // (480/16)*(2*16*1000 + 3*10) == 960900

	bedrockGas  = uint256.NewInt(1618)
	regolithGas = uint256.NewInt(530) // 530  = 1618 - (16*68)
	ecotoneGas  = uint256.NewInt(480)

	OptimismTestConfig = &chain.OptimismConfig{EIP1559Elasticity: 50, EIP1559Denominator: 10}

	// RollupCostData of emptyTx
	emptyTxRollupCostData = types.RollupCostData{Zeroes: 0, Ones: 30}
)

func TestBedrockL1CostFunc(t *testing.T) {
	costFunc0 := newL1CostFuncBedrockHelper(basefee, overhead, scalar, false /*isRegolith*/)
	costFunc1 := newL1CostFuncBedrockHelper(basefee, overhead, scalar, true)

	c0, g0 := costFunc0(emptyTxRollupCostData) // pre-Regolith
	c1, g1 := costFunc1(emptyTxRollupCostData)

	require.Equal(t, bedrockFee, c0)
	require.Equal(t, bedrockGas, g0) // gas-used

	require.Equal(t, regolithFee, c1)
	require.Equal(t, regolithGas, g1)
}

func TestEcotoneL1CostFunc(t *testing.T) {
	costFunc := newL1CostFuncEcotone(basefee, blobBasefee, basefeeScalar, blobBasefeeScalar)
	c, g := costFunc(emptyTxRollupCostData)
	require.Equal(t, ecotoneGas, g)
	require.Equal(t, ecotoneFee, c)
}

func TestExtractBedrockGasParams(t *testing.T) {
	regolithTime := uint64(1)
	config := &chain.Config{
		Optimism:     OptimismTestConfig,
		RegolithTime: big.NewInt(1),
	}

	data := getBedrockL1Attributes(basefee, overhead, scalar)

	_, costFuncPreRegolith, _, err := ExtractL1GasParams(config, regolithTime-1, data)
	require.NoError(t, err)

	// Function should continue to succeed even with extra data (that just gets ignored) since we
	// have been testing the data size is at least the expected number of bytes instead of exactly
	// the expected number of bytes. It's unclear if this flexibility was intentional, but since
	// it's been in production we shouldn't change this behavior.
	data = append(data, []byte{0xBE, 0xEE, 0xEE, 0xFF}...) // tack on garbage data
	_, costFuncRegolith, _, err := ExtractL1GasParams(config, regolithTime, data)
	require.NoError(t, err)

	c, _ := costFuncPreRegolith(emptyTxRollupCostData)
	require.Equal(t, bedrockFee, c)

	c, _ = costFuncRegolith(emptyTxRollupCostData)
	require.Equal(t, regolithFee, c)

	// try to extract from data which has not enough params, should get error.
	data = data[:len(data)-4-32]
	_, _, _, err = ExtractL1GasParams(config, regolithTime, data)
	require.Error(t, err)
}

func TestExtractEcotoneGasParams(t *testing.T) {
	zeroTime := big.NewInt(0)
	// create a config where ecotone upgrade is active
	config := &chain.Config{
		Optimism:     OptimismTestConfig,
		RegolithTime: zeroTime,
		EcotoneTime:  zeroTime,
	}
	require.True(t, config.IsOptimismEcotone(0))

	data := getEcotoneL1Attributes(basefee, blobBasefee, basefeeScalar, blobBasefeeScalar)

	_, costFunc, _, err := ExtractL1GasParams(config, 0, data)
	require.NoError(t, err)

	c, g := costFunc(emptyTxRollupCostData)

	require.Equal(t, ecotoneGas, g)
	require.Equal(t, ecotoneFee, c)

	// make sure wrong amont of data results in error
	data = append(data, 0x00) // tack on garbage byte
	_, _, err = extractL1GasParamsEcotone(data)
	require.Error(t, err)
}

// make sure the first block of the ecotone upgrade is properly detected, and invokes the bedrock
// cost function appropriately
func TestFirstBlockEcotoneGasParams(t *testing.T) {
	zeroTime := big.NewInt(0)
	// create a config where ecotone upgrade is active
	config := &chain.Config{
		Optimism:     OptimismTestConfig,
		RegolithTime: zeroTime,
		EcotoneTime:  zeroTime,
	}
	require.True(t, config.IsOptimismEcotone(0))

	data := getBedrockL1Attributes(basefee, overhead, scalar)

	_, oldCostFunc, _, err := ExtractL1GasParams(config, 0, data)
	require.NoError(t, err)
	c, _ := oldCostFunc(emptyTxRollupCostData)
	require.Equal(t, regolithFee, c)
}

func getBedrockL1Attributes(basefee, overhead, scalar *uint256.Int) []byte {
	uint256Bytes := make([]byte, 32)
	ignored := big.NewInt(1234)
	data := []byte{}
	data = append(data, BedrockL1AttributesSelector...)
	data = append(data, ignored.FillBytes(uint256Bytes)...)          // arg 0
	data = append(data, ignored.FillBytes(uint256Bytes)...)          // arg 1
	data = append(data, basefee.ToBig().FillBytes(uint256Bytes)...)  // arg 2
	data = append(data, ignored.FillBytes(uint256Bytes)...)          // arg 3
	data = append(data, ignored.FillBytes(uint256Bytes)...)          // arg 4
	data = append(data, ignored.FillBytes(uint256Bytes)...)          // arg 5
	data = append(data, overhead.ToBig().FillBytes(uint256Bytes)...) // arg 6
	data = append(data, scalar.ToBig().FillBytes(uint256Bytes)...)   // arg 7
	return data
}

func getEcotoneL1Attributes(basefee, blobBasefee, basefeeScalar, blobBasefeeScalar *uint256.Int) []byte {
	ignored := big.NewInt(1234)
	data := []byte{}
	uint256Bytes := make([]byte, 32)
	uint64Bytes := make([]byte, 8)
	uint32Bytes := make([]byte, 4)
	data = append(data, EcotoneL1AttributesSelector...)
	data = append(data, basefeeScalar.ToBig().FillBytes(uint32Bytes)...)
	data = append(data, blobBasefeeScalar.ToBig().FillBytes(uint32Bytes)...)
	data = append(data, ignored.FillBytes(uint64Bytes)...)
	data = append(data, ignored.FillBytes(uint64Bytes)...)
	data = append(data, ignored.FillBytes(uint64Bytes)...)
	data = append(data, basefee.ToBig().FillBytes(uint256Bytes)...)
	data = append(data, blobBasefee.ToBig().FillBytes(uint256Bytes)...)
	data = append(data, ignored.FillBytes(uint256Bytes)...)
	data = append(data, ignored.FillBytes(uint256Bytes)...)
	return data
}

type testStateGetter struct {
	basefee, blobBasefee, overhead, scalar *uint256.Int
	basefeeScalar, blobBasefeeScalar       uint32
}

func (sg *testStateGetter) GetState(addr common.Address, key *common.Hash, value *uint256.Int) {
	switch *key {
	case L1BaseFeeSlot:
		value.Set(sg.basefee)
	case OverheadSlot:
		value.Set(sg.overhead)
	case ScalarSlot:
		value.Set(sg.scalar)
	case L1BlobBaseFeeSlot:
		value.Set(sg.blobBasefee)
	case L1FeeScalarsSlot:
		offset := scalarSectionStart
		buf := common.Hash{}
		binary.BigEndian.PutUint32(buf[offset:offset+4], sg.basefeeScalar)
		binary.BigEndian.PutUint32(buf[offset+4:offset+8], sg.blobBasefeeScalar)
		value.SetBytes(buf.Bytes())
	default:
		panic("unknown slot")
	}
}

// TestNewL1CostFunc tests that the appropriate cost function is selected based on the
// configuration and statedb values.
func TestNewL1CostFunc(t *testing.T) {
	time := uint64(1)
	config := &chain.Config{
		Optimism: OptimismTestConfig,
	}
	statedb := &testStateGetter{
		basefee:           basefee,
		overhead:          overhead,
		scalar:            scalar,
		blobBasefee:       blobBasefee,
		basefeeScalar:     uint32(basefeeScalar.Uint64()),
		blobBasefeeScalar: uint32(blobBasefeeScalar.Uint64()),
	}

	costFunc := NewL1CostFunc(config, statedb)
	require.NotNil(t, costFunc)

	// empty cost data should result in nil fee
	fee := costFunc(types.RollupCostData{}, time)
	require.Nil(t, fee)

	// emptyTx fee w/ bedrock config should be the bedrock fee
	fee = costFunc(emptyTxRollupCostData, time)
	require.NotNil(t, fee)
	require.Equal(t, bedrockFee, fee)

	// emptyTx fee w/ regolith config should be the regolith fee
	config.RegolithTime = new(big.Int).SetUint64(time)
	costFunc = NewL1CostFunc(config, statedb)
	require.NotNil(t, costFunc)
	fee = costFunc(emptyTxRollupCostData, time)
	require.NotNil(t, fee)
	require.Equal(t, regolithFee, fee)

	// emptyTx fee w/ ecotone config should be the ecotone fee
	config.EcotoneTime = new(big.Int).SetUint64(time)
	costFunc = NewL1CostFunc(config, statedb)
	fee = costFunc(emptyTxRollupCostData, time)
	require.NotNil(t, fee)
	require.Equal(t, ecotoneFee, fee)

	// emptyTx fee w/ ecotone config, but simulate first ecotone block by blowing away the ecotone
	// params. Should result in regolith fee.
	statedb.basefeeScalar = 0
	statedb.blobBasefeeScalar = 0
	statedb.blobBasefee = new(uint256.Int)
	costFunc = NewL1CostFunc(config, statedb)
	fee = costFunc(emptyTxRollupCostData, time)
	require.NotNil(t, fee)
	require.Equal(t, regolithFee, fee)
}
