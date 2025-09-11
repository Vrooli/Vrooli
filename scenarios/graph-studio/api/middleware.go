package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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