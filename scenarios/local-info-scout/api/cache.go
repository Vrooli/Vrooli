package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	ctx         = context.Background()
	cacheTTL    = 5 * time.Minute // Cache TTL for search results
)

// initRedis initializes the Redis client for caching
func initRedis() {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password for local Redis
		DB:       0,  // use default DB
	})

	// Test Redis connection
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		cacheLogger.Warn("Redis not available, caching disabled", map[string]interface{}{
			"address": redisAddr,
			"error":   err.Error(),
		})
		redisClient = nil
	} else {
		cacheLogger.Info("Redis connected, caching enabled", map[string]interface{}{
			"address": redisAddr,
		})
	}
}

// getCacheKey generates a cache key for search requests
func getCacheKey(req SearchRequest) string {
	return fmt.Sprintf("search:%f:%f:%f:%s:%f:%d:%t:%t",
		req.Lat, req.Lon, req.Radius, req.Category,
		req.MinRating, req.MaxPrice, req.OpenNow, req.Accessible)
}

// getFromCache retrieves data from Redis cache
func getFromCache(key string) ([]Place, bool) {
	if redisClient == nil {
		return nil, false
	}

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var places []Place
	if err := json.Unmarshal([]byte(val), &places); err != nil {
		return nil, false
	}

	cacheLogger.Debug("Cache hit", map[string]interface{}{
		"key": key,
	})
	return places, true
}

// saveToCache saves data to Redis cache
func saveToCache(key string, places []Place) {
	if redisClient == nil {
		return
	}

	data, err := json.Marshal(places)
	if err != nil {
		return
	}

	if err := redisClient.Set(ctx, key, data, cacheTTL).Err(); err != nil {
		cacheLogger.Error("Failed to cache data", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
	} else {
		cacheLogger.Debug("Cached data", map[string]interface{}{
			"key": key,
		})
	}
}

// clearCache clears all cached search results
func clearCache() error {
	if redisClient == nil {
		return nil
	}

	iter := redisClient.Scan(ctx, 0, "search:*", 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return redisClient.Del(ctx, keys...).Err()
	}
	return nil
}

// getRedisClient returns the Redis client (useful for testing)
func getRedisClient() *redis.Client {
	return redisClient
}
