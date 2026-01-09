package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a secure default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	// Get allowed origins from environment (required)
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string

	if origins != "" {
		allowedOrigins = strings.Split(origins, ",")
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	} else {
		// No CORS origins configured - will be restrictive
		allowedOrigins = []string{}
	}

	return &CORSConfig{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Content-Length", "Accept", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORS creates a CORS middleware with security best practices
func CORS(config *CORSConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCORSConfig()
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if isOriginAllowed(origin, config.AllowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)

			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			c.Header("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed checks if an origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	for _, allowed := range allowedOrigins {
		// Handle wildcard subdomain matching (*.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}

		// Handle exact match
		if origin == allowed {
			return true
		}

		// Handle wildcard for all origins (use with caution)
		if allowed == "*" {
			return true
		}
	}

	return false
}

// SecureWebSocketUpgrader creates a WebSocket upgrader with proper origin validation
func SecureWebSocketUpgrader() *websocket.Upgrader {
	// Get allowed origins from environment (required)
	origins := os.Getenv("WS_ALLOWED_ORIGINS")
	var allowedOrigins []string

	if origins != "" {
		allowedOrigins = strings.Split(origins, ",")
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	} else {
		// No WebSocket origins configured - will be restrictive
		allowedOrigins = []string{}
	}

	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")

			// If no origin header, check the host
			if origin == "" {
				host := r.Host
				// Allow same-host connections
				if host == r.Header.Get("Host") {
					return true
				}
			}

			// In development mode, be more permissive
			if os.Getenv("ENV") == "development" || os.Getenv("NODE_ENV") == "development" {
				// Allow localhost connections
				if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
					return true
				}
			}

			// Check against allowed origins
			return isOriginAllowed(origin, allowedOrigins)
		},
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			// Don't expose internal error details to client
			http.Error(w, "WebSocket upgrade failed", status)
		},
	}
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent XSS attacks
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")

		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"font-src 'self' https://fonts.gstatic.com; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self' ws: wss:;"

		c.Header("Content-Security-Policy", csp)

		// Strict Transport Security (for HTTPS)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		c.Next()
	}
}

// RateLimiting provides basic rate limiting middleware
func RateLimiting(requestsPerMinute int) gin.HandlerFunc {
	// Simple in-memory rate limiting (consider using Redis for production)
	clientRequests := make(map[string][]int64)

	return func(c *gin.Context) {
		// Get client identifier (IP address)
		clientIP := c.ClientIP()
		now := time.Now().Unix()

		// Clean old entries
		if requests, exists := clientRequests[clientIP]; exists {
			cutoff := now - 60
			var filtered []int64
			for _, timestamp := range requests {
				if timestamp > cutoff {
					filtered = append(filtered, timestamp)
				}
			}
			clientRequests[clientIP] = filtered
		}

		// Check rate limit
		if len(clientRequests[clientIP]) >= requestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		// Add current request
		clientRequests[clientIP] = append(clientRequests[clientIP], now)

		c.Next()
	}
}

// ValidateAPIKey provides API key validation middleware
func ValidateAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip API key validation for health checks
		if strings.HasPrefix(c.Request.URL.Path, "/health") {
			c.Next()
			return
		}

		// Get API key from environment
		expectedKey := os.Getenv("API_KEY")
		if expectedKey == "" {
			// If no API key is configured, allow all requests (for development)
			c.Next()
			return
		}

		// Check for API key in header or query parameter
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey != expectedKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing API key",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
