// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"errors"
	"fmt"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/codec"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/components/djtx"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/nftfx"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

var (
	errWrongNumberOfCredentials = errors.New("should have the same number of credentials as inputs")
)

// UnsignedTx ...
type UnsignedTx interface {
	Initialize(unsignedBytes, bytes []byte)
	ID() ids.ID
	UnsignedBytes() []byte
	Bytes() []byte

	ConsumedAssetIDs() ids.Set
	AssetIDs() ids.Set

	NumCredentials() int
	InputUTXOs() []*djtx.UTXOID
	UTXOs() []*djtx.UTXO

	SyntacticVerify(ctx *snow.Context, c codec.Codec, txFeeAssetID ids.ID, txFee uint64, numFxs int) error
	SemanticVerify(vm *VM, tx UnsignedTx, creds []verify.Verifiable) error
	ExecuteWithSideEffects(vm *VM, batch database.Batch) error
}

// Tx is the core operation that can be performed. The tx uses the UTXO model.
// Specifically, a txs inputs will consume previous txs outputs. A tx will be
// valid if the inputs have the authority to consume the outputs they are
// attempting to consume and the inputs consume sufficient state to produce the
// outputs.
type Tx struct {
	UnsignedTx `serialize:"true" json:"unsignedTx"`

	Creds []verify.Verifiable `serialize:"true" json:"credentials"` // The credentials of this transaction
}

// Credentials describes the authorization that allows the Inputs to consume the
// specified UTXOs. The returned array should not be modified.
func (t *Tx) Credentials() []verify.Verifiable { return t.Creds }

// SyntacticVerify verifies that this transaction is well-formed.
func (t *Tx) SyntacticVerify(
	ctx *snow.Context,
	c codec.Codec,
	txFeeAssetID ids.ID,
	txFee uint64,
	numFxs int,
) error {
	switch {
	case t == nil || t.UnsignedTx == nil:
		return errNilTx
	}

	if err := t.UnsignedTx.SyntacticVerify(ctx, c, txFeeAssetID, txFee, numFxs); err != nil {
		return err
	}

	for _, cred := range t.Creds {
		if err := cred.Verify(); err != nil {
			return err
		}
	}

	if numCreds := t.UnsignedTx.NumCredentials(); numCreds != len(t.Creds) {
		return errWrongNumberOfCredentials
	}
	return nil
}

// SemanticVerify verifies that this transaction is well-formed.
func (t *Tx) SemanticVerify(vm *VM, tx UnsignedTx) error {
	if t == nil {
		return errNilTx
	}

	return t.UnsignedTx.SemanticVerify(vm, tx, t.Creds)
}

// SignSECP256K1Fx ...
func (t *Tx) SignSECP256K1Fx(c codec.Codec, signers [][]*crypto.PrivateKeySECP256K1R) error {
	unsignedBytes, err := c.Marshal(&t.UnsignedTx)
	if err != nil {
		return fmt.Errorf("problem creating transaction: %w", err)
	}

	hash := hashing.ComputeHash256(unsignedBytes)
	for _, keys := range signers {
		cred := &secp256k1fx.Credential{
			Sigs: make([][crypto.SECP256K1RSigLen]byte, len(keys)),
		}
		for i, key := range keys {
			sig, err := key.SignHash(hash)
			if err != nil {
				return fmt.Errorf("problem creating transaction: %w", err)
			}
			copy(cred.Sigs[i][:], sig)
		}
		t.Creds = append(t.Creds, cred)
	}

	signedBytes, err := c.Marshal(t)
	if err != nil {
		return fmt.Errorf("problem creating transaction: %w", err)
	}
	t.Initialize(unsignedBytes, signedBytes)
	return nil
}

// SignNFTFx ...
func (t *Tx) SignNFTFx(c codec.Codec, signers [][]*crypto.PrivateKeySECP256K1R) error {
	unsignedBytes, err := c.Marshal(&t.UnsignedTx)
	if err != nil {
		return fmt.Errorf("problem creating transaction: %w", err)
	}

	hash := hashing.ComputeHash256(unsignedBytes)
	for _, keys := range signers {
		cred := &nftfx.Credential{Credential: secp256k1fx.Credential{
			Sigs: make([][crypto.SECP256K1RSigLen]byte, len(keys)),
		}}
		for i, key := range keys {
			sig, err := key.SignHash(hash)
			if err != nil {
				return fmt.Errorf("problem creating transaction: %w", err)
			}
			copy(cred.Sigs[i][:], sig)
		}
		t.Creds = append(t.Creds, cred)
	}

	signedBytes, err := c.Marshal(t)
	if err != nil {
		return fmt.Errorf("problem creating transaction: %w", err)
	}
	t.Initialize(unsignedBytes, signedBytes)
	return nil
}
