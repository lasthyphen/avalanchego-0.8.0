// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package bootstrap

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/database/prefixdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/avalanche"
	"github.com/ava-labs/avalanchego/snow/consensus/snowstorm"
	"github.com/ava-labs/avalanchego/snow/engine/avalanche/vertex"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/engine/common/queue"
	"github.com/ava-labs/avalanchego/snow/validators"
	"github.com/ava-labs/avalanchego/utils/constants"
)

var (
	errUnknownVertex       = errors.New("unknown vertex")
	errParsedUnknownVertex = errors.New("parsed unknown vertex")
)

func newConfig(t *testing.T) (Config, ids.ShortID, *common.SenderTest, *vertex.TestManager, *vertex.TestVM) {
	ctx := snow.DefaultContextTest()

	peers := validators.NewSet()
	db := memdb.New()
	sender := &common.SenderTest{}
	manager := &vertex.TestManager{}
	vm := &vertex.TestVM{}

	sender.T = t
	manager.T = t
	vm.T = t

	sender.Default(true)
	manager.Default(true)
	vm.Default(true)

	sender.CantGetAcceptedFrontier = false

	peer := ids.GenerateTestShortID()
	peers.AddWeight(peer, 1)

	vtxBlocker, _ := queue.New(prefixdb.New([]byte("vtx"), db))
	txBlocker, _ := queue.New(prefixdb.New([]byte("tx"), db))

	commonConfig := common.Config{
		Ctx:        ctx,
		Validators: peers,
		Beacons:    peers,
		Alpha:      uint64(peers.Len()/2 + 1),
		Sender:     sender,
	}
	return Config{
		Config:     commonConfig,
		VtxBlocked: vtxBlocker,
		TxBlocked:  txBlocker,
		Manager:    manager,
		VM:         vm,
	}, peer, sender, manager, vm
}

// Three vertices in the accepted frontier. None have parents. No need to fetch anything
func TestBootstrapperSingleFrontier(t *testing.T) {
	config, _, _, manager, vm := newConfig(t)

	vtxID0 := ids.Empty.Prefix(0)
	vtxID1 := ids.Empty.Prefix(1)
	vtxID2 := ids.Empty.Prefix(2)

	vtxBytes0 := []byte{0}
	vtxBytes1 := []byte{1}
	vtxBytes2 := []byte{2}

	vtx0 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID0,
			StatusV: choices.Processing,
		},
		HeightV: 0,
		BytesV:  vtxBytes0,
	}
	vtx1 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID1,
			StatusV: choices.Processing,
		},
		HeightV: 0,
		BytesV:  vtxBytes1,
	}
	vtx2 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID2,
			StatusV: choices.Processing,
		},
		HeightV: 0,
		BytesV:  vtxBytes2,
	}

	bs := Bootstrapper{}
	finished := new(bool)
	err := bs.Initialize(
		config,
		func() error { *finished = true; return nil },
		fmt.Sprintf("%s_%s_bs", constants.PlatformName, config.Ctx.ChainID),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatal(err)
	}

	acceptedIDs := ids.Set{}
	acceptedIDs.Add(vtxID0, vtxID1, vtxID2)

	manager.GetVertexF = func(vtxID ids.ID) (avalanche.Vertex, error) {
		switch {
		case vtxID.Equals(vtxID0):
			return vtx0, nil
		case vtxID.Equals(vtxID1):
			return vtx1, nil
		case vtxID.Equals(vtxID2):
			return vtx2, nil
		default:
			t.Fatal(errUnknownVertex)
			panic(errUnknownVertex)
		}
	}

	manager.ParseVertexF = func(vtxBytes []byte) (avalanche.Vertex, error) {
		switch {
		case bytes.Equal(vtxBytes, vtxBytes0):
			return vtx0, nil
		case bytes.Equal(vtxBytes, vtxBytes1):
			return vtx1, nil
		case bytes.Equal(vtxBytes, vtxBytes2):
			return vtx2, nil
		}
		t.Fatal(errParsedUnknownVertex)
		return nil, errParsedUnknownVertex
	}

	vm.CantBootstrapping = false
	vm.CantBootstrapped = false

	bs.ForceAccepted(acceptedIDs)

	switch {
	case !*finished:
		t.Fatalf("Bootstrapping should have finished")
	case vtx0.Status() != choices.Accepted:
		t.Fatalf("Vertex should be accepted")
	case vtx1.Status() != choices.Accepted:
		t.Fatalf("Vertex should be accepted")
	case vtx2.Status() != choices.Accepted:
		t.Fatalf("Vertex should be accepted")
	}
}

