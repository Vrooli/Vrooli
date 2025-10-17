package main

import (
	"net/http"
	"os"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use explicit allowed origin from environment, default to localhost for development
		allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			// Default to localhost with UI_PORT for local development
			uiPort := os.Getenv("UI_PORT")
			if uiPort != "" {
				allowedOrigin = "http://localhost:" + uiPort
			} else {
				allowedOrigin = "http://localhost:36222"
			}
		}
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers are already set by corsMiddleware
	w.WriteHeader(http.StatusOK)
}
