package main

import (
	"context"
	"sync"
	"time"
)

// GitConfigCache caches git config values (e.g. user.name, user.email)
// with a per-repo TTL to avoid spawning a git process on every status poll.
type GitConfigCache struct {
	mu      sync.RWMutex
	entries map[string]configCacheEntry
	ttl     time.Duration
	now     func() time.Time // injectable clock for testing
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