// Accepted frontier has one vertex, which has one vertex as a dependency.
// Requests again and gets an unexpected vertex.
// Requests again and gets the expected vertex and an additional vertex that should not be accepted.
func TestBootstrapperByzantineResponses(t *testing.T) {
	config, peerID, sender, manager, vm := newConfig(t)

	vtxID0 := ids.Empty.Prefix(0)
	vtxID1 := ids.Empty.Prefix(1)
	vtxID2 := ids.Empty.Prefix(2)

	vtxBytes0 := []byte{0}
	vtxBytes1 := []byte{1}
	vtxBytes2 := []byte{2}

	vtx0 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID0,
			StatusV: choices.Unknown,
		},
		HeightV: 0,
		BytesV:  vtxBytes0,
	}
	vtx1 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID1,
			StatusV: choices.Processing,
		},
		ParentsV: []avalanche.Vertex{vtx0},
		HeightV:  1,
		BytesV:   vtxBytes1,
	}
	// Should not receive transitive votes from [vtx1]
	vtx2 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID2,
			StatusV: choices.Unknown,
		},
		HeightV: 0,
		BytesV:  vtxBytes2,
	}

	bs := Bootstrapper{}
	finished := new(bool)
	err := bs.Initialize(
		config,
		func() error { *finished = true; return nil },
		fmt.Sprintf("%s_%s_bs", constants.PlatformName, config.Ctx.ChainID),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatal(err)
	}

	acceptedIDs := ids.Set{}
	acceptedIDs.Add(vtxID1)

	manager.GetVertexF = func(vtxID ids.ID) (avalanche.Vertex, error) {
		switch {
		case vtxID.Equals(vtxID1):
			return vtx1, nil
		case vtxID.Equals(vtxID0):
			return nil, errUnknownVertex
		default:
			t.Fatal(errUnknownVertex)
			panic(errUnknownVertex)
		}
	}

	requestID := new(uint32)
	reqVtxID := ids.Empty
	sender.GetAncestorsF = func(vdr ids.ShortID, reqID uint32, vtxID ids.ID) {
		switch {
		case !vdr.Equals(peerID):
			t.Fatalf("Should have requested vertex from %s, requested from %s",
				peerID, vdr)
		case !vtxID.Equals(vtxID0):
			t.Fatalf("should have requested vtx0")
		}
		*requestID = reqID
		reqVtxID = vtxID
	}

	manager.ParseVertexF = func(vtxBytes []byte) (avalanche.Vertex, error) {
		switch {
		case bytes.Equal(vtxBytes, vtxBytes0):
			vtx0.StatusV = choices.Processing
			return vtx0, nil
		case bytes.Equal(vtxBytes, vtxBytes1):
			vtx1.StatusV = choices.Processing
			return vtx1, nil
		case bytes.Equal(vtxBytes, vtxBytes2):
			vtx2.StatusV = choices.Processing
			return vtx2, nil
		}
		t.Fatal(errParsedUnknownVertex)
		return nil, errParsedUnknownVertex
	}
	vm.CantBootstrapping = false

	if err := bs.ForceAccepted(acceptedIDs); err != nil { // should request vtx0
		t.Fatal(err)
	} else if !reqVtxID.Equals(vtxID0) {
		t.Fatalf("should have requested vtxID0 but requested %s", reqVtxID)
	}

	oldReqID := *requestID
	if err := bs.MultiPut(peerID, *requestID, [][]byte{vtxBytes2}); err != nil { // send unexpected vertex
		t.Fatal(err)
	} else if *requestID == oldReqID {
		t.Fatal("should have issued new request")
	} else if !reqVtxID.Equals(vtxID0) {
		t.Fatalf("should have requested vtxID0 but requested %s", reqVtxID)
	}

	oldReqID = *requestID
	manager.GetVertexF = func(vtxID ids.ID) (avalanche.Vertex, error) {
		switch {
		case vtxID.Equals(vtxID1):
			return vtx1, nil
		case vtxID.Equals(vtxID0):
			return vtx0, nil
		default:
			t.Fatal(errUnknownVertex)
			panic(errUnknownVertex)
		}
	}

	vm.CantBootstrapped = false

	if err := bs.MultiPut(peerID, *requestID, [][]byte{vtxBytes0, vtxBytes2}); err != nil { // send expected vertex and vertex that should not be accepted
		t.Fatal(err)
	}

	switch {
	case *requestID != oldReqID:
		t.Fatal("should not have issued new request")
	case !*finished:
		t.Fatalf("Bootstrapping should have finished")
	case vtx0.Status() != choices.Accepted:
		t.Fatalf("Vertex should be accepted")
	case vtx1.Status() != choices.Accepted:
		t.Fatalf("Vertex should be accepted")
	}
	if vtx2.Status() == choices.Accepted {
		t.Fatalf("Vertex should not have been accepted")
	}
}

