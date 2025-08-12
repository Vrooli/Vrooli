package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

// Middleware functions

// loggingMiddleware logs all incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		next.ServeHTTP(wrapped, r)
		
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

// corsMiddleware adds CORS headers with configurable origins
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Get allowed origins from environment or use secure defaults
		allowedOrigins := getAllowedOrigins()
		
		// Check if origin is allowed
		originAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				originAllowed = true
				break
			}
		}
		
		// Only set CORS headers for allowed origins
		if originAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight for 24 hours
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// authMiddleware checks for valid authentication token
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check for Bearer token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}
		
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}
		
		token := parts[1]
		
		// Simple token validation (in production, use proper JWT or OAuth)
		if !isValidToken(token) {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		
		// Add user context
		ctx := context.WithValue(r.Context(), "user", getUserFromToken(token))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// recoveryMiddleware recovers from panics
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}

// Helper types and functions

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func isValidToken(token string) bool {
	// Simple token validation - in production use proper authentication
	validTokens := []string{
		"metareasoning_cli_default_2024",
		"test_token_2024",
		"admin_token_2024",
	}
	
	for _, valid := range validTokens {
		if token == valid {
			return true
		}
	}
	
	return false
}

func getUserFromToken(token string) string {
	// Map tokens to users - in production decode JWT or lookup in database
	switch token {
	case "admin_token_2024":
		return "admin"
	case "metareasoning_cli_default_2024":
		return "cli"
	default:
		return "user"
	}
}

// getAllowedOrigins returns the list of allowed CORS origins
func getAllowedOrigins() []string {
	// Get from environment variable, fallback to secure defaults
	if corsOrigins := getEnv("CORS_ALLOWED_ORIGINS", ""); corsOrigins != "" {
		return strings.Split(corsOrigins, ",")
	}
	
	// Secure defaults for development and production
	return []string{
		"http://localhost:3000",   // React dev server
		"http://localhost:5173",   // Vite dev server  
		"http://localhost:8080",   // General dev server
		"https://app.vrooli.com",  // Production frontend
		"https://vrooli.com",      // Production main site
	}
}