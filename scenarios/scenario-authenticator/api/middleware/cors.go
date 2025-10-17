package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORSMiddleware adds CORS headers to responses with configurable origins
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment, default to localhost for development
		allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			// Default to secure localhost origins for development
			allowedOrigins = "http://localhost:3000,http://localhost:5173,http://localhost:8080"
		}

		// Check if request origin is allowed
		origin := r.Header.Get("Origin")
		if origin != "" {
			origins := strings.Split(allowedOrigins, ",")
			allowed := false
			for _, allowedOrigin := range origins {
				if strings.TrimSpace(allowedOrigin) == origin || strings.TrimSpace(allowedOrigin) == "*" {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