// Vertex has a dependency and tx has a dependency
func TestBootstrapperTxDependencies(t *testing.T) {
	config, peerID, sender, manager, vm := newConfig(t)

	utxos := []ids.ID{ids.GenerateTestID(), ids.GenerateTestID()}

	txID0 := ids.GenerateTestID()
	txID1 := ids.GenerateTestID()

	txBytes0 := []byte{0}
	txBytes1 := []byte{1}

	tx0 := &snowstorm.TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     txID0,
			StatusV: choices.Processing,
		},
		BytesV: txBytes0,
	}
	tx0.InputIDsV.Add(utxos[0])

	// Depends on tx0
	tx1 := &snowstorm.TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     txID1,
			StatusV: choices.Processing,
		},
		DependenciesV: []snowstorm.Tx{tx0},
		BytesV:        txBytes1,
	}
	tx1.InputIDsV.Add(utxos[1])

	vtxID0 := ids.GenerateTestID()
	vtxID1 := ids.GenerateTestID()

	vtxBytes0 := []byte{2}
	vtxBytes1 := []byte{3}
	vm.ParseTxF = func(b []byte) (snowstorm.Tx, error) {
		switch {
		case bytes.Equal(b, txBytes0):
			return tx0, nil
		case bytes.Equal(b, txBytes1):
			return tx1, nil
		default:
			return nil, errors.New("wrong tx")
		}
	}

	vtx0 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID0,
			StatusV: choices.Unknown,
		},
		HeightV: 0,
		TxsV:    []snowstorm.Tx{tx1},
		BytesV:  vtxBytes0,
	}
	vtx1 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID1,
			StatusV: choices.Processing,
		},
		ParentsV: []avalanche.Vertex{vtx0}, // Depends on vtx0
		HeightV:  1,
		TxsV:     []snowstorm.Tx{tx0},
		BytesV:   vtxBytes1,
	}

	bs := Bootstrapper{}
	finished := new(bool)
	err := bs.Initialize(
		config,
		func() error { *finished = true; return nil },
		fmt.Sprintf("%s_%s_bs", constants.PlatformName, config.Ctx.ChainID),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatal(err)
	}

	acceptedIDs := ids.Set{}
	acceptedIDs.Add(vtxID1)

	manager.ParseVertexF = func(vtxBytes []byte) (avalanche.Vertex, error) {
		switch {
		case bytes.Equal(vtxBytes, vtxBytes1):
			return vtx1, nil
		case bytes.Equal(vtxBytes, vtxBytes0):
			return vtx0, nil
		}
		t.Fatal(errParsedUnknownVertex)
		return nil, errParsedUnknownVertex
	}
	manager.GetVertexF = func(vtxID ids.ID) (avalanche.Vertex, error) {
		switch {
		case vtxID.Equals(vtxID1):
			return vtx1, nil
		case vtxID.Equals(vtxID0):
			return nil, errUnknownVertex
		default:
			t.Fatal(errUnknownVertex)
			panic(errUnknownVertex)
		}
	}

	reqIDPtr := new(uint32)
	sender.GetAncestorsF = func(vdr ids.ShortID, reqID uint32, vtxID ids.ID) {
		if !vdr.Equals(peerID) {
			t.Fatalf("Should have requested vertex from %s, requested from %s", peerID, vdr)
		}
		switch {
		case vtxID.Equals(vtxID0):
		default:
			t.Fatal(errUnknownVertex)
		}

		*reqIDPtr = reqID
	}

	vm.CantBootstrapping = false

	if err := bs.ForceAccepted(acceptedIDs); err != nil { // should request vtx0
		t.Fatal(err)
	}

	manager.ParseVertexF = func(vtxBytes []byte) (avalanche.Vertex, error) {
		switch {
		case bytes.Equal(vtxBytes, vtxBytes1):
			return vtx1, nil
		case bytes.Equal(vtxBytes, vtxBytes0):
			vtx0.StatusV = choices.Processing
			return vtx0, nil
		}
		t.Fatal(errParsedUnknownVertex)
		return nil, errParsedUnknownVertex
	}

	vm.CantBootstrapped = false

	if err := bs.MultiPut(peerID, *reqIDPtr, [][]byte{vtxBytes0}); err != nil {
		t.Fatal(err)
	}

	if !*finished {
		t.Fatalf("Should have finished bootstrapping")
	}
	if tx0.Status() != choices.Accepted {
		t.Fatalf("Tx should be accepted")
	}
	if tx1.Status() != choices.Accepted {
		t.Fatalf("Tx should be accepted")
	}

	if vtx0.Status() != choices.Accepted {
		t.Fatalf("Vertex should be accepted")
	}
	if vtx1.Status() != choices.Accepted {
		t.Fatalf("Vertex should be accepted")
	}
}

