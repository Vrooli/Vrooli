package main

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// DatabaseMiddleware injects database connection into context
func DatabaseMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// LoggingMiddleware logs request details
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log only errors and slow requests
		latency := time.Since(start)
		if c.Writer.Status() >= 400 || latency > 1*time.Second {
			if raw != "" {
				path = path + "?" + raw
			}

			log.Printf("[%s] %s %s %d %v",
				c.GetString("request_id"),
				c.Request.Method,
				path,
				c.Writer.Status(),
				latency,
			)
		}
	}
}

// UserContextMiddleware extracts user information from request
func UserContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from header or token
		// For now, we'll use a header-based approach
		// In production, this should validate JWT tokens
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			// Generate anonymous user ID for this session
			userID = "anon-" + uuid.New().String()[:8]
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// ErrorHandlerMiddleware handles panics and errors
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				c.JSON(500, ErrorResponse{
					Error:   "Internal server error",
					Details: "An unexpected error occurred",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// TimeoutMiddleware adds request timeout
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			c.JSON(504, ErrorResponse{
				Error:   "Request timeout",
				Details: "The request took too long to process",
			})
			c.Abort()
		}
	}
}

// getDB extracts database connection from context
func getDB(c *gin.Context) *sql.DB {
	db, exists := c.Get("db")
	if !exists {
		panic("database not found in context")
	}
	return db.(*sql.DB)
}

// getUserID extracts user ID from context
func getUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return "system"
	}
	return userID.(string)
}

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable browser XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'")

		// Permissions Policy (formerly Feature-Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// RequestSizeLimitMiddleware limits the size of incoming requests
func RequestSizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = &limitedReader{
			reader:   c.Request.Body,
			maxBytes: maxBytes,
		}
		c.Next()
	}
}

// limitedReader wraps request body to enforce size limit
type limitedReader struct {
	reader interface {
		Read([]byte) (int, error)
		Close() error
	}
	maxBytes int64
	n        int64
}

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	if lr.n >= lr.maxBytes {
		return 0, &requestTooLargeError{}
	}

	// Limit read size
	maxRead := lr.maxBytes - lr.n
	if int64(len(p)) > maxRead {
		p = p[0:maxRead]
	}

	n, err = lr.reader.Read(p)
	lr.n += int64(n)

	if lr.n > lr.maxBytes {
		return n, &requestTooLargeError{}
	}

	return n, err
}

func (lr *limitedReader) Close() error {
	return lr.reader.Close()
}

type requestTooLargeError struct{}

func (e *requestTooLargeError) Error() string {
	return "request body too large"
}

// RateLimitMiddleware implements rate limiting per IP address
func RateLimitMiddleware(requestsPerSecond int, burst int) gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Cleanup old clients every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 10*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		if _, exists := clients[ip]; !exists {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
			}
		}
		clients[ip].lastSeen = time.Now()
		limiter := clients[ip].limiter
		mu.Unlock()

		if !limiter.Allow() {
			c.JSON(429, ErrorResponse{
				Error:   "Rate limit exceeded",
				Details: "Too many requests, please slow down",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
