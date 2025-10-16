package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	EnableAuth     bool     `json:"enable_auth"`
	AllowedOrigins []string `json:"allowed_origins"`
	RateLimit      int      `json:"rate_limit"`
	APITokens      []string `json:"api_tokens"`
}

// InputValidator validates and sanitizes input
type InputValidator struct {
	titleRegex        *regexp.Regexp
	descRegex         *regexp.Regexp
	filePathRegex     *regexp.Regexp
	maxTitleLength    int
	maxDescLength     int
	allowedTypes      map[string]bool
	allowedPriorities map[string]bool
	allowedStatuses   map[string]bool
}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{
		titleRegex:        regexp.MustCompile(`^[\w\s\-\.\!\?\,\(\)]+$`),
		descRegex:         regexp.MustCompile(`^[\w\s\-\.\!\?\,\(\)\n\r]+$`),
		filePathRegex:     regexp.MustCompile(`^[\w\-\/\.]+$`),
		maxTitleLength:    200,
		maxDescLength:     5000,
		allowedTypes:      map[string]bool{"bug": true, "feature": true, "task": true, "improvement": true},
		allowedPriorities: map[string]bool{"critical": true, "high": true, "medium": true, "low": true},
		allowedStatuses:   map[string]bool{"open": true, "active": true, "completed": true, "failed": true},
	}
}

// ValidateIssue validates issue input
func (v *InputValidator) ValidateIssue(issue *Issue) error {
	// Validate title
	if issue.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(issue.Title) > v.maxTitleLength {
		return fmt.Errorf("title exceeds maximum length of %d characters", v.maxTitleLength)
	}
	if !v.titleRegex.MatchString(issue.Title) {
		return fmt.Errorf("title contains invalid characters")
	}

	// Validate description
	if len(issue.Description) > v.maxDescLength {
		return fmt.Errorf("description exceeds maximum length of %d characters", v.maxDescLength)
	}

	// Validate type
	if issue.Type != "" && !v.allowedTypes[issue.Type] {
		return fmt.Errorf("invalid issue type: %s", issue.Type)
	}

	// Validate priority
	if issue.Priority != "" && !v.allowedPriorities[issue.Priority] {
		return fmt.Errorf("invalid priority: %s", issue.Priority)
	}

	// Validate status
	if issue.Status != "" && !v.allowedStatuses[issue.Status] {
		return fmt.Errorf("invalid status: %s", issue.Status)
	}

	// Sanitize file paths in ErrorContext
	for i, path := range issue.ErrorContext.AffectedFiles {
		issue.ErrorContext.AffectedFiles[i] = filepath.Clean(path)
		if strings.Contains(issue.ErrorContext.AffectedFiles[i], "..") {
			return fmt.Errorf("invalid file path: %s", path)
		}
	}

	return nil
}

// SanitizePath sanitizes file paths to prevent directory traversal
func SanitizePath(path string) (string, error) {
	// Clean the path
	cleaned := filepath.Clean(path)

	// Check for directory traversal
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("invalid path: contains directory traversal")
	}

	// Check for absolute paths
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("invalid path: absolute paths not allowed")
	}

	return cleaned, nil
}

// RateLimiter implements basic rate limiting
type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request is allowed
func (r *RateLimiter) Allow(clientID string) bool {
	now := time.Now()

	// Clean old requests
	if times, exists := r.requests[clientID]; exists {
		var validTimes []time.Time
		for _, t := range times {
			if now.Sub(t) <= r.window {
				validTimes = append(validTimes, t)
			}
		}
		r.requests[clientID] = validTimes
	}

	// Check limit
	if len(r.requests[clientID]) >= r.limit {
		return false
	}

	// Add current request
	r.requests[clientID] = append(r.requests[clientID], now)
	return true
}

// authMiddleware provides basic authentication
func authMiddleware(next http.Handler, tokens []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoint
		if r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for API token
		token := r.Header.Get("X-API-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		// If tokens are configured, validate
		if len(tokens) > 0 {
			valid := false
			for _, validToken := range tokens {
				if token == validToken {
					valid = true
					break
				}
			}

			if !valid {
				response := ApiResponse{
					Success: false,
					Message: "Unauthorized: Invalid or missing API token",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(response)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// securedCorsMiddleware provides more restrictive CORS handling
func securedCorsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Token")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitMiddleware applies rate limiting
func rateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as client ID
			clientID := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientID = forwarded
			}

			if !limiter.Allow(clientID) {
				response := ApiResponse{
					Success: false,
					Message: "Rate limit exceeded",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(response)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoadSecurityConfig loads security configuration

func splitAndTrimCSV(value string) []string {
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func LoadSecurityConfig(path string) (*SecurityConfig, error) {
	config := &SecurityConfig{
		EnableAuth:     false,
		AllowedOrigins: []string{"*"},
		RateLimit:      100,
		APITokens:      []string{},
	}

	// Try to load from environment
	if envAuth, ok := os.LookupEnv("ENABLE_AUTH"); ok {
		config.EnableAuth = strings.EqualFold(strings.TrimSpace(envAuth), "true")
	}

	if envOrigins, ok := os.LookupEnv("ALLOWED_ORIGINS"); ok {
		trimmed := strings.TrimSpace(envOrigins)
		if trimmed == "" {
			return nil, fmt.Errorf("ALLOWED_ORIGINS cannot be empty when set")
		}
		config.AllowedOrigins = splitAndTrimCSV(trimmed)
	}

	if envTokens, ok := os.LookupEnv("API_TOKENS"); ok {
		trimmed := strings.TrimSpace(envTokens)
		if trimmed == "" {
			return nil, fmt.Errorf("API_TOKENS cannot be empty when set")
		}
		config.APITokens = splitAndTrimCSV(trimmed)
	}

	if config.EnableAuth && len(config.APITokens) == 0 {
		return nil, fmt.Errorf("API_TOKENS must be provided when ENABLE_AUTH is true")
	}

	// Try to load from file
	if path != "" {
		if data, err := ioutil.ReadFile(path); err == nil {
			json.Unmarshal(data, config)
		}
	}

	return config, nil
}