// Unfulfilled tx dependency
func TestBootstrapperMissingTxDependency(t *testing.T) {
	config, peerID, sender, manager, vm := newConfig(t)

	utxos := []ids.ID{ids.GenerateTestID(), ids.GenerateTestID()}

	txID0 := ids.GenerateTestID()
	txID1 := ids.GenerateTestID()

	txBytes1 := []byte{1}

	tx0 := &snowstorm.TestTx{TestDecidable: choices.TestDecidable{
		IDV:     txID0,
		StatusV: choices.Unknown,
	}}

	tx1 := &snowstorm.TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     txID1,
			StatusV: choices.Processing,
		},
		DependenciesV: []snowstorm.Tx{tx0},
		BytesV:        txBytes1,
	}
	tx1.InputIDsV.Add(utxos[1])

	vtxID0 := ids.GenerateTestID()
	vtxID1 := ids.GenerateTestID()

	vtxBytes0 := []byte{2}
	vtxBytes1 := []byte{3}

	vtx0 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID0,
			StatusV: choices.Unknown,
		},
		HeightV: 0,
		BytesV:  vtxBytes0,
	}
	vtx1 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID1,
			StatusV: choices.Processing,
		},
		ParentsV: []avalanche.Vertex{vtx0}, // depends on vtx0
		HeightV:  1,
		TxsV:     []snowstorm.Tx{tx1},
		BytesV:   vtxBytes1,
	}

	bs := Bootstrapper{}
	finished := new(bool)
	err := bs.Initialize(
		config,
		func() error { *finished = true; return nil },
		fmt.Sprintf("%s_%s_bs", constants.PlatformName, config.Ctx.ChainID),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatal(err)
	}

	acceptedIDs := ids.Set{}
	acceptedIDs.Add(vtxID1)

	manager.GetVertexF = func(vtxID ids.ID) (avalanche.Vertex, error) {
		switch {
		case vtxID.Equals(vtxID1):
			return vtx1, nil
		case vtxID.Equals(vtxID0):
			return nil, errUnknownVertex
		default:
			t.Fatal(errUnknownVertex)
			panic(errUnknownVertex)
		}
	}
	manager.ParseVertexF = func(vtxBytes []byte) (avalanche.Vertex, error) {
		switch {
		case bytes.Equal(vtxBytes, vtxBytes1):
			return vtx1, nil
		case bytes.Equal(vtxBytes, vtxBytes0):
			vtx0.StatusV = choices.Processing
			return vtx0, nil
		}
		t.Fatal(errParsedUnknownVertex)
		return nil, errParsedUnknownVertex
	}

	reqIDPtr := new(uint32)
	sender.GetAncestorsF = func(vdr ids.ShortID, reqID uint32, vtxID ids.ID) {
		if !vdr.Equals(peerID) {
			t.Fatalf("Should have requested vertex from %s, requested from %s", peerID, vdr)
		}
		switch {
		case vtxID.Equals(vtxID0):
		default:
			t.Fatalf("Requested wrong vertex")
		}

		*reqIDPtr = reqID
	}

	vm.CantBootstrapping = false

	if err := bs.ForceAccepted(acceptedIDs); err != nil { // should request vtx1
		t.Fatal(err)
	}

	vm.CantBootstrapped = false

	if err := bs.MultiPut(peerID, *reqIDPtr, [][]byte{vtxBytes0}); err != nil {
		t.Fatal(err)
	}

	if !*finished {
		t.Fatalf("Bootstrapping should have finished")
	}
	if tx0.Status() != choices.Unknown { // never saw this tx
		t.Fatalf("Tx should be unknown")
	}
	if tx1.Status() != choices.Processing { // can't accept because we don't have tx0
		t.Fatalf("Tx should be processing")
	}

	if vtx0.Status() != choices.Accepted {
		t.Fatalf("Vertex should be accepted")
	}
	if vtx1.Status() != choices.Processing { // can't accept because we don't have tx1 accepted
		t.Fatalf("Vertex should be processing")
	}
}

