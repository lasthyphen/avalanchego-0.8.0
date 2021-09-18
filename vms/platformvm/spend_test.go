package platformvm

import (
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/components/djtx"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

func TestSemanticVerifySpendUTXOs(t *testing.T) {
	vm, _ := defaultVM()
	vm.Ctx.Lock.Lock()
	defer func() {
		vm.Shutdown()
		vm.Ctx.Lock.Unlock()
	}()

	now := time.Now()
	vm.clock.Set(now)

	unsignedTx := djtx.Metadata{}
	unsignedTx.Initialize([]byte{0}, []byte{1})

	tests := []struct {
		description string
		utxos       []*djtx.UTXO
		ins         []*djtx.TransferableInput
		outs        []*djtx.TransferableOutput
		creds       []verify.Verifiable
		fee         uint64
		assetID     ids.ID
		shouldErr   bool
	}{
		{
			description: "no inputs, no outputs, no fee",
			utxos:       []*djtx.UTXO{},
			ins:         []*djtx.TransferableInput{},
			outs:        []*djtx.TransferableOutput{},
			creds:       []verify.Verifiable{},
			fee:         0,
			assetID:     vm.Ctx.DJTXAssetID,
			shouldErr:   false,
		},
		{
			description: "no inputs, no outputs, positive fee",
			utxos:       []*djtx.UTXO{},
			ins:         []*djtx.TransferableInput{},
			outs:        []*djtx.TransferableOutput{},
			creds:       []verify.Verifiable{},
			fee:         1,
			assetID:     vm.Ctx.DJTXAssetID,
			shouldErr:   true,
		},
		{
			description: "no inputs, no outputs, positive fee",
			utxos:       []*djtx.UTXO{},
			ins:         []*djtx.TransferableInput{},
			outs:        []*djtx.TransferableOutput{},
			creds:       []verify.Verifiable{},
			fee:         1,
			assetID:     vm.Ctx.DJTXAssetID,
			shouldErr:   true,
		},
		{
			description: "one input, no outputs, positive fee",
			utxos: []*djtx.UTXO{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*djtx.TransferableInput{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*djtx.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.DJTXAssetID,
			shouldErr: false,
		},
		{
			description: "wrong number of credentials",
			utxos: []*djtx.UTXO{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*djtx.TransferableInput{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs:      []*djtx.TransferableOutput{},
			creds:     []verify.Verifiable{},
			fee:       1,
			assetID:   vm.Ctx.DJTXAssetID,
			shouldErr: true,
		},
		{
			description: "wrong number of UTXOs",
			utxos:       []*djtx.UTXO{},
			ins: []*djtx.TransferableInput{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*djtx.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.DJTXAssetID,
			shouldErr: true,
		},
		{
			description: "invalid credential",
			utxos: []*djtx.UTXO{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*djtx.TransferableInput{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*djtx.TransferableOutput{},
			creds: []verify.Verifiable{
				(*secp256k1fx.Credential)(nil),
			},
			fee:       1,
			assetID:   vm.Ctx.DJTXAssetID,
			shouldErr: true,
		},
		{
			description: "one input, no outputs, positive fee",
			utxos: []*djtx.UTXO{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*djtx.TransferableInput{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*djtx.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.DJTXAssetID,
			shouldErr: false,
		},
		{
			description: "locked one input, no outputs, no fee",
			utxos: []*djtx.UTXO{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				Out: &StakeableLockOut{
					Locktime: uint64(now.Unix()) + 1,
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			}},
			ins: []*djtx.TransferableInput{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				In: &StakeableLockIn{
					Locktime: uint64(now.Unix()) + 1,
					TransferableIn: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			}},
			outs: []*djtx.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       0,
			assetID:   vm.Ctx.DJTXAssetID,
			shouldErr: false,
		},
		{
			description: "locked one input, no outputs, positive fee",
			utxos: []*djtx.UTXO{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				Out: &StakeableLockOut{
					Locktime: uint64(now.Unix()) + 1,
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			}},
			ins: []*djtx.TransferableInput{{
				Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
				In: &StakeableLockIn{
					Locktime: uint64(now.Unix()) + 1,
					TransferableIn: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			}},
			outs: []*djtx.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.DJTXAssetID,
			shouldErr: true,
		},
		{
			description: "one locked one unlock input, one locked output, positive fee",
			utxos: []*djtx.UTXO{
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*djtx.TransferableInput{
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					In: &StakeableLockIn{
						Locktime: uint64(now.Unix()) + 1,
						TransferableIn: &secp256k1fx.TransferInput{
							Amt: 1,
						},
					},
				},
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*djtx.TransferableOutput{
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.DJTXAssetID,
			shouldErr: false,
		},
		{
			description: "one locked one unlock input, one locked output, positive fee, partially locked",
			utxos: []*djtx.UTXO{
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 2,
					},
				},
			},
			ins: []*djtx.TransferableInput{
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					In: &StakeableLockIn{
						Locktime: uint64(now.Unix()) + 1,
						TransferableIn: &secp256k1fx.TransferInput{
							Amt: 1,
						},
					},
				},
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 2,
					},
				},
			},
			outs: []*djtx.TransferableOutput{
				{
					Asset: djtx.Asset{ID: vm.Ctx.DJTXAssetID},
					Out: &StakeableLockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 2,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
				&secp256k1fx.Credential{},
			},
			fee:       1,
			assetID:   vm.Ctx.DJTXAssetID,
			shouldErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			err := vm.semanticVerifySpendUTXOs(
				&unsignedTx,
				test.utxos,
				test.ins,
				test.outs,
				test.creds,
				test.fee,
				test.assetID,
			)

			if err == nil && test.shouldErr {
				t.Fatalf("expected error but got none")
			} else if err != nil && !test.shouldErr {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
