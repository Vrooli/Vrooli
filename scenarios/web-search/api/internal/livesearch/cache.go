package livesearch

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"web-search/internal/clock"
)

// DefaultCacheTTL is the freshness window for a cached live-search result.
const DefaultCacheTTL = 5 * time.Minute

// cacheEntry is one stored result set with its expiry instant. Engine
// degradation rides the entry so a cached response honestly reports the
// engines as they were at fetch time.
type cacheEntry struct {
	results      []Result
	engineIssues []EngineIssue
	expiresAt    time.Time
}

// Cache is an in-memory TTL cache keyed by the normalized query+limit. It is
// the freshness/budget shield in front of SearXNG: a repeated query within the
// TTL serves from memory and never touches the upstream. The clock is injected
// so expiry is deterministic in tests.
type Cache struct {
	ttl   time.Duration
	clock clock.Clock

	mu      sync.Mutex
	entries map[string]cacheEntry
}

// NewCache constructs a TTL cache. A non-positive ttl falls back to
// DefaultCacheTTL; a nil clock falls back to clock.System.
func NewCache(ttl time.Duration, clk clock.Clock) *Cache {
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	if clk == nil {
		clk = clock.System{}
	}
	return &Cache{
		ttl:     ttl,
		clock:   clk,
		entries: make(map[string]cacheEntry),
	}
}

// cacheKey normalizes a query+limit into a stable map key. Whitespace is
// trimmed and case folded so "Anthropic" and " anthropic " hit the same entry.
func cacheKey(query string, limit int) string {
	return fmt.Sprintf("%s\x00%d", strings.ToLower(strings.TrimSpace(query)), limit)
}

// Get returns the cached results (and the engine issues recorded with them)
// for query+limit when a non-expired entry exists. ok=false on a miss or an
// expired entry (which is evicted).
func (c *Cache) Get(query string, limit int) (results []Result, engineIssues []EngineIssue, ok bool) {
	key := cacheKey(query, limit)
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, found := c.entries[key]
	if !found {
		return nil, nil, false
	}
	if !c.clock.Now().Before(entry.expiresAt) {
		delete(c.entries, key)
		return nil, nil, false
	}
	return entry.results, entry.engineIssues, true
}

// Put stores results for query+limit with a fresh TTL measured from now.
func (c *Cache) Put(query string, limit int, results []Result, engineIssues []EngineIssue) {
	key := cacheKey(query, limit)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{
		results:      results,
		engineIssues: engineIssues,
		expiresAt:    c.clock.Now().Add(c.ttl),
	}
}