func TestBootstrapperAcceptedFrontier(t *testing.T) {
	config, _, _, manager, _ := newConfig(t)

	vtxID0 := ids.GenerateTestID()
	vtxID1 := ids.GenerateTestID()
	vtxID2 := ids.GenerateTestID()

	bs := Bootstrapper{}
	err := bs.Initialize(
		config,
		nil,
		fmt.Sprintf("%s_%s_bs", constants.PlatformName, config.Ctx.ChainID),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatal(err)
	}

	manager.EdgeF = func() []ids.ID {
		return []ids.ID{
			vtxID0,
			vtxID1,
		}
	}

	accepted := bs.CurrentAcceptedFrontier()

	manager.EdgeF = nil

	if !accepted.Contains(vtxID0) {
		t.Fatalf("Vtx should be accepted")
	}
	if !accepted.Contains(vtxID1) {
		t.Fatalf("Vtx should be accepted")
	}
	if accepted.Contains(vtxID2) {
		t.Fatalf("Vtx shouldn't be accepted")
	}
}

func TestBootstrapperFilterAccepted(t *testing.T) {
	config, _, _, manager, _ := newConfig(t)

	vtxID0 := ids.GenerateTestID()
	vtxID1 := ids.GenerateTestID()
	vtxID2 := ids.GenerateTestID()

	vtx0 := &avalanche.TestVertex{TestDecidable: choices.TestDecidable{
		IDV:     vtxID0,
		StatusV: choices.Accepted,
	}}
	vtx1 := &avalanche.TestVertex{TestDecidable: choices.TestDecidable{
		IDV:     vtxID1,
		StatusV: choices.Accepted,
	}}

	bs := Bootstrapper{}
	finished := new(bool)
	err := bs.Initialize(
		config,
		func() error { *finished = true; return nil },
		fmt.Sprintf("%s_%s_bs", constants.PlatformName, config.Ctx.ChainID),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatal(err)
	}

	vtxIDs := ids.Set{}
	vtxIDs.Add(
		vtxID0,
		vtxID1,
		vtxID2,
	)

	manager.GetVertexF = func(vtxID ids.ID) (avalanche.Vertex, error) {
		switch {
		case vtxID.Equals(vtxID0):
			return vtx0, nil
		case vtxID.Equals(vtxID1):
			return vtx1, nil
		case vtxID.Equals(vtxID2):
			return nil, errUnknownVertex
		}
		t.Fatal(errUnknownVertex)
		return nil, errUnknownVertex
	}

	accepted := bs.FilterAccepted(vtxIDs)

	manager.GetVertexF = nil

	if !accepted.Contains(vtxID0) {
		t.Fatalf("Vtx should be accepted")
	}
	if !accepted.Contains(vtxID1) {
		t.Fatalf("Vtx should be accepted")
	}
	if accepted.Contains(vtxID2) {
		t.Fatalf("Vtx shouldn't be accepted")
	}
}

