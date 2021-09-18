// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package timer

import (
	"time"
)

// Useful latency buckets
var (
	MillisecondsBuckets = []float64{
		10,    // 10 ms is ~ instant
		100,   // 100 ms
		250,   // 250 ms
		500,   // 500 ms
		1000,  // 1 second
		1500,  // 1.5 seconds
		2000,  // 2 seconds
		3000,  // 3 seconds
		5000,  // 5 seconds
		10000, // 10 seconds
		// anything larger than 10 seconds will be bucketed together
	}
	NanosecondsBuckets = []float64{
		float64(100 * time.Nanosecond),
		float64(time.Microsecond),
		float64(10 * time.Microsecond),
		float64(100 * time.Microsecond),
		float64(time.Millisecond),
		float64(10 * time.Millisecond),
		float64(100 * time.Millisecond),
		float64(time.Second),
		// anything larger than a second will be bucketed together
	}
)
