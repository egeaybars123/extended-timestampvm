// (c) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timestampvm

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/version"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var blockchainID = ids.ID{1, 2, 3}
var testDataSig = "6e8b0d92516ee4289145e3b78cea58daac177b1c618beeedbc6cdabd388a6e557f447712381a49554875df353b160858905e44c37e75dad168e1a4abf32dbf6869a2eaeaf042f001c36c43e01996486b91ed929dea0d2c68d8e1fc9d2559fabf"
var testArray [96]byte

// Assert that after initialization, the vm has the state we expect
func TestGenesis(t *testing.T) {
	assert := assert.New(t)
	ctx := context.TODO()
	// Initialize the vm
	vm, _, _, err := newTestVM()
	assert.NoError(err)
	// Verify that the db is initialized
	ok, err := vm.state.IsInitialized()
	assert.NoError(err)
	assert.True(ok)

	// Get lastAccepted
	lastAccepted, err := vm.LastAccepted(ctx)
	assert.NoError(err)
	assert.NotEqual(ids.Empty, lastAccepted)

	// Verify that getBlock returns the genesis block, and the genesis block
	// is the type we expect
	genesisBlock, err := vm.getBlock(lastAccepted) // genesisBlock as snowman.Block
	assert.NoError(err)

	// Verify that the genesis block has the data, signaure
	// and registrant address we expect
	assert.Equal(ids.Empty, genesisBlock.Parent())
	assert.Equal([32]byte{0, 0, 0, 0, 0}, genesisBlock.Data())
	assert.Equal([64]byte{0, 0, 0, 0, 0, 0}, genesisBlock.Signature())
	assert.Equal(ethcommon.HexToAddress("0x0000000000000000000000000000000000000000"), genesisBlock.Reg)
}

func TestHappyPath(t *testing.T) {
	assert := assert.New(t)
	ctx := context.TODO()

	// Initialize the vm
	vm, snowCtx, msgChan, err := newTestVM()
	assert.NoError(err)

	lastAcceptedID, err := vm.LastAccepted(ctx)
	assert.NoError(err)
	genesisBlock, err := vm.getBlock(lastAcceptedID)
	assert.NoError(err)

	// in an actual execution, the engine would set the preference
	assert.NoError(vm.SetPreference(ctx, genesisBlock.ID()))

	testSlice, _ := hex.DecodeString(testDataSig)
	copy(testArray[:], testSlice)

	snowCtx.Lock.Lock()
	vm.proposeBlock(testArray) // propose a value
	snowCtx.Lock.Unlock()

	select { // assert there is a pending tx message to the engine
	case msg := <-msgChan:
		assert.Equal(common.PendingTxs, msg)
	default:
		assert.FailNow("should have been pendingTxs message on channel")
	}

	// build the block
	snowCtx.Lock.Lock()
	snowmanBlock2, err := vm.BuildBlock(ctx)
	assert.NoError(err)

	assert.NoError(snowmanBlock2.Verify(ctx))
	assert.NoError(snowmanBlock2.Accept(ctx))
	assert.NoError(vm.SetPreference(ctx, snowmanBlock2.ID()))

	lastAcceptedID, err = vm.LastAccepted(ctx)
	assert.NoError(err)

	// Should be the block we just accepted
	block2, err := vm.getBlock(lastAcceptedID)
	assert.NoError(err)

	// Assert the block we accepted has the data we expect
	assert.Equal(genesisBlock.ID(), block2.Parent())
	assert.Equal([DataLen]byte{0, 0, 0, 0, 1}, block2.Data())
	assert.Equal(snowmanBlock2.ID(), block2.ID())
	assert.NoError(block2.Verify(ctx))

	vm.proposeBlock(testArray) // propose a block
	snowCtx.Lock.Unlock()

	select { // verify there is a pending tx message to the engine
	case msg := <-msgChan:
		assert.Equal(common.PendingTxs, msg)
	default:
		assert.FailNow("should have been pendingTxs message on channel")
	}

	snowCtx.Lock.Lock()

	// build the block
	snowmanBlock3, err := vm.BuildBlock(ctx)
	assert.NoError(err)
	assert.NoError(snowmanBlock3.Verify(ctx))
	assert.NoError(snowmanBlock3.Accept(ctx))
	assert.NoError(vm.SetPreference(ctx, snowmanBlock3.ID()))

	lastAcceptedID, err = vm.LastAccepted(ctx)
	assert.NoError(err)
	// The block we just accepted
	block3, err := vm.getBlock(lastAcceptedID)
	assert.NoError(err)

	// Assert the block we accepted has the data we expect
	assert.Equal(snowmanBlock2.ID(), block3.Parent())
	assert.Equal([DataLen]byte{0, 0, 0, 0, 2}, block3.Data())
	assert.Equal(snowmanBlock3.ID(), block3.ID())
	assert.NoError(block3.Verify(ctx))

	// Next, check the blocks we added are there
	block2FromState, err := vm.getBlock(block2.ID())
	assert.NoError(err)
	assert.Equal(block2.ID(), block2FromState.ID())

	block3FromState, err := vm.getBlock(snowmanBlock3.ID())
	assert.NoError(err)
	assert.Equal(snowmanBlock3.ID(), block3FromState.ID())

	snowCtx.Lock.Unlock()
}

func TestService(t *testing.T) {
	// Initialize the vm
	assert := assert.New(t)
	// Initialize the vm
	vm, _, _, err := newTestVM()
	assert.NoError(err)
	service := Service{vm}
	assert.NoError(service.GetBlock(nil, &GetBlockArgs{}, &GetBlockReply{}))
}

func TestSetState(t *testing.T) {
	// Initialize the vm
	assert := assert.New(t)
	ctx := context.TODO()
	// Initialize the vm
	vm, _, _, err := newTestVM()
	assert.NoError(err)
	// bootstrapping
	assert.NoError(vm.SetState(ctx, snow.Bootstrapping))
	assert.False(vm.bootstrapped.GetValue())
	// bootstrapped
	assert.NoError(vm.SetState(ctx, snow.NormalOp))
	assert.True(vm.bootstrapped.GetValue())
	// unknown
	unknownState := snow.State(99)
	assert.ErrorIs(vm.SetState(ctx, unknownState), snow.ErrUnknownState)
}

func newTestVM() (*VM, *snow.Context, chan common.Message, error) {
	dbManager := manager.NewMemDB(&version.Semantic{
		Major: 1,
		Minor: 0,
		Patch: 0,
	})
	msgChan := make(chan common.Message, 1)
	vm := &VM{}
	snowCtx := snow.DefaultContextTest()
	snowCtx.ChainID = blockchainID
	err := vm.Initialize(context.TODO(), snowCtx, dbManager, []byte{0, 0, 0, 0, 0}, nil, nil, msgChan, nil, nil)
	return vm, snowCtx, msgChan, err
}
