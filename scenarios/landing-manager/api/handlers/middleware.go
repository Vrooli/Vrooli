package handlers

import (
	"net/http"
	"time"

	"landing-manager/util"
)

// LoggingMiddleware logs request details
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fields := map[string]interface{}{
			"method":   r.Method,
			"path":     r.RequestURI,
			"duration": time.Since(start).String(),
		}
		util.LogStructured("request_completed", fields)
	})
}

// RequestSizeLimitMiddleware limits request body size to prevent DoS attacks
func RequestSizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Limit request body to 10MB
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
		next.ServeHTTP(w, r)
	})
}