// MultiPut only contains 1 of the two needed vertices; have to issue another GetAncestors
func TestBootstrapperIncompleteMultiPut(t *testing.T) {
	config, peerID, sender, manager, vm := newConfig(t)

	vtxID0 := ids.Empty.Prefix(0)
	vtxID1 := ids.Empty.Prefix(1)
	vtxID2 := ids.Empty.Prefix(2)

	vtxBytes0 := []byte{0}
	vtxBytes1 := []byte{1}
	vtxBytes2 := []byte{2}

	vtx0 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID0,
			StatusV: choices.Unknown,
		},
		HeightV: 0,
		BytesV:  vtxBytes0,
	}
	vtx1 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID1,
			StatusV: choices.Unknown,
		},
		ParentsV: []avalanche.Vertex{vtx0},
		HeightV:  1,
		BytesV:   vtxBytes1,
	}
	vtx2 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID2,
			StatusV: choices.Processing,
		},
		ParentsV: []avalanche.Vertex{vtx1},
		HeightV:  2,
		BytesV:   vtxBytes2,
	}

	bs := Bootstrapper{}
	finished := new(bool)
	err := bs.Initialize(
		config,
		func() error { *finished = true; return nil },
		fmt.Sprintf("%s_%s_bs", constants.PlatformName, config.Ctx.ChainID),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatal(err)
	}

	acceptedIDs := ids.Set{}
	acceptedIDs.Add(vtxID2)

	manager.GetVertexF = func(vtxID ids.ID) (avalanche.Vertex, error) {
		switch {
		case vtxID.Equals(vtxID0):
			return nil, errUnknownVertex
		case vtxID.Equals(vtxID1):
			return nil, errUnknownVertex
		case vtxID.Equals(vtxID2):
			return vtx2, nil
		default:
			t.Fatal(errUnknownVertex)
			panic(errUnknownVertex)
		}
	}
	manager.ParseVertexF = func(vtxBytes []byte) (avalanche.Vertex, error) {
		switch {
		case bytes.Equal(vtxBytes, vtxBytes0):
			vtx0.StatusV = choices.Processing
			return vtx0, nil

		case bytes.Equal(vtxBytes, vtxBytes1):
			vtx1.StatusV = choices.Processing
			return vtx1, nil
		case bytes.Equal(vtxBytes, vtxBytes2):
			return vtx2, nil
		}
		t.Fatal(errParsedUnknownVertex)
		return nil, errParsedUnknownVertex
	}
	reqIDPtr := new(uint32)
	requested := ids.Empty
	sender.GetAncestorsF = func(vdr ids.ShortID, reqID uint32, vtxID ids.ID) {
		if !vdr.Equals(peerID) {
			t.Fatalf("Should have requested vertex from %s, requested from %s", peerID, vdr)
		}
		switch {
		case vtxID.Equals(vtxID1), vtxID.Equals(vtxID0):
		default:
			t.Fatal(errUnknownVertex)
		}
		*reqIDPtr = reqID
		requested = vtxID
	}

	vm.CantBootstrapping = false

	if err := bs.ForceAccepted(acceptedIDs); err != nil { // should request vtx1
		t.Fatal(err)
	} else if !requested.Equals(vtxID1) {
		t.Fatal("requested wrong vtx")
	}

	if err := bs.MultiPut(peerID, *reqIDPtr, [][]byte{vtxBytes1}); err != nil { // Provide vtx1; should request vtx0
		t.Fatal(err)
	} else if bs.Ctx.IsBootstrapped() {
		t.Fatalf("should not have finished")
	} else if !requested.Equals(vtxID0) {
		t.Fatal("should hae requested vtx0")
	}

	vm.CantBootstrapped = false

	if err := bs.MultiPut(peerID, *reqIDPtr, [][]byte{vtxBytes0}); err != nil { // Provide vtx0; can finish now
		t.Fatal(err)
	} else if !bs.Ctx.IsBootstrapped() {
		t.Fatal("should have finished")
	} else if vtx0.Status() != choices.Accepted {
		t.Fatal("should be accepted")
	} else if vtx1.Status() != choices.Accepted {
		t.Fatal("should be accepted")
	} else if vtx2.Status() != choices.Accepted {
		t.Fatal("should be accepted")
	}
}

