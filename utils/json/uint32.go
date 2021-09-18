// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package json

import (
	"errors"
	"math"
	"strconv"
)

var (
	errTooLarge32 = errors.New("value overflowed uint32")
)

// Uint32 ...
type Uint32 uint32

// MarshalJSON ...
func (u Uint32) MarshalJSON() ([]byte, error) {
	return []byte("\"" + strconv.FormatUint(uint64(u), 10) + "\""), nil
}

// UnmarshalJSON ...
func (u *Uint32) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == "null" {
		return nil
	}
	if len(str) >= 2 {
		if lastIndex := len(str) - 1; str[0] == '"' && str[lastIndex] == '"' {
			str = str[1:lastIndex]
		}
	}
	val, err := strconv.ParseUint(str, 10, 0)
	if val > math.MaxUint32 {
		return errTooLarge32
	}
	*u = Uint32(val)
	return err
}
