// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"errors"

	"github.com/ava-labs/avalanchego/chains/atomic"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/codec"
	"github.com/ava-labs/avalanchego/vms/components/djtx"
	"github.com/ava-labs/avalanchego/vms/components/verify"
)

var (
	errNoExportOutputs = errors.New("no export outputs")
)

// ExportTx is a transaction that exports an asset to another blockchain.
type ExportTx struct {
	BaseTx `serialize:"true"`

	// Which chain to send the funds to
	DestinationChain ids.ID `serialize:"true" json:"destinationChain"`

	// The outputs this transaction is sending to the other chain
	ExportedOuts []*djtx.TransferableOutput `serialize:"true" json:"exportedOutputs"`
}

// SyntacticVerify that this transaction is well-formed.
func (t *ExportTx) SyntacticVerify(
	ctx *snow.Context,
	c codec.Codec,
	txFeeAssetID ids.ID,
	txFee uint64,
	_ int,
) error {
	switch {
	case t == nil:
		return errNilTx
	case t.DestinationChain.IsZero():
		return errWrongBlockchainID
	case len(t.ExportedOuts) == 0:
		return errNoExportOutputs
	}

	if err := t.MetadataVerify(ctx); err != nil {
		return err
	}

	return djtx.VerifyTx(
		txFee,
		txFeeAssetID,
		[][]*djtx.TransferableInput{t.Ins},
		[][]*djtx.TransferableOutput{
			t.Outs,
			t.ExportedOuts,
		},
		c,
	)
}

// SemanticVerify that this transaction is valid to be spent.
func (t *ExportTx) SemanticVerify(vm *VM, tx UnsignedTx, creds []verify.Verifiable) error {
	subnetID, err := vm.ctx.SNLookup.SubnetID(t.DestinationChain)
	if err != nil {
		return err
	}
	if !vm.ctx.SubnetID.Equals(subnetID) || t.DestinationChain.Equals(vm.ctx.ChainID) {
		return errWrongBlockchainID
	}

	for _, out := range t.ExportedOuts {
		fxIndex, err := vm.getFx(out.Out)
		if err != nil {
			return err
		}
		assetID := out.AssetID()
		if !out.AssetID().Equals(vm.ctx.DJTXAssetID) {
			return errWrongAssetID
		}
		if !vm.verifyFxUsage(fxIndex, assetID) {
			return errIncompatibleFx
		}
	}

	return t.BaseTx.SemanticVerify(vm, tx, creds)
}

// ExecuteWithSideEffects writes the batch with any additional side effects
func (t *ExportTx) ExecuteWithSideEffects(vm *VM, batch database.Batch) error {
	txID := t.ID()

	elems := make([]*atomic.Element, len(t.ExportedOuts))
	for i, out := range t.ExportedOuts {
		utxo := &djtx.UTXO{
			UTXOID: djtx.UTXOID{
				TxID:        txID,
				OutputIndex: uint32(len(t.Outs) + i),
			},
			Asset: djtx.Asset{ID: out.AssetID()},
			Out:   out.Out,
		}

		utxoBytes, err := vm.codec.Marshal(utxo)
		if err != nil {
			return err
		}

		elem := &atomic.Element{
			Key:   utxo.InputID().Bytes(),
			Value: utxoBytes,
		}
		if out, ok := utxo.Out.(djtx.Addressable); ok {
			elem.Traits = out.Addresses()
		}

		elems[i] = elem
	}

	return vm.ctx.SharedMemory.Put(t.DestinationChain, elems, batch)
}
