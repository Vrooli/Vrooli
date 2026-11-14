package repocontract

import (
	"container/list"
	"crypto/sha256"
	"sync"
)

// Only immutable, successfully validated contracts are retained. Load still
// reads the current file on every call: deletion, read failures, invalid edits,
// same-size/mtime replacements, and repository-context changes stay visible.
// This caches parsing work, never filesystem freshness or root-marker checks.
var parsedContracts = newContractCache(8, 256<<10)

type (
	contractCacheEntry struct {
		digest   [sha256.Size]byte
		contract *Contract
	}
	contractCache struct {
		mu                       sync.Mutex
		capacity, maxSourceBytes int
		entries                  map[[sha256.Size]byte]*list.Element
		order                    list.List
	}
)

func newContractCache(capacity, maxSourceBytes int) *contractCache {
	return &contractCache{capacity: capacity, maxSourceBytes: maxSourceBytes, entries: make(map[[sha256.Size]byte]*list.Element)}
}

func (c *contractCache) load(data []byte, decode func() (*Contract, error)) (*Contract, error) {
	if c.capacity <= 0 || len(data) > c.maxSourceBytes {
		return decode()
	}
	digest := sha256.Sum256(data)
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[digest]; ok {
		c.order.MoveToFront(entry)
		return entry.Value.(contractCacheEntry).contract, nil
	}
	// Decode under the lock so concurrent loads of the same bytes do not repeat
	// semantic validation. File I/O and oversized documents stay outside it.
	contract, err := decode()
	if err != nil {
		return nil, err
	}
	entry := c.order.PushFront(contractCacheEntry{digest: digest, contract: contract})
	c.entries[digest] = entry
	if c.order.Len() > c.capacity {
		old := c.order.Back()
		delete(c.entries, old.Value.(contractCacheEntry).digest)
		c.order.Remove(old)
	}
	return contract, nil
}
