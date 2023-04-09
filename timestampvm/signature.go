package timestampvm

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

func SignData(data []byte, privKey *ecdsa.PrivateKey) ([]byte, error) {
	sig, err := crypto.Sign(data[:], privKey)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func Ecrecover(hash, sig []byte) ([]byte, error) {
	return secp256k1.RecoverPubkey(hash, sig)
}

func ValidateSig(data, sig []byte, sender common.Address) (bool, error) {
	ec, err := Ecrecover(data, sig)
	if err != nil {
		return false, err
	}

	x, y := elliptic.Unmarshal(crypto.S256(), ec)
	pub := &ecdsa.PublicKey{Curve: crypto.S256(), X: x, Y: y}
	recoveredProposer := crypto.PubkeyToAddress(*pub)

	if recoveredProposer != sender {
		return false, nil
	}

	compressed := crypto.CompressPubkey(pub)

	// sig does not already have the recover byte.
	result := secp256k1.VerifySignature(compressed, data[:], sig)

	return result, nil
}
