// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ids

import (
	"fmt"
	"strings"
)

const (
	minUniqueBagSize = 16
)

// UniqueBag ...
type UniqueBag map[[32]byte]BitSet

func (b *UniqueBag) init() {
	if *b == nil {
		*b = make(map[[32]byte]BitSet, minUniqueBagSize)
	}
}

// Add ...
func (b *UniqueBag) Add(setID uint, idSet ...ID) {
	bs := BitSet(0)
	bs.Add(setID)

	for _, id := range idSet {
		b.UnionSet(id, bs)
	}
}

// UnionSet ...
func (b *UniqueBag) UnionSet(id ID, set BitSet) {
	b.init()

	key := id.Key()
	previousSet := (*b)[key]
	previousSet.Union(set)
	(*b)[key] = previousSet
}

// DifferenceSet ...
func (b *UniqueBag) DifferenceSet(id ID, set BitSet) {
	b.init()

	key := id.Key()
	previousSet := (*b)[key]
	previousSet.Difference(set)
	(*b)[key] = previousSet
}

// Difference ...
func (b *UniqueBag) Difference(diff *UniqueBag) {
	b.init()

	for key, previousSet := range *b {
		if previousSetDiff, exists := (*diff)[key]; exists {
			previousSet.Difference(previousSetDiff)
		}
		(*b)[key] = previousSet
	}
}

// GetSet ...
func (b *UniqueBag) GetSet(id ID) BitSet { return (*b)[*id.ID] }

// RemoveSet ...
func (b *UniqueBag) RemoveSet(id ID) { delete(*b, id.Key()) }

// List ...
func (b *UniqueBag) List() []ID {
	idList := make([]ID, len(*b))
	i := 0
	for id := range *b {
		idList[i] = NewID(id)
		i++
	}
	return idList
}

// Bag ...
func (b *UniqueBag) Bag(alpha int) Bag {
	bag := Bag{}
	bag.SetThreshold(alpha)
	for id, bs := range *b {
		bag.AddCount(NewID(id), bs.Len())
	}
	return bag
}

func (b *UniqueBag) String() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("UniqueBag: (Size = %d)", len(*b)))
	for idBytes, set := range *b {
		id := NewID(idBytes)
		sb.WriteString(fmt.Sprintf("\n    ID[%s]: Members = %s", id, set))
	}

	return sb.String()
}
