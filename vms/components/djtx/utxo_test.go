// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package djtx

import (
	"bytes"
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/codec"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

func TestUTXOVerifyNil(t *testing.T) {
	utxo := (*UTXO)(nil)

	if err := utxo.Verify(); err == nil {
		t.Fatalf("Should have errored due to a nil utxo")
	}
}

func TestUTXOVerifyEmpty(t *testing.T) {
	utxo := &UTXO{
		UTXOID: UTXOID{TxID: ids.Empty},
		Asset:  Asset{ID: ids.Empty},
	}

	if err := utxo.Verify(); err == nil {
		t.Fatalf("Should have errored due to an empty utxo")
	}
}

func TestUTXOSerialize(t *testing.T) {
	c := codec.NewDefault()
	c.RegisterType(&secp256k1fx.MintOutput{})
	c.RegisterType(&secp256k1fx.TransferOutput{})
	c.RegisterType(&secp256k1fx.Input{})
	c.RegisterType(&secp256k1fx.TransferInput{})
	c.RegisterType(&secp256k1fx.Credential{})

	expected := []byte{
		// Codec version
		0x00, 0x00,
		// txID:
		0xf9, 0x66, 0x75, 0x0f, 0x43, 0x88, 0x67, 0xc3,
		0xc9, 0x82, 0x8d, 0xdc, 0xdb, 0xe6, 0x60, 0xe2,
		0x1c, 0xcd, 0xbb, 0x36, 0xa9, 0x27, 0x69, 0x58,
		0xf0, 0x11, 0xba, 0x47, 0x2f, 0x75, 0xd4, 0xe7,
		// utxo index:
		0x00, 0x00, 0x00, 0x00,
		// assetID:
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
		// output:
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xd4, 0x31, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x02, 0x00, 0x01, 0x02, 0x03,
		0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b,
		0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13,
		0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b,
		0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23,
		0x24, 0x25, 0x26, 0x27,
	}
	utxo := &UTXO{
		UTXOID: UTXOID{
			TxID: ids.NewID([32]byte{
				0xf9, 0x66, 0x75, 0x0f, 0x43, 0x88, 0x67, 0xc3,
				0xc9, 0x82, 0x8d, 0xdc, 0xdb, 0xe6, 0x60, 0xe2,
				0x1c, 0xcd, 0xbb, 0x36, 0xa9, 0x27, 0x69, 0x58,
				0xf0, 0x11, 0xba, 0x47, 0x2f, 0x75, 0xd4, 0xe7,
			}),
			OutputIndex: 0,
		},
		Asset: Asset{
			ID: ids.NewID([32]byte{
				0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
				0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
				0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
				0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
			}),
		},
		Out: &secp256k1fx.TransferOutput{
			Amt: 12345,
			OutputOwners: secp256k1fx.OutputOwners{
				Locktime:  54321,
				Threshold: 1,
				Addrs: []ids.ShortID{
					ids.NewShortID([20]byte{
						0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
						0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
						0x10, 0x11, 0x12, 0x13,
					}),
					ids.NewShortID([20]byte{
						0x14, 0x15, 0x16, 0x17,
						0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
					}),
				},
			},
		},
	}

	utxoBytes, err := c.Marshal(utxo)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(utxoBytes, expected) {
		t.Fatalf("Expected:\n%s\nResult:\n%s",
			formatting.DumpBytes{Bytes: expected},
			formatting.DumpBytes{Bytes: utxoBytes},
		)
	}
}
