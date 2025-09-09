package middleware

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys
type contextKey string

const (
	// UserContextKey is the key for user information in context
	UserContextKey contextKey = "user"
	// TokenContextKey is the key for token information in context
	TokenContextKey contextKey = "token"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled       bool
	TokenHeader   string
	TokenPrefix   string
	APIKeys       []string
	RequireAuth   bool
	ExcludePaths  []string
}

// NewAuthConfig creates a default auth configuration
func NewAuthConfig() AuthConfig {
	return AuthConfig{
		Enabled:      false,
		TokenHeader:  "Authorization",
		TokenPrefix:  "Bearer ",
		APIKeys:      []string{},
		RequireAuth:  false,
		ExcludePaths: []string{"/health", "/metrics"},
	}
}

// BasicAuth provides basic authentication middleware
func BasicAuth(username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			
			if !ok || user != username || pass != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			
			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyAuth provides API key authentication middleware
func APIKeyAuth(config AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is excluded
			for _, path := range config.ExcludePaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}
			
			// Skip auth if not enabled or not required
			if !config.Enabled || !config.RequireAuth {
				next.ServeHTTP(w, r)
				return
			}
			
			// Get token from header
			authHeader := r.Header.Get(config.TokenHeader)
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}
			
			// Extract token
			token := authHeader
			if config.TokenPrefix != "" {
				if !strings.HasPrefix(authHeader, config.TokenPrefix) {
					http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
					return
				}
				token = strings.TrimPrefix(authHeader, config.TokenPrefix)
			}
			
			// Validate API key
			valid := false
			for _, apiKey := range config.APIKeys {
				if token == apiKey {
					valid = true
					break
				}
			}
			
			if !valid {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}
			
			// Add token to context
			ctx := context.WithValue(r.Context(), TokenContextKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth is a simple middleware that requires authentication
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user is in context (set by previous auth middleware)
		user := r.Context().Value(UserContextKey)
		if user == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// OptionalAuth allows requests with or without authentication
func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This middleware doesn't block, just passes through
		// Previous auth middleware should have added user to context if authenticated
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves user information from context
func GetUserFromContext(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(UserContextKey).(string)
	return user, ok
}

// GetTokenFromContext retrieves token from context
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(TokenContextKey).(string)
	return token, ok
}

// RateLimiter provides basic rate limiting middleware
type RateLimiter struct {
	requests map[string][]int64
	limit    int
	window   int64 // in seconds
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, windowSeconds int64) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]int64),
		limit:    limit,
		window:   windowSeconds,
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client identifier (IP address)
			clientIP := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = forwarded
			}
			
			// Check rate limit
			if !rl.Allow(clientIP) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(clientID string) bool {
	now := time.Now().Unix()
	windowStart := now - rl.window
	
	// Get or create request history for client
	history, exists := rl.requests[clientID]
	if !exists {
		rl.requests[clientID] = []int64{now}
		return true
	}
	
	// Filter out old requests
	var recentRequests []int64
	for _, timestamp := range history {
		if timestamp > windowStart {
			recentRequests = append(recentRequests, timestamp)
		}
	}
	
	// Check if under limit
	if len(recentRequests) >= rl.limit {
		rl.requests[clientID] = recentRequests
		return false
	}
	
	// Add current request
	recentRequests = append(recentRequests, now)
	rl.requests[clientID] = recentRequests
	
	return true
}

// Cleanup removes old entries from rate limiter
func (rl *RateLimiter) Cleanup() {
	now := time.Now().Unix()
	windowStart := now - rl.window
	
	for clientID, history := range rl.requests {
		var recentRequests []int64
		for _, timestamp := range history {
			if timestamp > windowStart {
				recentRequests = append(recentRequests, timestamp)
			}
		}
		
		if len(recentRequests) == 0 {
			delete(rl.requests, clientID)
		} else {
			rl.requests[clientID] = recentRequests
		}
	}
}