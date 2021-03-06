// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"testing"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/codec"
	"github.com/ava-labs/avalanchego/vms/components/djtx"
	"github.com/ava-labs/avalanchego/vms/components/verify"
)

type testOperable struct {
	djtx.TestTransferable `serialize:"true"`

	Outputs []verify.State `serialize:"true"`
}

func (o *testOperable) Outs() []verify.State { return o.Outputs }

func TestOperationVerifyNil(t *testing.T) {
	c := codec.NewDefault()
	op := (*Operation)(nil)
	if err := op.Verify(c); err == nil {
		t.Fatalf("Should have errored due to nil operation")
	}
}

func TestOperationVerifyEmpty(t *testing.T) {
	c := codec.NewDefault()
	op := &Operation{
		Asset: djtx.Asset{ID: ids.Empty},
	}
	if err := op.Verify(c); err == nil {
		t.Fatalf("Should have errored due to empty operation")
	}
}

func TestOperationVerifyUTXOIDsNotSorted(t *testing.T) {
	c := codec.NewDefault()
	op := &Operation{
		Asset: djtx.Asset{ID: ids.Empty},
		UTXOIDs: []*djtx.UTXOID{
			{
				TxID:        ids.Empty,
				OutputIndex: 1,
			},
			{
				TxID:        ids.Empty,
				OutputIndex: 0,
			},
		},
		Op: &testOperable{},
	}
	if err := op.Verify(c); err == nil {
		t.Fatalf("Should have errored due to unsorted utxoIDs")
	}
}

func TestOperationVerify(t *testing.T) {
	c := codec.NewDefault()
	op := &Operation{
		Asset: djtx.Asset{ID: ids.Empty},
		UTXOIDs: []*djtx.UTXOID{
			{
				TxID:        ids.Empty,
				OutputIndex: 1,
			},
		},
		Op: &testOperable{},
	}
	if err := op.Verify(c); err != nil {
		t.Fatal(err)
	}
}

func TestOperationSorting(t *testing.T) {
	c := codec.NewDefault()
	c.RegisterType(&testOperable{})

	ops := []*Operation{
		{
			Asset: djtx.Asset{ID: ids.Empty},
			UTXOIDs: []*djtx.UTXOID{
				{
					TxID:        ids.Empty,
					OutputIndex: 1,
				},
			},
			Op: &testOperable{},
		},
		{
			Asset: djtx.Asset{ID: ids.Empty},
			UTXOIDs: []*djtx.UTXOID{
				{
					TxID:        ids.Empty,
					OutputIndex: 0,
				},
			},
			Op: &testOperable{},
		},
	}
	if isSortedAndUniqueOperations(ops, c) {
		t.Fatalf("Shouldn't be sorted")
	}
	sortOperations(ops, c)
	if !isSortedAndUniqueOperations(ops, c) {
		t.Fatalf("Should be sorted")
	}
	ops = append(ops, &Operation{
		Asset: djtx.Asset{ID: ids.Empty},
		UTXOIDs: []*djtx.UTXOID{
			{
				TxID:        ids.Empty,
				OutputIndex: 1,
			},
		},
		Op: &testOperable{},
	})
	if isSortedAndUniqueOperations(ops, c) {
		t.Fatalf("Shouldn't be unique")
	}
}

func TestOperationTxNotState(t *testing.T) {
	intf := interface{}(&OperationTx{})
	if _, ok := intf.(verify.State); ok {
		t.Fatalf("shouldn't be marked as state")
	}
}
