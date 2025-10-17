package middleware

import (
	"net/http"
	"os"
	"strconv"
)

// RequestSizeLimitMiddleware limits the size of incoming request bodies to prevent DoS attacks
func RequestSizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Default max request size is 1MB, configurable via MAX_REQUEST_SIZE_MB
		maxSizeMB := int64(1)
		if envSize := os.Getenv("MAX_REQUEST_SIZE_MB"); envSize != "" {
			if size, err := strconv.ParseInt(envSize, 10, 64); err == nil && size > 0 && size <= 100 {
				maxSizeMB = size
			}
		}

		// Limit request body size
		r.Body = http.MaxBytesReader(w, r.Body, maxSizeMB*1024*1024)

		next.ServeHTTP(w, r)
	})
}