func TestBootstrapperFinalized(t *testing.T) {
	config, peerID, sender, manager, vm := newConfig(t)

	vtxID0 := ids.Empty.Prefix(0)
	vtxID1 := ids.Empty.Prefix(1)

	vtxBytes0 := []byte{0}
	vtxBytes1 := []byte{1}

	vtx0 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID0,
			StatusV: choices.Unknown,
		},
		HeightV: 0,
		BytesV:  vtxBytes0,
	}
	vtx1 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID1,
			StatusV: choices.Unknown,
		},
		ParentsV: []avalanche.Vertex{vtx0},
		HeightV:  1,
		BytesV:   vtxBytes1,
	}

	bs := Bootstrapper{}
	finished := new(bool)
	err := bs.Initialize(
		config,
		func() error { *finished = true; return nil },
		fmt.Sprintf("%s_%s_bs", constants.PlatformName, config.Ctx.ChainID),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatal(err)
	}

	acceptedIDs := ids.Set{}
	acceptedIDs.Add(vtxID0)
	acceptedIDs.Add(vtxID1)

	parsedVtx0 := false
	parsedVtx1 := false
	manager.GetVertexF = func(vtxID ids.ID) (avalanche.Vertex, error) {
		switch {
		case vtxID.Equals(vtxID0):
			if parsedVtx0 {
				return vtx0, nil
			}
			return nil, errUnknownVertex
		case vtxID.Equals(vtxID1):
			if parsedVtx1 {
				return vtx1, nil
			}
			return nil, errUnknownVertex
		default:
			t.Fatal(errUnknownVertex)
			panic(errUnknownVertex)
		}
	}
	manager.ParseVertexF = func(vtxBytes []byte) (avalanche.Vertex, error) {
		switch {
		case bytes.Equal(vtxBytes, vtxBytes0):
			vtx0.StatusV = choices.Processing
			parsedVtx0 = true
			return vtx0, nil
		case bytes.Equal(vtxBytes, vtxBytes1):
			vtx1.StatusV = choices.Processing
			parsedVtx1 = true
			return vtx1, nil
		}
		t.Fatal(errUnknownVertex)
		return nil, errUnknownVertex
	}

	requestIDs := map[[32]byte]uint32{}
	sender.GetAncestorsF = func(vdr ids.ShortID, reqID uint32, vtxID ids.ID) {
		if !vdr.Equals(peerID) {
			t.Fatalf("Should have requested block from %s, requested from %s", peerID, vdr)
		}
		requestIDs[vtxID.Key()] = reqID
	}

	vm.CantBootstrapping = false

	if err := bs.ForceAccepted(acceptedIDs); err != nil { // should request vtx0 and vtx1
		t.Fatal(err)
	}

	reqID, ok := requestIDs[vtxID1.Key()]
	if !ok {
		t.Fatalf("should have requested vtx1")
	}

	vm.CantBootstrapped = false

	if err := bs.MultiPut(peerID, reqID, [][]byte{vtxBytes1, vtxBytes0}); err != nil {
		t.Fatal(err)
	}

	reqID, ok = requestIDs[vtxID0.Key()]
	if !ok {
		t.Fatalf("should have requested vtx0")
	}

	if err := bs.GetAncestorsFailed(peerID, reqID); err != nil {
		t.Fatal(err)
	}

	if !*finished {
		t.Fatalf("Bootstrapping should have finished")
	} else if vtx0.Status() != choices.Accepted {
		t.Fatalf("Vertex should be accepted")
	} else if vtx1.Status() != choices.Accepted {
		t.Fatalf("Vertex should be accepted")
	}
}

