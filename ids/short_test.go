// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ids

import (
	"testing"
)

func TestShortString(t *testing.T) {
	id := NewShortID([20]byte{1})

	xPrefixedID := id.PrefixedString("X-")
	pPrefixedID := id.PrefixedString("P-")

	newID, err := ShortFromPrefixedString(xPrefixedID, "X-")
	if err != nil {
		t.Fatal(err)
	}
	if !newID.Equals(id) {
		t.Fatalf("ShortFromPrefixedString did not produce the identical ID")
	}

	_, err = ShortFromPrefixedString(pPrefixedID, "X-")
	if err == nil {
		t.Fatal("Using the incorrect prefix did not cause an error")
	}

	tooLongPrefix := "hasnfaurnourieurn3eiur3nriu3nri34iurni34unr3iunrasfounaeouern3ur"
	_, err = ShortFromPrefixedString(xPrefixedID, tooLongPrefix)
	if err == nil {
		t.Fatal("Using the incorrect prefix did not cause an error")
	}
}

func TestIsUniqueShortIDs(t *testing.T) {
	ids := []ShortID{}
	if IsUniqueShortIDs(ids) == false {
		t.Fatal("should be unique")
	}
	id1 := GenerateTestShortID()
	ids = append(ids, id1)
	if IsUniqueShortIDs(ids) == false {
		t.Fatal("should be unique")
	}
	ids = append(ids, GenerateTestShortID())
	if IsUniqueShortIDs(ids) == false {
		t.Fatal("should be unique")
	}
	ids = append(ids, id1)
	if IsUniqueShortIDs(ids) == true {
		t.Fatal("should not be unique")
	}
}
