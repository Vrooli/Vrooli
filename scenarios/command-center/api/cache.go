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

// Cache keeps the newest raw response per source/path and qualified values per
// source/path/metric ID. Endpoint ordering and per-metric trust are separate:
// a schema-invalid response can advance the source snapshot without replacing
// a metric's last-good qualified value.
type Cache struct {
	mu        sync.RWMutex
	entries   map[string]cacheEntry
	endpoints map[string]Envelope
}

// NewCache returns a ready-to-use cache.
func NewCache() *Cache {
	return &Cache{entries: make(map[string]cacheEntry), endpoints: make(map[string]Envelope)}
}

// CommitEndpoint retains the newest timestamped raw response for one
// source/path. A slow request that finishes after a newer response uses the
// newer snapshot for every metric in its board projection.
func (c *Cache) CommitEndpoint(key string, env Envelope) Envelope {
	c.mu.Lock()
	defer c.mu.Unlock()
	if previous, ok := c.endpoints[key]; ok && previous.ObservationAt != nil &&
		(env.ObservationAt == nil || env.ObservationAt.Before(*previous.ObservationAt)) {
		previous.FromCache = true
		return previous
	}
	env.FromCache = false
	c.endpoints[key] = env
	return env
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

// Put stores a fresh envelope for the given key with the supplied TTL. A
// refresh may finish after a newer concurrent request; never let its older
// producer timestamp replace the last-good snapshot. A qualified cache entry
// also cannot be replaced by an untimestamped response.
func (c *Cache) Put(key string, env Envelope, ttl time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if previous, ok := c.entries[key]; ok && previous.env.ObservationAt != nil {
		if env.ObservationAt == nil || env.ObservationAt.Before(*previous.env.ObservationAt) {
			return false
		}
	}
	env.FromCache = false
	c.entries[key] = cacheEntry{env: env, ttl: ttl}
	return true
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
	case SourceOffer:
		return 5 * time.Minute
	case SourceDeploy:
		return 30 * time.Second
	default:
		return 30 * time.Second
	}
}
