package interfaces

import "time"

// CacheManager defines the interface for caching operations
type CacheManager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
	Stats() CacheStats
	Close()
}

// CacheStats provides cache statistics
type CacheStats struct {
	Size     int
	Hits     int64
	Misses   int64
	HitRate  float64
}