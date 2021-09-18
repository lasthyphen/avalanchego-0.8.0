// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ids

import (
	"fmt"
	"sync"
)

// Aliaser allows one to give an ID aliases and lookup the aliases given to an
// ID. An ID can have arbitrarily many aliases; two IDs may not have the same
// alias.
type Aliaser struct {
	lock    sync.RWMutex
	dealias map[string]ID
	aliases map[[32]byte][]string
}

// Initialize the aliaser to have no aliases
func (a *Aliaser) Initialize() {
	a.dealias = make(map[string]ID)
	a.aliases = make(map[[32]byte][]string)
}

// Lookup returns the ID associated with alias
func (a *Aliaser) Lookup(alias string) (ID, error) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	if ID, ok := a.dealias[alias]; ok {
		return ID, nil
	}
	return ID{}, fmt.Errorf("there is no ID with alias %s", alias)
}

// Aliases returns the aliases of an ID
func (a *Aliaser) Aliases(id ID) []string {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.aliases[id.Key()]
}

// PrimaryAlias returns the first alias of [id]
func (a *Aliaser) PrimaryAlias(id ID) (string, error) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	aliases, exists := a.aliases[id.Key()]
	if !exists || len(aliases) == 0 {
		return "", fmt.Errorf("there is no alias for ID %s", id)
	}
	return aliases[0], nil
}

// Alias gives [id] the alias [alias]
func (a *Aliaser) Alias(id ID, alias string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if _, exists := a.dealias[alias]; exists {
		return fmt.Errorf("%s is already used as an alias for an ID", alias)
	}
	key := id.Key()

	a.dealias[alias] = id
	a.aliases[key] = append(a.aliases[key], alias)
	return nil
}

// RemoveAliases of the provided ID
func (a *Aliaser) RemoveAliases(id ID) {
	a.lock.Lock()
	defer a.lock.Unlock()

	key := id.Key()

	aliases := a.aliases[key]
	delete(a.aliases, key)
	for _, alias := range aliases {
		delete(a.dealias, alias)
	}
}
