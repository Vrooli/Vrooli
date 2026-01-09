package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type CacheClient struct {
	redis  *redis.Client
	Enable bool // Exported for health checks
}

func NewCacheClient() *CacheClient {
	redisURL := os.Getenv("REDIS_URL")

	// Redis is optional - if REDIS_URL not provided, gracefully disable caching
	if redisURL == "" {
		return &CacheClient{
			redis:  nil,
			Enable: false,
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:         redisURL,
		Password:     "",
		DB:           0,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	})

	// Test connection
	_, err := client.Ping(ctx).Result()
	Enable := err == nil

	return &CacheClient{
		redis:  client,
		Enable: Enable,
	}
}

// CacheIngredientCheck stores ingredient check results
func (c *CacheClient) CacheIngredientCheck(ingredients string, isVegan bool, nonVeganItems []string, reasons []string) {
	if !c.Enable {
		return
	}

	key := "ingredient:" + strings.ToLower(strings.TrimSpace(ingredients))
	value := map[string]interface{}{
		"isVegan":       isVegan,
		"nonVeganItems": nonVeganItems,
		"reasons":       reasons,
		"cachedAt":      time.Now().Unix(),
	}

	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	// Cache for 1 hour
	c.redis.Set(ctx, key, data, 1*time.Hour)
}

// GetCachedIngredientCheck retrieves cached results
func (c *CacheClient) GetCachedIngredientCheck(ingredients string) (bool, []string, []string, bool) {
	if !c.Enable {
		return false, nil, nil, false
	}

	key := "ingredient:" + strings.ToLower(strings.TrimSpace(ingredients))
	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		return false, nil, nil, false
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return false, nil, nil, false
	}

	isVegan, _ := result["isVegan"].(bool)

	// Convert nonVeganItems
	var nonVeganItems []string
	if items, ok := result["nonVeganItems"].([]interface{}); ok {
		for _, item := range items {
			if str, ok := item.(string); ok {
				nonVeganItems = append(nonVeganItems, str)
			}
		}
	}

	// Convert reasons
	var reasons []string
	if reasonsList, ok := result["reasons"].([]interface{}); ok {
		for _, reason := range reasonsList {
			if str, ok := reason.(string); ok {
				reasons = append(reasons, str)
			}
		}
	}

	return isVegan, nonVeganItems, reasons, true
}

// GetCacheStats returns cache statistics
func (c *CacheClient) GetCacheStats() map[string]interface{} {
	if !c.Enable {
		return map[string]interface{}{
			"enabled": false,
			"message": "Redis caching is not available",
		}
	}

	info, err := c.redis.Info(ctx, "stats").Result()
	if err != nil {
		return map[string]interface{}{
			"enabled": true,
			"error":   err.Error(),
		}
	}

	return map[string]interface{}{
		"enabled": true,
		"info":    info,
	}
}
