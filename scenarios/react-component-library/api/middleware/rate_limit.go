package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Simple in-memory rate limiter (in production, would use Redis)
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) IsAllowed(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean old requests
	requests := rl.requests[clientIP]
	validRequests := []time.Time{}
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if limit exceeded
	if len(validRequests) >= rl.limit {
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[clientIP] = validRequests

	return true
}

var (
	generalLimiter *RateLimiter
	aiLimiter      *RateLimiter
)

// RateLimit creates rate limiting middleware
func RateLimit(requestsPerMinute, aiRequestsPerMinute int) gin.HandlerFunc {
	generalLimiter = NewRateLimiter(requestsPerMinute, time.Minute)
	aiLimiter = NewRateLimiter(aiRequestsPerMinute, time.Minute)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !generalLimiter.IsAllowed(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AIRateLimit creates special rate limiting for AI endpoints
func AIRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if aiLimiter != nil && !aiLimiter.IsAllowed(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "AI rate limit exceeded",
				"message": "Too many AI requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// generateRequestID creates a simple request ID
func generateRequestID() string {
	return strings.Replace(time.Now().Format("20060102150405.000000"), ".", "", -1)
}