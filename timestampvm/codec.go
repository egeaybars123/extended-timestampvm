// (c) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timestampvm

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
)

const (
	// CodecVersion is the current default codec version
	CodecVersion = 0
)

// Codecs do serialization and deserialization
var (
	Codec codec.Manager
)

func init() {
	// Create default codec and manager
	c := linearcodec.NewDefault()
	Codec = codec.NewDefaultManager()

	// Register codec to manager with CodecVersion
	if err := Codec.RegisterCodec(CodecVersion, c); err != nil {
		panic(err)
	}
}

func DecodeMempool(data [DataLen + SigLen]byte) ([32]byte, [64]byte) {
	var hash [32]byte
	var sig [64]byte

	copy(hash[:], data[:DataLen])
	copy(sig[:], data[DataLen:DataLen+SigLen])

	return hash, sig
}
