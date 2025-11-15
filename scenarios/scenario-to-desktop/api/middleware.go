package main

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"
)

// corsMiddleware handles CORS for cross-origin requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SECURITY: Get allowed origin from environment - required for production
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			// SECURITY: In development, UI_PORT must be provided by lifecycle system
			// In production, ALLOWED_ORIGIN should be explicitly configured
			uiPort := os.Getenv("UI_PORT")
			if uiPort != "" {
				// UI_PORT is set - use it to build allowed origins
				origin := r.Header.Get("Origin")
				allowedOrigins := []string{
					"http://localhost:" + uiPort,
					"http://127.0.0.1:" + uiPort,
				}

				// Check if origin matches any allowed origin
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						allowedOrigin = origin
						break
					}
				}

				// If no match found, use primary UI port
				if allowedOrigin == "" {
					allowedOrigin = "http://localhost:" + uiPort
				}
			} else {
				// SECURITY: Neither ALLOWED_ORIGIN nor UI_PORT is set
				// This is a configuration error - log and use restrictive CORS
				globalLogger.Warn("CORS configuration missing",
					"message", "Neither ALLOWED_ORIGIN nor UI_PORT is set",
					"action", "using restrictive localhost-only CORS for security")
				// Set to request origin only if it's localhost
				origin := r.Header.Get("Origin")
				if origin != "" && (origin == "http://localhost" || origin == "http://127.0.0.1") {
					allowedOrigin = origin
				} else {
					allowedOrigin = "http://localhost" // Minimal fallback
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Build-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs all HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, next)
}
