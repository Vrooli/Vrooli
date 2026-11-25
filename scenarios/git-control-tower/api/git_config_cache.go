package main

import (
	"context"
	"errors"
	"strconv"
	"strings"
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
	flight  singleflight.Group
}

// RepoStatusCache coalesces concurrent status polls and keeps the result for a
// short TTL. Status is intentionally not cached globally: the repository path
// and hotspot mode are part of the key, and callers without this dependency
// continue to get a fresh read.
type RepoStatusCache struct {
	mu      sync.RWMutex
	entries map[string]repoStatusCacheEntry
	// generations scopes in-flight loads to the repository state observed
	// before loading. Invalidation advances the generation to reject old fills.
	generations   map[string]uint64
	ttl           time.Duration
	now           func() time.Time
	flight        singleflight.Group
	pageSnapshots *RepoStatusPageCache
}

type repoStatusCacheEntry struct {
	value     *RepoStatus
	expiresAt time.Time
}

var errRepoStatusCacheInvalidated = errors.New("repository status cache invalidated during load")

func NewRepoStatusCache(ttl time.Duration) *RepoStatusCache {
	return &RepoStatusCache{
		entries:     make(map[string]repoStatusCacheEntry),
		generations: make(map[string]uint64),
		ttl:         ttl,
		now:         time.Now,
	}
}

func (c *RepoStatusCache) SetPageSnapshots(cache *RepoStatusPageCache) {
	if c != nil {
		c.mu.Lock()
		c.pageSnapshots = cache
		c.mu.Unlock()
	}
}

// Invalidate removes all cached views for a repository after a mutation and
// advances its generation so an older in-flight load cannot restore stale data.
func (c *RepoStatusCache) Invalidate(repoDir string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	pageSnapshots := c.pageSnapshots
	defer c.mu.Unlock()
	c.generations[repoDir]++
	for key := range c.entries {
		if repoStatusCacheRepo(key) == repoDir {
			delete(c.entries, key)
		}
	}
	if pageSnapshots != nil {
		pageSnapshots.InvalidateRepo(repoDir)
	}
}

// Generation reports the current repository generation. Page assembly uses it
// to make sure an invalidated status read cannot publish a later continuation.
func (c *RepoStatusCache) Generation(repoDir string) uint64 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.generations[repoDir]
}

// WithGeneration runs publish while invalidation is excluded. This closes the
// gap between a status-cache hit and publishing its immutable page snapshot.
func (c *RepoStatusCache) WithGeneration(repoDir string, generation uint64, publish func() error) (bool, error) {
	if c == nil {
		return true, publish()
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.generations[repoDir] != generation {
		return false, nil
	}
	return true, publish()
}

func (c *RepoStatusCache) Get(ctx context.Context, key string, load func(context.Context) (*RepoStatus, error)) (*RepoStatus, error) {
	if c == nil {
		return load(ctx)
	}
	repoDir := repoStatusCacheRepo(key)
	for {
		c.mu.RLock()
		generation := c.generations[repoDir]
		entry, ok := c.entries[key]
		fresh := ok && c.now().Before(entry.expiresAt)
		c.mu.RUnlock()
		if fresh {
			return entry.value, nil
		}

		flightKey := key + "\x00generation:" + strconv.FormatUint(generation, 10)
		v, err, _ := c.flight.Do(flightKey, func() (any, error) {
			c.mu.RLock()
			entry, ok := c.entries[key]
			currentGeneration := c.generations[repoDir]
			if ok && c.now().Before(entry.expiresAt) {
				c.mu.RUnlock()
				return entry.value, nil
			}
			c.mu.RUnlock()
			if currentGeneration != generation {
				return nil, errRepoStatusCacheInvalidated
			}
			value, err := load(ctx)
			if err != nil {
				return nil, err
			}
			c.mu.Lock()
			if c.generations[repoDir] != generation {
				c.mu.Unlock()
				return nil, errRepoStatusCacheInvalidated
			}
			c.entries[key] = repoStatusCacheEntry{value: value, expiresAt: c.now().Add(c.ttl)}
			c.mu.Unlock()
			return value, nil
		})
		if errors.Is(err, errRepoStatusCacheInvalidated) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return v.(*RepoStatus), nil
	}
}

func repoStatusCacheRepo(key string) string {
	if separator := strings.IndexByte(key, '\x00'); separator >= 0 {
		return key[:separator]
	}
	return key
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
	flightKey := cacheKey(repoDir, key)
	result := c.flight.DoChan(flightKey, func() (any, error) {
		// Another request may have filled the cache between the fast-path read
		// above and this call becoming the singleflight leader.
		if val, ok := c.Get(repoDir, key); ok {
			return val, nil
		}
		val, err := git.ConfigGet(ctx, repoDir, key)
		if err != nil {
			return "", err
		}
		c.Set(repoDir, key, val)
		return val, nil
	})
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case shared := <-result:
		if shared.Err != nil {
			return "", shared.Err
		}
		return shared.Val.(string), nil
	}
}
