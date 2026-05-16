package tts

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// CacheKey identifies a cached TTS audio entry. Pre-existing CacheKey in
// service.go has the same shape and is kept here as the single canonical
// definition.
//
// (CacheKey was previously defined in service.go; the move from package main
// consolidates it here alongside the cache implementation.)

// cacheKeyHash returns a deterministic string hash for map storage.
func cacheKeyHash(k CacheKey) string {
	raw := fmt.Sprintf("%s|%s|%.4f|%s", k.EventID, k.Voice, k.Speed, k.Version)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum[:16])
}

// CacheEntry holds cached audio data.
type CacheEntry struct {
	Audio       []byte
	ContentType string
	CreatedAt   time.Time
	eventID     string // for reverse lookup during eviction
}

// CacheStats provides observability into cache state.
type CacheStats struct {
	EntryCount int
	TotalBytes int
	MaxBytes   int
}

// Cache is an in-memory LRU audio cache for pre-synthesized TTS.
type Cache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	order    []string // LRU order: oldest first
	maxSize  int
	currSize int
}

// NewCache creates a cache with the given maximum size in bytes.
func NewCache(maxSizeBytes int) *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSizeBytes,
	}
}

// Get retrieves a cached entry. Returns nil, false on miss.
func (c *Cache) Get(key CacheKey) (*CacheEntry, bool) {
	hash := cacheKeyHash(key)
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[hash]
	return entry, ok
}

// Put stores audio in the cache, evicting oldest entries if necessary.
func (c *Cache) Put(key CacheKey, audio []byte, contentType string) {
	hash := cacheKeyHash(key)
	size := len(audio)

	c.mu.Lock()
	defer c.mu.Unlock()

	// If this single entry exceeds max size, don't cache it.
	if size > c.maxSize {
		return
	}

	// Remove existing entry with this key if present (update case).
	if existing, ok := c.entries[hash]; ok {
		c.currSize -= len(existing.Audio)
		delete(c.entries, hash)
		c.removeFromOrder(hash)
	}

	// Evict oldest entries until there's room.
	for c.currSize+size > c.maxSize && len(c.order) > 0 {
		c.evictOldest()
	}

	c.entries[hash] = &CacheEntry{
		Audio:       audio,
		ContentType: contentType,
		CreatedAt:   time.Now(),
		eventID:     key.EventID,
	}
	c.order = append(c.order, hash)
	c.currSize += size
}

// Evict removes all cached entries for a given event ID (all versions/voices).
func (c *Cache) Evict(eventID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toRemove []string
	for hash, entry := range c.entries {
		if entry.eventID == eventID {
			toRemove = append(toRemove, hash)
		}
	}
	for _, hash := range toRemove {
		if entry, ok := c.entries[hash]; ok {
			c.currSize -= len(entry.Audio)
			delete(c.entries, hash)
			c.removeFromOrder(hash)
		}
	}
}

// Stats returns current cache statistics.
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return CacheStats{
		EntryCount: len(c.entries),
		TotalBytes: c.currSize,
		MaxBytes:   c.maxSize,
	}
}

func (c *Cache) evictOldest() {
	if len(c.order) == 0 {
		return
	}
	oldest := c.order[0]
	c.order = c.order[1:]
	if entry, ok := c.entries[oldest]; ok {
		c.currSize -= len(entry.Audio)
		delete(c.entries, oldest)
	}
}

func (c *Cache) removeFromOrder(hash string) {
	for i, h := range c.order {
		if h == hash {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}
