// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package sampler

import (
	"math"
	"math/rand"
	"time"
)

func init() { rand.Seed(time.Now().UnixNano()) }

type defaultMap map[uint64]uint64

func (m defaultMap) get(key uint64, defaultVal uint64) uint64 {
	if val, ok := m[key]; ok {
		return val
	}
	return defaultVal
}

// uniformReplacer allows for sampling over a uniform distribution without
// replacement.
//
// Sampling is performed by lazily performing an array shuffle of the array
// [0, 1, ..., length - 1]. By performing the first count swaps of this shuffle,
// we can create an array of length count with elements sampled with uniform
// probability.
//
// Initialization takes O(1) time.
//
// Sampling is performed in O(count) time and O(count) space.
type uniformReplacer struct {
	length uint64
}

func (s *uniformReplacer) Initialize(length uint64) error {
	if length > math.MaxInt64 {
		return errOutOfRange
	}
	s.length = length
	return nil
}

func (s *uniformReplacer) Sample(count int) ([]uint64, error) {
	if count < 0 || s.length < uint64(count) {
		return nil, errOutOfRange
	}

	drawn := make(defaultMap, count)
	results := make([]uint64, count)
	for i := 0; i < count; i++ {
		// We don't use a cryptographically secure source of randomness here, as
		// there's no need to ensure a truly random sampling.
		draw := uint64(rand.Int63n(int64(s.length-uint64(i)))) + uint64(i)

		ret := drawn.get(draw, draw)
		drawn[draw] = drawn.get(uint64(i), uint64(i))

		results[i] = ret
	}

	return results, nil
}
