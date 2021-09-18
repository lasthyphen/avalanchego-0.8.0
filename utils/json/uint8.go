// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package json

import (
	"errors"
	"math"
	"strconv"
)

var (
	errTooLarge8 = errors.New("value overflowed uint8")
)

// Uint8 ...
type Uint8 uint8

// MarshalJSON ...
func (u Uint8) MarshalJSON() ([]byte, error) {
	return []byte("\"" + strconv.FormatUint(uint64(u), 10) + "\""), nil
}

// UnmarshalJSON ...
func (u *Uint8) UnmarshalJSON(b []byte) error {
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
	if val > math.MaxUint8 {
		return errTooLarge8
	}
	*u = Uint8(val)
	return err
}
