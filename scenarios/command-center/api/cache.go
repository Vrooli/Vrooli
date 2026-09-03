package main

import (
	"encoding/json"
	"sync"
	"time"
)

// Envelope wraps a cached upstream response with freshness metadata.
// Consumers use StalenessTS to decide whether to render a "stale" badge.
type Envelope struct {
	Data          json.RawMessage `json:"data"`
	CachedAt      time.Time       `json:"cached_at"`
	StalenessTS   *time.Time      `json:"staleness_ts,omitempty"`
	FromCache     bool            `json:"from_cache"`
	Source        string          `json:"source"`
	ObservationAt *time.Time      `json:"observation_at,omitempty"`
}

// cacheEntry holds an envelope plus a per-key TTL.
type cacheEntry struct {
	env Envelope
	ttl time.Duration
}

// Cache is a process-local map keyed by "source:path" strings.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

// NewCache returns a ready-to-use cache.
func NewCache() *Cache {
	return &Cache{entries: make(map[string]cacheEntry)}
}

// Get returns the current envelope (if any) plus whether it is still fresh.
// When fresh is false the caller should attempt a refetch; the envelope may
// still be used as a fallback when the upstream fails.
func (c *Cache) Get(key string) (Envelope, bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok {
		return Envelope{}, false, false
	}
	fresh := time.Since(e.env.CachedAt) <= e.ttl
	env := e.env
	env.FromCache = true
	return env, fresh, true
}

// Put stores a fresh envelope for the given key with the supplied TTL.
func (c *Cache) Put(key string, env Envelope, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	env.FromCache = false
	c.entries[key] = cacheEntry{env: env, ttl: ttl}
}

// MarkStale flips the staleness timestamp on the cached entry so subsequent
// reads can surface that the upstream is returning errors. The entry itself
// is preserved as a last-good fallback.
func (c *Cache) MarkStale(key string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return
	}
	t := now
	e.env.StalenessTS = &t
	c.entries[key] = e
}

// TTLFor returns the canonical TTL for a given upstream source.
func TTLFor(source UpstreamSource) time.Duration {
	switch source {
	case SourceSwarm:
		return 30 * time.Second
	case SourceVrooli:
		return 60 * time.Second
	case SourceLPBS:
		return 5 * time.Minute
	default:
		return 30 * time.Second
	}
}
