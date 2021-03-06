// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/codec"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/vms/components/djtx"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

func TestTxNil(t *testing.T) {
	ctx := NewContext(t)
	c := codec.NewDefault()
	tx := (*Tx)(nil)
	if err := tx.SyntacticVerify(ctx, c, ids.Empty, 0, 1); err == nil {
		t.Fatalf("Should have errored due to nil tx")
	}
	if err := tx.SemanticVerify(nil, nil); err == nil {
		t.Fatalf("Should have errored due to nil tx")
	}
}

func setupCodec() codec.Codec {
	c := codec.NewDefault()
	c.RegisterType(&BaseTx{})
	c.RegisterType(&CreateAssetTx{})
	c.RegisterType(&OperationTx{})
	c.RegisterType(&ImportTx{})
	c.RegisterType(&ExportTx{})
	c.RegisterType(&secp256k1fx.TransferInput{})
	c.RegisterType(&secp256k1fx.MintOutput{})
	c.RegisterType(&secp256k1fx.TransferOutput{})
	c.RegisterType(&secp256k1fx.MintOperation{})
	c.RegisterType(&secp256k1fx.Credential{})
	return c
}

func TestTxEmpty(t *testing.T) {
	ctx := NewContext(t)
	c := setupCodec()
	tx := &Tx{}
	if err := tx.SyntacticVerify(ctx, c, ids.Empty, 0, 1); err == nil {
		t.Fatalf("Should have errored due to nil tx")
	}
}

func TestTxInvalidCredential(t *testing.T) {
	ctx := NewContext(t)
	c := setupCodec()
	c.RegisterType(&djtx.TestVerifiable{})

	tx := &Tx{
		UnsignedTx: &BaseTx{BaseTx: djtx.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Ins: []*djtx.TransferableInput{{
				UTXOID: djtx.UTXOID{
					TxID:        ids.Empty,
					OutputIndex: 0,
				},
				Asset: djtx.Asset{ID: asset},
				In: &secp256k1fx.TransferInput{
					Amt: 20 * units.KiloDjtx,
					Input: secp256k1fx.Input{
						SigIndices: []uint32{
							0,
						},
					},
				},
			}},
		}},
		Creds: []verify.Verifiable{&djtx.TestVerifiable{Err: errors.New("")}},
	}
	if err := tx.SignSECP256K1Fx(c, nil); err != nil {
		t.Fatal(err)
	}

	if err := tx.SyntacticVerify(ctx, c, ids.Empty, 0, 1); err == nil {
		t.Fatalf("Tx should have failed due to an invalid credential")
	}
}

func TestTxInvalidUnsignedTx(t *testing.T) {
	ctx := NewContext(t)
	c := setupCodec()
	c.RegisterType(&djtx.TestVerifiable{})

	tx := &Tx{
		UnsignedTx: &BaseTx{BaseTx: djtx.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Ins: []*djtx.TransferableInput{
				{
					UTXOID: djtx.UTXOID{
						TxID:        ids.Empty,
						OutputIndex: 0,
					},
					Asset: djtx.Asset{ID: asset},
					In: &secp256k1fx.TransferInput{
						Amt: 20 * units.KiloDjtx,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
				{
					UTXOID: djtx.UTXOID{
						TxID:        ids.Empty,
						OutputIndex: 0,
					},
					Asset: djtx.Asset{ID: asset},
					In: &secp256k1fx.TransferInput{
						Amt: 20 * units.KiloDjtx,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
			},
		}},
		Creds: []verify.Verifiable{
			&djtx.TestVerifiable{},
			&djtx.TestVerifiable{},
		},
	}
	if err := tx.SignSECP256K1Fx(c, nil); err != nil {
		t.Fatal(err)
	}

	if err := tx.SyntacticVerify(ctx, c, ids.Empty, 0, 1); err == nil {
		t.Fatalf("Tx should have failed due to an invalid unsigned tx")
	}
}

func TestTxInvalidNumberOfCredentials(t *testing.T) {
	ctx := NewContext(t)
	c := setupCodec()
	c.RegisterType(&djtx.TestVerifiable{})

	tx := &Tx{
		UnsignedTx: &BaseTx{BaseTx: djtx.BaseTx{
			NetworkID:    networkID,
			BlockchainID: chainID,
			Ins: []*djtx.TransferableInput{
				{
					UTXOID: djtx.UTXOID{TxID: ids.Empty, OutputIndex: 0},
					Asset:  djtx.Asset{ID: asset},
					In: &secp256k1fx.TransferInput{
						Amt: 20 * units.KiloDjtx,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
				{
					UTXOID: djtx.UTXOID{TxID: ids.Empty, OutputIndex: 1},
					Asset:  djtx.Asset{ID: asset},
					In: &secp256k1fx.TransferInput{
						Amt: 20 * units.KiloDjtx,
						Input: secp256k1fx.Input{
							SigIndices: []uint32{
								0,
							},
						},
					},
				},
			},
		}},
		Creds: []verify.Verifiable{&djtx.TestVerifiable{}},
	}
	if err := tx.SignSECP256K1Fx(c, nil); err != nil {
		t.Fatal(err)
	}

	if err := tx.SyntacticVerify(ctx, c, ids.Empty, 0, 1); err == nil {
		t.Fatalf("Tx should have failed due to an invalid unsigned tx")
	}
}
