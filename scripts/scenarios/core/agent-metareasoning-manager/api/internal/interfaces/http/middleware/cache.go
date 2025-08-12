package middleware

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"metareasoning-api/internal/pkg/interfaces"
)

// CacheMiddleware adds response caching for GET requests
func CacheMiddleware(cache interfaces.CacheManager, ttlSeconds int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests
			if r.Method != "GET" {
				next.ServeHTTP(w, r)
				return
			}
			
			// Skip caching for user-specific endpoints
			if shouldSkipCache(r) {
				next.ServeHTTP(w, r)
				return
			}
			
			// Generate cache key
			cacheKey := generateCacheKey(r)
			
			// Try to get from cache
			if cached, found := cache.Get(cacheKey); found {
				if cachedResponse, ok := cached.(CachedResponse); ok {
					// Check if still valid
					if time.Since(cachedResponse.Timestamp) < time.Duration(ttlSeconds)*time.Second {
						// Add cache headers
						w.Header().Set("X-Cache", "HIT")
						w.Header().Set("X-Cache-Age", fmt.Sprintf("%.0fs", time.Since(cachedResponse.Timestamp).Seconds()))
						
						// Write cached response
						for key, values := range cachedResponse.Headers {
							for _, value := range values {
								w.Header().Add(key, value)
							}
						}
						w.WriteHeader(cachedResponse.StatusCode)
						w.Write(cachedResponse.Body)
						return
					}
				}
			}
			
			// Miss - capture the response
			recorder := &ResponseRecorder{
				ResponseWriter: w,
				Body:          &bytes.Buffer{},
				Headers:       make(http.Header),
			}
			
			next.ServeHTTP(recorder, r)
			
			// Cache successful responses
			if recorder.StatusCode == http.StatusOK {
				cachedResp := CachedResponse{
					StatusCode: recorder.StatusCode,
					Headers:    recorder.Headers,
					Body:       recorder.Body.Bytes(),
					Timestamp:  time.Now(),
				}
				
				cache.Set(cacheKey, cachedResp, time.Duration(ttlSeconds)*time.Second)
			}
			
			// Add cache miss header
			w.Header().Set("X-Cache", "MISS")
		})
	}
}

// CachedResponse represents a cached HTTP response
type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Timestamp  time.Time
}

// ResponseRecorder captures HTTP response for caching
type ResponseRecorder struct {
	http.ResponseWriter
	Body       *bytes.Buffer
	Headers    http.Header
	StatusCode int
}

// Write captures the response body
func (rr *ResponseRecorder) Write(data []byte) (int, error) {
	rr.Body.Write(data)
	return rr.ResponseWriter.Write(data)
}

// WriteHeader captures the status code and headers
func (rr *ResponseRecorder) WriteHeader(code int) {
	rr.StatusCode = code
	
	// Capture headers
	for key, values := range rr.ResponseWriter.Header() {
		rr.Headers[key] = values
	}
	
	rr.ResponseWriter.WriteHeader(code)
}

// Header returns the header map
func (rr *ResponseRecorder) Header() http.Header {
	return rr.ResponseWriter.Header()
}

// shouldSkipCache determines if a request should skip caching
func shouldSkipCache(r *http.Request) bool {
	// Skip user-specific endpoints
	if r.URL.Path == "/workflows" && r.URL.Query().Get("user") != "" {
		return true
	}
	
	// Skip if authorization header present (user-specific data)
	if r.Header.Get("Authorization") != "" && (r.URL.Path == "/workflows" || r.URL.Path == "/stats") {
		return true
	}
	
	return false
}

// generateCacheKey creates a unique cache key for the request
func generateCacheKey(r *http.Request) string {
	// Include method, path, and relevant query parameters
	key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)
	
	// Add query parameters (sorted for consistency)
	if rawQuery := r.URL.RawQuery; rawQuery != "" {
		key += "?" + rawQuery
	}
	
	// Hash to keep keys short
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("%x", hash)
}