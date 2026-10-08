package tts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// CacheKey identifies a cached TTS audio entry. Pre-existing CacheKey in
// service.go has the same shape and is kept here as the single canonical
// definition.
//
// (CacheKey was previously defined in service.go; the move from package main
// consolidates it here alongside the cache implementation.)

// cacheKeyHash returns a deterministic string hash for map storage.
func cacheKeyHash(k CacheKey) string {
	raw := fmt.Sprintf("%s|%s|%.4f|%s|%d", k.EventID, k.Voice, k.Speed, k.Version, k.ChunkIndex)
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
	clk      schedule.Clock
	diskDir  string
	maxAge   time.Duration
}

type diskCacheMetadata struct {
	EventID     string    `json:"event_id"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewCache creates a cache with the given maximum size in bytes using
// the system schedule.
func NewCache(maxSizeBytes int) *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSizeBytes,
		clk:     schedule.System(),
	}
}

func (c *Cache) now() time.Time {
	if c.clk == nil {
		return schedule.System().Now()
	}
	return c.clk.Now()
}

// NewCacheWithClock is the clock-injected constructor for deterministic
// CreatedAt stamping in tests.
func NewCacheWithClock(maxSizeBytes int, clk schedule.Clock) *Cache {
	if clk == nil {
		clk = schedule.System()
	}
	return &Cache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSizeBytes,
		clk:     clk,
	}
}

// NewPersistentCache adds a bounded disk tier beneath the in-memory LRU. Disk
// entries are keyed by the same content hash as memory entries, so a process
// restart can replay synthesized paragraphs without synthesizing them again.
// A non-positive maxAge disables age eviction while retaining the size bound.
func NewPersistentCache(maxSizeBytes int, dir string, maxAge time.Duration) (*Cache, error) {
	c := NewCache(maxSizeBytes)
	c.diskDir = dir
	c.maxAge = maxAge
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create TTS cache directory: %w", err)
	}
	if err := c.loadDisk(); err != nil {
		return nil, err
	}
	return c, nil
}

// Get retrieves a cached entry. Returns nil, false on miss.
func (c *Cache) Get(key CacheKey) (*CacheEntry, bool) {
	hash := cacheKeyHash(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[hash]; ok {
		return entry, true
	}
	return c.loadDiskEntryLocked(hash)
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
		CreatedAt:   c.now(),
		eventID:     key.EventID,
	}
	c.order = append(c.order, hash)
	c.currSize += size
	c.persistDiskLocked(hash, c.entries[hash])
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
		c.removeDiskLocked(hash)
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
	c.removeDiskLocked(oldest)
}

func (c *Cache) removeFromOrder(hash string) {
	for i, h := range c.order {
		if h == hash {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}

func (c *Cache) diskPaths(hash string) (string, string) {
	return filepath.Join(c.diskDir, hash+".audio"), filepath.Join(c.diskDir, hash+".json")
}

func (c *Cache) persistDiskLocked(hash string, entry *CacheEntry) {
	if c.diskDir == "" {
		return
	}
	audioPath, metadataPath := c.diskPaths(hash)
	metadata, err := json.Marshal(diskCacheMetadata{EventID: entry.eventID, ContentType: entry.ContentType, CreatedAt: entry.CreatedAt})
	if err != nil {
		return
	}
	if err := os.WriteFile(audioPath, entry.Audio, 0o600); err != nil {
		return
	}
	_ = os.WriteFile(metadataPath, metadata, 0o600)
}

func (c *Cache) removeDiskLocked(hash string) {
	if c.diskDir == "" {
		return
	}
	audioPath, metadataPath := c.diskPaths(hash)
	_ = os.Remove(audioPath)
	_ = os.Remove(metadataPath)
}

func (c *Cache) loadDiskEntryLocked(hash string) (*CacheEntry, bool) {
	if c.diskDir == "" {
		return nil, false
	}
	audioPath, metadataPath := c.diskPaths(hash)
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, false
	}
	var metadata diskCacheMetadata
	if json.Unmarshal(metadataBytes, &metadata) != nil || (c.maxAge > 0 && c.now().Sub(metadata.CreatedAt) > c.maxAge) {
		c.removeDiskLocked(hash)
		return nil, false
	}
	audio, err := os.ReadFile(audioPath)
	if err != nil || len(audio) > c.maxSize {
		c.removeDiskLocked(hash)
		return nil, false
	}
	entry := &CacheEntry{Audio: audio, ContentType: metadata.ContentType, CreatedAt: metadata.CreatedAt, eventID: metadata.EventID}
	for c.currSize+len(audio) > c.maxSize && len(c.order) > 0 {
		c.evictOldest()
	}
	c.entries[hash] = entry
	c.order = append(c.order, hash)
	c.currSize += len(audio)
	return entry, true
}

func (c *Cache) loadDisk() error {
	entries, err := os.ReadDir(c.diskDir)
	if err != nil {
		return fmt.Errorf("read TTS cache directory: %w", err)
	}
	type diskEntry struct {
		hash string
		when time.Time
	}
	var candidates []diskEntry
	for _, item := range entries {
		if item.IsDir() || filepath.Ext(item.Name()) != ".json" {
			continue
		}
		hash := item.Name()[:len(item.Name())-len(filepath.Ext(item.Name()))]
		data, readErr := os.ReadFile(filepath.Join(c.diskDir, item.Name()))
		var metadata diskCacheMetadata
		if readErr != nil || json.Unmarshal(data, &metadata) != nil {
			c.removeDiskLocked(hash)
			continue
		}
		candidates = append(candidates, diskEntry{hash: hash, when: metadata.CreatedAt})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].when.Before(candidates[j].when) })
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, candidate := range candidates {
		if _, ok := c.loadDiskEntryLocked(candidate.hash); !ok {
			continue
		}
	}
	return nil
}
