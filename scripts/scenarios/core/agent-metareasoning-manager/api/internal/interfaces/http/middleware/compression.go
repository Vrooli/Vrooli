package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// CompressionMiddleware adds gzip compression support
func CompressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client supports gzip compression
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		
		// Set compression headers
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		
		// Create gzip writer
		gz := gzip.NewWriter(w)
		defer gz.Close()
		
		// Wrap the response writer
		compressed := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gz,
		}
		
		next.ServeHTTP(compressed, r)
	})
}

// gzipResponseWriter wraps http.ResponseWriter with gzip compression
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// Write compresses the data before writing
func (grw *gzipResponseWriter) Write(data []byte) (int, error) {
	return grw.Writer.Write(data)
}

// WriteHeader ensures headers are set before compression
func (grw *gzipResponseWriter) WriteHeader(code int) {
	grw.ResponseWriter.WriteHeader(code)
}