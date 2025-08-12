package cache

import (
	"sync"
	"time"

	"metareasoning-api/internal/pkg/interfaces"
)

// MemoryCache implements interfaces.CacheManager with in-memory storage
type MemoryCache struct {
	cache       map[string]cacheEntry
	mutex       sync.RWMutex
	maxEntries  int
	stats       interfaces.CacheStats
	lastCleanup time.Time
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() interfaces.CacheManager {
	cache := &MemoryCache{
		cache:      make(map[string]cacheEntry),
		maxEntries: 1000, // Prevent unlimited growth
		stats:      interfaces.CacheStats{},
	}
	
	// Start cleanup goroutine
	go cache.cleanupRoutine()
	
	return cache
}

// Get retrieves a value from cache
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	entry, exists := mc.cache[key]
	if !exists {
		mc.stats.Misses++
		return nil, false
	}
	
	// Check expiration
	if time.Now().After(entry.expiresAt) {
		mc.stats.Misses++
		// Delete expired entry (upgrade to write lock)
		mc.mutex.RUnlock()
		mc.mutex.Lock()
		delete(mc.cache, key)
		mc.stats.Size = len(mc.cache)
		mc.mutex.Unlock()
		mc.mutex.RLock()
		return nil, false
	}
	
	mc.stats.Hits++
	mc.updateHitRate()
	
	return entry.value, true
}

// Set stores a value in cache with TTL
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	// Check if we need to make room
	if len(mc.cache) >= mc.maxEntries {
		mc.evictOldest()
	}
	
	mc.cache[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	
	mc.stats.Size = len(mc.cache)
}

// Delete removes a key from cache
func (mc *MemoryCache) Delete(key string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	delete(mc.cache, key)
	mc.stats.Size = len(mc.cache)
}

// Clear removes all entries from cache
func (mc *MemoryCache) Clear() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	mc.cache = make(map[string]cacheEntry)
	mc.stats.Size = 0
}

// Stats returns current cache statistics
func (mc *MemoryCache) Stats() interfaces.CacheStats {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	return mc.stats
}

// Close shuts down the cache
func (mc *MemoryCache) Close() {
	mc.Clear()
}

// evictOldest removes the oldest entries to make room
func (mc *MemoryCache) evictOldest() {
	// Simple eviction: remove 10% of entries
	toRemove := len(mc.cache) / 10
	if toRemove < 1 {
		toRemove = 1
	}
	
	count := 0
	for key := range mc.cache {
		delete(mc.cache, key)
		count++
		if count >= toRemove {
			break
		}
	}
}

// updateHitRate calculates the current hit rate
func (mc *MemoryCache) updateHitRate() {
	total := mc.stats.Hits + mc.stats.Misses
	if total > 0 {
		mc.stats.HitRate = float64(mc.stats.Hits) / float64(total)
	}
}

// cleanupRoutine periodically removes expired entries
func (mc *MemoryCache) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		mc.cleanup()
	}
}

// cleanup removes expired entries
func (mc *MemoryCache) cleanup() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	now := time.Now()
	for key, entry := range mc.cache {
		if now.After(entry.expiresAt) {
			delete(mc.cache, key)
		}
	}
	
	mc.stats.Size = len(mc.cache)
	mc.lastCleanup = now
}