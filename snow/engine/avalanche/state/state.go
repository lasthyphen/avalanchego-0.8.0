// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"github.com/ava-labs/avalanchego/cache"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/utils/wrappers"
)

type state struct {
	serializer *Serializer

	dbCache cache.Cacher
	db      database.Database
}

func (s *state) Vertex(id ids.ID) *innerVertex {
	if vtxIntf, found := s.dbCache.Get(id); found {
		vtx, _ := vtxIntf.(*innerVertex)
		return vtx
	}

	if b, err := s.db.Get(id.Bytes()); err == nil {
		// The key was in the database
		if vtx, err := s.serializer.parseVertex(b); err == nil {
			s.dbCache.Put(id, vtx) // Cache the element
			return vtx
		}
		s.serializer.ctx.Log.Error("Parsing failed on saved vertex.\nPrefixed key = %s\nBytes = %s",
			id,
			formatting.DumpBytes{Bytes: b})
	}

	s.dbCache.Put(id, nil) // Cache the miss
	return nil
}

// SetVertex persists the vertex to the database and returns an error if it
// fails to write to the db
func (s *state) SetVertex(id ids.ID, vtx *innerVertex) error {
	s.dbCache.Put(id, vtx)

	if vtx == nil {
		return s.db.Delete(id.Bytes())
	}

	return s.db.Put(id.Bytes(), vtx.bytes)
}

func (s *state) Status(id ids.ID) choices.Status {
	if statusIntf, found := s.dbCache.Get(id); found {
		status, _ := statusIntf.(choices.Status)
		return status
	}

	if b, err := s.db.Get(id.Bytes()); err == nil {
		// The key was in the database
		p := wrappers.Packer{Bytes: b}
		status := choices.Status(p.UnpackInt())
		if p.Offset == len(b) && !p.Errored() {
			s.dbCache.Put(id, status)
			return status
		}
		s.serializer.ctx.Log.Error("Parsing failed on saved status.\nPrefixed key = %s\nBytes = \n%s",
			id,
			formatting.DumpBytes{Bytes: b})
	}

	s.dbCache.Put(id, choices.Unknown)
	return choices.Unknown
}

// SetStatus sets the status of the vertex and returns an error if it fails to write to the db
func (s *state) SetStatus(id ids.ID, status choices.Status) error {
	s.dbCache.Put(id, status)

	if status == choices.Unknown {
		return s.db.Delete(id.Bytes())
	}

	p := wrappers.Packer{Bytes: make([]byte, 4)}

	p.PackInt(uint32(status))

	s.serializer.ctx.Log.AssertNoError(p.Err)
	s.serializer.ctx.Log.AssertTrue(p.Offset == len(p.Bytes), "Wrong offset after packing")

	return s.db.Put(id.Bytes(), p.Bytes)
}

func (s *state) Edge(id ids.ID) []ids.ID {
	if frontierIntf, found := s.dbCache.Get(id); found {
		frontier, _ := frontierIntf.([]ids.ID)
		return frontier
	}

	if b, err := s.db.Get(id.Bytes()); err == nil {
		p := wrappers.Packer{Bytes: b}

		frontier := []ids.ID{}
		for i := p.UnpackInt(); i > 0 && !p.Errored(); i-- {
			id, _ := ids.ToID(p.UnpackFixedBytes(hashing.HashLen))
			frontier = append(frontier, id)
		}

		if p.Offset == len(b) && !p.Errored() {
			s.dbCache.Put(id, frontier)
			return frontier
		}
		s.serializer.ctx.Log.Error("Parsing failed on saved ids.\nPrefixed key = %s\nBytes = %s",
			id,
			formatting.DumpBytes{Bytes: b})
	}

	s.dbCache.Put(id, nil) // Cache the miss
	return nil
}

// SetEdge sets the frontier and returns an error if it fails to write to the db
func (s *state) SetEdge(id ids.ID, frontier []ids.ID) error {
	s.dbCache.Put(id, frontier)

	if len(frontier) == 0 {
		return s.db.Delete(id.Bytes())
	}

	size := wrappers.IntLen + hashing.HashLen*len(frontier)
	p := wrappers.Packer{Bytes: make([]byte, size)}

	p.PackInt(uint32(len(frontier)))
	for _, id := range frontier {
		p.PackFixedBytes(id.Bytes())
	}

	s.serializer.ctx.Log.AssertNoError(p.Err)
	s.serializer.ctx.Log.AssertTrue(p.Offset == len(p.Bytes), "Wrong offset after packing")

	return s.db.Put(id.Bytes(), p.Bytes)
}
