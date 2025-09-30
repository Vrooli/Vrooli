package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/models"
	"scenario-authenticator/utils"
	"strings"
	"sync"
	"time"
)

// RateLimiter stores rate limit data in memory with Redis fallback
type RateLimiter struct {
	mu      sync.RWMutex
	limits  map[string]*RateLimit
	cleanup *time.Ticker
}

// RateLimit represents a rate limit entry
type RateLimit struct {
	Count      int
	ResetTime  time.Time
	Identifier string
	Limit      int
}

var limiter = &RateLimiter{
	limits:  make(map[string]*RateLimit),
	cleanup: time.NewTicker(1 * time.Minute),
}

func init() {
	// Start cleanup goroutine
	go func() {
		for range limiter.cleanup.C {
			limiter.cleanupExpired()
		}
	}()
}

// cleanupExpired removes expired rate limit entries
func (rl *RateLimiter) cleanupExpired() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, limit := range rl.limits {
		if now.After(limit.ResetTime) {
			delete(rl.limits, key)
		}
	}
}

// RateLimitMiddleware implements per-user and per-API-key rate limiting
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identifier := getIdentifier(r)
		endpoint := r.URL.Path
		limit := getLimit(r)

		// Create key for this identifier+endpoint combination
		key := fmt.Sprintf("%s:%s", identifier, endpoint)

		// Check rate limit
		if !checkRateLimit(key, limit) {
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
			utils.SendError(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Set rate limit headers
		remaining := getRemainingRequests(key, limit)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

		next.ServeHTTP(w, r)
	}
}

// getIdentifier extracts the rate limit identifier from the request
func getIdentifier(r *http.Request) string {
	// Check for API key first
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return fmt.Sprintf("apikey:%s", hashKey(apiKey))
	}

	// Check for JWT token
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := authHeader[7:]
		if claims, err := auth.ValidateToken(token); err == nil {
			return fmt.Sprintf("user:%s", claims.UserID)
		}
	}

	// Fall back to IP address
	return fmt.Sprintf("ip:%s", auth.GetClientIP(r))
}

// getLimit returns the rate limit for the request
func getLimit(r *http.Request) int {
	// Check for API key with custom limit
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		// Remove prefix if present
		if len(apiKey) > 3 && apiKey[:3] == "sk_" {
			apiKey = apiKey[3:]
		}

		// Look up API key rate limit from database
		var rateLimit int
		keyHash := auth.HashToken(apiKey)
		err := db.DB.QueryRow(`
			SELECT rate_limit FROM api_keys
			WHERE key_hash = $1 AND revoked_at IS NULL`,
			keyHash,
		).Scan(&rateLimit)

		if err == nil && rateLimit > 0 {
			return rateLimit
		}
	}

	// Check for authenticated user
	if claims := r.Context().Value("claims"); claims != nil {
		if userClaims, ok := claims.(*models.Claims); ok {
			// Admin users get higher limits
			for _, role := range userClaims.Roles {
				if role == "admin" {
					return 10000 // 10k requests per minute for admins
				}
			}
			return 1000 // 1k requests per minute for regular users
		}
	}

	// Default limit for unauthenticated requests
	return 100 // 100 requests per minute
}

// checkRateLimit checks if the request is within rate limits
func checkRateLimit(key string, limit int) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()

	// Check if we have an existing rate limit entry
	if rl, exists := limiter.limits[key]; exists {
		// If the window has expired, reset
		if now.After(rl.ResetTime) {
			rl.Count = 1
			rl.ResetTime = now.Add(time.Minute)
			rl.Limit = limit
			return true
		}

		// Check if we've exceeded the limit
		if rl.Count >= rl.Limit {
			return false
		}

		// Increment counter
		rl.Count++
		return true
	}

	// Create new rate limit entry
	limiter.limits[key] = &RateLimit{
		Count:      1,
		ResetTime:  now.Add(time.Minute),
		Identifier: key,
		Limit:      limit,
	}

	// Also store in Redis for distributed rate limiting
	go storeInRedis(key, limit)

	return true
}

// getRemainingRequests returns the number of remaining requests
func getRemainingRequests(key string, limit int) int {
	limiter.mu.RLock()
	defer limiter.mu.RUnlock()

	if rl, exists := limiter.limits[key]; exists {
		remaining := rl.Limit - rl.Count
		if remaining < 0 {
			return 0
		}
		return remaining
	}

	return limit - 1
}

// storeInRedis stores rate limit data in Redis for distributed limiting
func storeInRedis(key string, limit int) {
	// Create Redis key with TTL
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// Store in Redis with 1 minute TTL
	err := db.RedisClient.Set(db.Ctx, redisKey, 1, time.Minute).Err()
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to store rate limit in Redis: %v\n", err)
	}
}

// hashKey creates a simple hash of the API key for storage
func hashKey(key string) string {
	// Use the existing HashToken function
	return auth.HashToken(key)[:16] // Use first 16 chars for brevity
}

// APIKeyMiddleware validates API key authentication
func APIKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			utils.SendError(w, "API key required", http.StatusUnauthorized)
			return
		}

		// Remove prefix if present
		if len(apiKey) > 3 && apiKey[:3] == "sk_" {
			apiKey = apiKey[3:]
		}

		// Validate API key
		keyHash := auth.HashToken(apiKey)
		var userID string
		var permissions string

		err := db.DB.QueryRow(`
			SELECT user_id, permissions
			FROM api_keys
			WHERE key_hash = $1 AND revoked_at IS NULL
			AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)`,
			keyHash,
		).Scan(&userID, &permissions)

		if err != nil {
			utils.SendError(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Update last used
		go func() {
			db.DB.Exec("UPDATE api_keys SET last_used = CURRENT_TIMESTAMP WHERE key_hash = $1", keyHash)
		}()

		// Parse permissions and create pseudo-claims
		var perms []string
		json.Unmarshal([]byte(permissions), &perms)

		// Add to context
		claims := &models.Claims{
			UserID: userID,
			Roles:  perms, // Use permissions as roles for compatibility
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "claims", claims)
		ctx = context.WithValue(ctx, "auth_method", "api_key")

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}