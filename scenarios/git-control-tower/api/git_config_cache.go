package main

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// GitConfigCache caches git config values (e.g. user.name, user.email)
// with a per-repo TTL to avoid spawning a git process on every status poll.
type GitConfigCache struct {
	mu      sync.RWMutex
	entries map[string]configCacheEntry
	ttl     time.Duration
	now     func() time.Time // injectable clock for testing
}

// RepoStatusCache coalesces concurrent status polls and keeps the result for a
// short TTL. Status is intentionally not cached globally: the repository path
// and hotspot mode are part of the key, and callers without this dependency
// continue to get a fresh read.
type RepoStatusCache struct {
	mu      sync.RWMutex
	entries map[string]repoStatusCacheEntry
	ttl     time.Duration
	now     func() time.Time
	flight  singleflight.Group
}

type repoStatusCacheEntry struct {
	value     *RepoStatus
	expiresAt time.Time
}

func NewRepoStatusCache(ttl time.Duration) *RepoStatusCache {
	return &RepoStatusCache{entries: make(map[string]repoStatusCacheEntry), ttl: ttl, now: time.Now}
}

// Invalidate removes all cached views for a repository after a mutation. The
// hotspot and non-hotspot variants are separate keys because their payloads
// differ.
func (c *RepoStatusCache) Invalidate(repoDir string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, repoDir)
	delete(c.entries, repoDir+"\x00hotspots")
}

func (c *RepoStatusCache) Get(ctx context.Context, key string, load func(context.Context) (*RepoStatus, error)) (*RepoStatus, error) {
	if c == nil {
		return load(ctx)
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	fresh := ok && c.now().Before(entry.expiresAt)
	c.mu.RUnlock()
	if fresh {
		return entry.value, nil
	}
	v, err, _ := c.flight.Do(key, func() (any, error) {
		c.mu.RLock()
		entry, ok := c.entries[key]
		if ok && c.now().Before(entry.expiresAt) {
			c.mu.RUnlock()
			return entry.value, nil
		}
		c.mu.RUnlock()
		value, err := load(ctx)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.entries[key] = repoStatusCacheEntry{value: value, expiresAt: c.now().Add(c.ttl)}
		c.mu.Unlock()
		return value, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*RepoStatus), nil
}

type configCacheEntry struct {
	value     string
	expiresAt time.Time
}

// NewGitConfigCache creates a cache with the given TTL.
func NewGitConfigCache(ttl time.Duration) *GitConfigCache {
	return &GitConfigCache{
		entries: make(map[string]configCacheEntry),
		ttl:     ttl,
		now:     time.Now,
	}
}

// cacheKey builds a composite key from repoDir and config key.
func cacheKey(repoDir, key string) string {
	return repoDir + "\x00" + key
}

// Get returns a cached value and true if it exists and hasn't expired.
func (c *GitConfigCache) Get(repoDir, key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[cacheKey(repoDir, key)]
	if !ok || c.now().After(entry.expiresAt) {
		return "", false
	}
	return entry.value, true
}

// Set stores a value with the configured TTL.
func (c *GitConfigCache) Set(repoDir, key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[cacheKey(repoDir, key)] = configCacheEntry{
		value:     value,
		expiresAt: c.now().Add(c.ttl),
	}
}

// ConfigGet returns a cached config value, falling back to the GitRunner
// on a cache miss. Both hits and successful lookups are cached for the TTL.
func (c *GitConfigCache) ConfigGet(ctx context.Context, git GitRunner, repoDir, key string) (string, error) {
	if val, ok := c.Get(repoDir, key); ok {
		return val, nil
	}

	val, err := git.ConfigGet(ctx, repoDir, key)
	if err != nil {
		return "", err
	}
	c.Set(repoDir, key, val)
	return val, nil
}
