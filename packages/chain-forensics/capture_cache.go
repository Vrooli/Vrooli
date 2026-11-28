package chainforensics

import (
	"sync"
	"time"
)

const defaultCaptureCacheMaxBytes = 64 << 20

// CaptureCache is a bounded, TTL-aware in-process cache for immutable public
// chain captures. The clone function is called on both writes and reads so a
// caller cannot mutate retained evidence through a shared slice or map.
//
// The cache deliberately does not decide finality or TTL policy. Each chain
// reader chooses an identity-complete key and an expiry appropriate to its
// source's reorganization and rate-limit behavior.
type CaptureCache[T any] struct {
	mu       sync.Mutex
	limit    int
	maxBytes int64
	size     func(T) int64
	bytes    int64
	clone    func(T) T
	sequence uint64
	entries  map[string]captureCacheEntry[T]
}

type captureCacheEntry[T any] struct {
	value   T
	expires time.Time
	used    uint64
	weight  int64
}

// NewCaptureCache creates a cache with a fixed maximum entry count. Limits
// below one are raised to one. A nil clone function uses identity cloning and
// is suitable only for immutable values without reference-backed fields.
func NewCaptureCache[T any](limit int, clone func(T) T) *CaptureCache[T] {
	return newCaptureCache(limit, 0, clone, nil)
}

// NewWeightedCaptureCache creates a cache bounded by both entry count and
// estimated retained bytes. Captures whose estimate exceeds maxBytes are
// returned to the caller normally but are not retained.
func NewWeightedCaptureCache[T any](limit int, maxBytes int64, clone func(T) T, size func(T) int64) *CaptureCache[T] {
	if maxBytes < 1 || size == nil {
		maxBytes = 1
		size = func(T) int64 { return 1 }
	}
	return newCaptureCache(limit, maxBytes, clone, size)
}

func newCaptureCache[T any](limit int, maxBytes int64, clone func(T) T, size func(T) int64) *CaptureCache[T] {
	if limit < 1 {
		limit = 1
	}
	if clone == nil {
		clone = func(value T) T { return value }
	}
	return &CaptureCache[T]{limit: limit, maxBytes: maxBytes, size: size, clone: clone, entries: make(map[string]captureCacheEntry[T], limit)}
}

// Get returns a clone of a non-expired entry. An entry whose expiry is exactly
// now is expired.
func (c *CaptureCache[T]) Get(key string, now time.Time) (T, bool) {
	var zero T
	if c == nil || key == "" {
		return zero, false
	}
	c.mu.Lock()
	entry, ok := c.entries[key]
	if !ok {
		c.mu.Unlock()
		return zero, false
	}
	if !now.Before(entry.expires) {
		c.deleteLocked(key, entry)
		c.mu.Unlock()
		return zero, false
	}
	c.sequence++
	entry.used = c.sequence
	c.entries[key] = entry
	c.mu.Unlock()
	return c.clone(entry.value), true
}

// Put stores a defensive copy until expires. Expired entries are discarded
// first; when full, the least-recently-used live entry is evicted.
func (c *CaptureCache[T]) Put(key string, value T, expires, now time.Time) {
	if c == nil || key == "" || !now.Before(expires) {
		return
	}
	stored := c.clone(value)
	weight := int64(1)
	if c.size != nil {
		weight = c.size(stored)
		if weight < 1 {
			weight = 1
		}
		if weight > c.maxBytes {
			return
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeExpiredLocked(now)
	if existing, exists := c.entries[key]; exists {
		c.bytes -= existing.weight
		delete(c.entries, key)
	}
	for len(c.entries) >= c.limit || (c.maxBytes > 0 && c.bytes+weight > c.maxBytes) {
		if !c.evictLeastRecentlyUsedLocked() {
			return
		}
	}
	c.sequence++
	c.entries[key] = captureCacheEntry[T]{value: stored, expires: expires, used: c.sequence, weight: weight}
	c.bytes += weight
}

// Len reports the number of live entries after pruning expired captures.
func (c *CaptureCache[T]) Len(now time.Time) int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeExpiredLocked(now)
	return len(c.entries)
}

// Bytes reports the cache's current estimated weight after pruning expired
// entries. Unweighted caches report entry-count weight.
func (c *CaptureCache[T]) Bytes(now time.Time) int64 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeExpiredLocked(now)
	return c.bytes
}

func (c *CaptureCache[T]) removeExpiredLocked(now time.Time) {
	for key, entry := range c.entries {
		if !now.Before(entry.expires) {
			c.deleteLocked(key, entry)
		}
	}
}

func (c *CaptureCache[T]) evictLeastRecentlyUsedLocked() bool {
	var oldestKey string
	var oldestUsed uint64
	for key, entry := range c.entries {
		if oldestKey == "" || entry.used < oldestUsed {
			oldestKey = key
			oldestUsed = entry.used
		}
	}
	if oldestKey != "" {
		c.deleteLocked(oldestKey, c.entries[oldestKey])
		return true
	}
	return false
}

func (c *CaptureCache[T]) deleteLocked(key string, entry captureCacheEntry[T]) {
	delete(c.entries, key)
	c.bytes -= entry.weight
}