// Test that MultiPut accepts the parents of the first vertex returned
func TestBootstrapperAcceptsMultiPutParents(t *testing.T) {
	config, peerID, sender, manager, vm := newConfig(t)

	vtxID0 := ids.Empty.Prefix(0)
	vtxID1 := ids.Empty.Prefix(1)
	vtxID2 := ids.Empty.Prefix(2)

	vtxBytes0 := []byte{0}
	vtxBytes1 := []byte{1}
	vtxBytes2 := []byte{2}

	vtx0 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID0,
			StatusV: choices.Unknown,
		},
		HeightV: 0,
		BytesV:  vtxBytes0,
	}
	vtx1 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID1,
			StatusV: choices.Unknown,
		},
		ParentsV: []avalanche.Vertex{vtx0},
		HeightV:  1,
		BytesV:   vtxBytes1,
	}
	vtx2 := &avalanche.TestVertex{
		TestDecidable: choices.TestDecidable{
			IDV:     vtxID2,
			StatusV: choices.Unknown,
		},
		ParentsV: []avalanche.Vertex{vtx1},
		HeightV:  2,
		BytesV:   vtxBytes2,
	}

	bs := Bootstrapper{}
	finished := new(bool)
	err := bs.Initialize(
		config,
		func() error { *finished = true; return nil },
		fmt.Sprintf("%s_%s_bs", constants.PlatformName, config.Ctx.ChainID),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatal(err)
	}

	acceptedIDs := ids.Set{}
	acceptedIDs.Add(vtxID2)

	parsedVtx0 := false
	parsedVtx1 := false
	parsedVtx2 := false
	manager.GetVertexF = func(vtxID ids.ID) (avalanche.Vertex, error) {
		switch {
		case vtxID.Equals(vtxID0):
			if parsedVtx0 {
				return vtx0, nil
			}
			return nil, errUnknownVertex
		case vtxID.Equals(vtxID1):
			if parsedVtx1 {
				return vtx1, nil
			}
			return nil, errUnknownVertex
		case vtxID.Equals(vtxID2):
			if parsedVtx2 {
				return vtx2, nil
			}
		default:
			t.Fatal(errUnknownVertex)
			panic(errUnknownVertex)
		}
		return nil, errUnknownVertex
	}
	manager.ParseVertexF = func(vtxBytes []byte) (avalanche.Vertex, error) {
		switch {
		case bytes.Equal(vtxBytes, vtxBytes0):
			vtx0.StatusV = choices.Processing
			parsedVtx0 = true
			return vtx0, nil
		case bytes.Equal(vtxBytes, vtxBytes1):
			vtx1.StatusV = choices.Processing
			parsedVtx1 = true
			return vtx1, nil
		case bytes.Equal(vtxBytes, vtxBytes2):
			vtx2.StatusV = choices.Processing
			parsedVtx2 = true
			return vtx2, nil
		}
		t.Fatal(errUnknownVertex)
		return nil, errUnknownVertex
	}

	requestIDs := map[[32]byte]uint32{}
	sender.GetAncestorsF = func(vdr ids.ShortID, reqID uint32, vtxID ids.ID) {
		if !vdr.Equals(peerID) {
			t.Fatalf("Should have requested block from %s, requested from %s", peerID, vdr)
		}
		requestIDs[vtxID.Key()] = reqID
	}

	vm.CantBootstrapping = false

	if err := bs.ForceAccepted(acceptedIDs); err != nil { // should request vtx2
		t.Fatal(err)
	}

	reqID, ok := requestIDs[vtxID2.Key()]
	if !ok {
		t.Fatalf("should have requested vtx2")
	}

	vm.CantBootstrapped = false

	if err := bs.MultiPut(peerID, reqID, [][]byte{vtxBytes2, vtxBytes1, vtxBytes0}); err != nil {
		t.Fatal(err)
	}

	if !*finished {
		t.Fatalf("Bootstrapping should have finished")
	} else if vtx0.Status() != choices.Accepted {
		t.Fatalf("Vertex should be accepted")
	} else if vtx1.Status() != choices.Accepted {
		t.Fatalf("Vertex should be accepted")
	} else if vtx2.Status() != choices.Accepted {
		t.Fatalf("Vertex should be accepted")
	}
}
