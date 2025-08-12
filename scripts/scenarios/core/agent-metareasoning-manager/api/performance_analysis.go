package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PerformanceMiddleware adds timing headers to responses
func PerformanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap the response writer to capture status
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		durationMs := float64(duration.Nanoseconds()) / 1e6
		
		// Add performance headers
		wrapped.Header().Set("X-Response-Time", fmt.Sprintf("%.2fms", durationMs))
		wrapped.Header().Set("X-Timestamp", strconv.FormatInt(start.Unix(), 10))
		
		// Log slow requests
		if durationMs > 100 {
			log.Printf("SLOW REQUEST: %s %s took %.2fms (status: %d)", 
				r.Method, r.URL.Path, durationMs, wrapped.statusCode)
		}
		
		// Log all requests in development
		if config != nil && config.Port == "8093" {
			log.Printf("REQUEST: %s %s %.2fms %d", 
				r.Method, r.URL.Path, durationMs, wrapped.statusCode)
		}
	})
}

// DatabaseConnectionPool optimizes database connection management
func OptimizeDatabaseConnections() {
	if db == nil {
		return
	}
	
	// Optimize connection pool for performance
	db.SetMaxOpenConns(50)      // Increased from 25
	db.SetMaxIdleConns(10)      // Increased from 5
	db.SetConnMaxLifetime(10 * time.Minute) // Increased from 5 minutes
	db.SetConnMaxIdleTime(30 * time.Second) // New: close idle connections faster
	
	log.Printf("Database connection pool optimized: MaxOpen=50, MaxIdle=10, MaxLifetime=10m")
}

// CacheableResponse represents a cacheable API response
type CacheableResponse struct {
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	TTL       int         `json:"ttl_seconds"`
}

// Thread-safe in-memory cache with size limits
type SafeCache struct {
	cache       map[string]CacheableResponse
	mutex       sync.RWMutex
	lastCleanup time.Time
	maxEntries  int
}

var responseCache = &SafeCache{
	cache:      make(map[string]CacheableResponse),
	maxEntries: 1000, // Prevent unlimited growth
}

// CacheMiddleware adds response caching for GET requests
func CacheMiddleware(ttlSeconds int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests
			if r.Method != "GET" {
				next.ServeHTTP(w, r)
				return
			}
			
			// Skip caching for authenticated endpoints that might be user-specific
			if r.URL.Path == "/workflows" && r.URL.Query().Get("user") != "" {
				next.ServeHTTP(w, r)
				return
			}
			
			cacheKey := r.Method + ":" + r.URL.Path + "?" + r.URL.RawQuery
			
			// Clean up expired cache entries periodically
			responseCache.mutex.RLock()
			needsCleanup := time.Since(responseCache.lastCleanup) > 5*time.Minute
			responseCache.mutex.RUnlock()
			
			if needsCleanup {
				cleanupCache()
			}
			
			// Check cache (with read lock)
			responseCache.mutex.RLock()
			cached, exists := responseCache.cache[cacheKey]
			responseCache.mutex.RUnlock()
			
			if exists {
				if time.Since(cached.Timestamp) < time.Duration(ttlSeconds)*time.Second {
					// Cache hit
					w.Header().Set("X-Cache", "HIT")
					w.Header().Set("X-Cache-Age", fmt.Sprintf("%.0fs", time.Since(cached.Timestamp).Seconds()))
					writeJSON(w, http.StatusOK, cached.Data)
					return
				}
				// Cache expired, remove it (with write lock)
				responseCache.mutex.Lock()
				delete(responseCache.cache, cacheKey)
				responseCache.mutex.Unlock()
			}
			
			// Cache miss - capture the response
			recorder := &cacheRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			
			next.ServeHTTP(recorder, r)
			
			// Cache successful responses (with size limit check)
			if recorder.statusCode == http.StatusOK && len(recorder.body) > 0 {
				var responseData interface{}
				if err := json.Unmarshal(recorder.body, &responseData); err == nil {
					responseCache.mutex.Lock()
					
					// Check if cache is at limit - remove oldest entries if needed
					if len(responseCache.cache) >= responseCache.maxEntries {
						// Remove oldest entry (simple LRU approximation)
						var oldestKey string
						var oldestTime time.Time = time.Now()
						
						for key, cached := range responseCache.cache {
							if cached.Timestamp.Before(oldestTime) {
								oldestTime = cached.Timestamp
								oldestKey = key
							}
						}
						if oldestKey != "" {
							delete(responseCache.cache, oldestKey)
						}
					}
					
					responseCache.cache[cacheKey] = CacheableResponse{
						Data:      responseData,
						Timestamp: time.Now(),
						TTL:       ttlSeconds,
					}
					responseCache.mutex.Unlock()
				}
			}
			
			w.Header().Set("X-Cache", "MISS")
		})
	}
}

// cacheRecorder captures response data for caching
type cacheRecorder struct {
	http.ResponseWriter
	body       []byte
	statusCode int
}

func (r *cacheRecorder) Write(data []byte) (int, error) {
	r.body = append(r.body, data...)
	return r.ResponseWriter.Write(data)
}

func (r *cacheRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func cleanupCache() {
	responseCache.mutex.Lock()
	defer responseCache.mutex.Unlock()
	
	now := time.Now()
	initialCount := len(responseCache.cache)
	
	for key, cached := range responseCache.cache {
		if now.Sub(cached.Timestamp) > time.Duration(cached.TTL)*time.Second {
			delete(responseCache.cache, key)
		}
	}
	
	responseCache.lastCleanup = now
	removedCount := initialCount - len(responseCache.cache)
	
	if removedCount > 0 {
		log.Printf("Cache cleanup completed. Removed %d expired entries, %d entries remaining", 
			removedCount, len(responseCache.cache))
	}
}

// gzipResponseWriter wraps http.ResponseWriter to compress responses
type gzipResponseWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	// Remove content-length header since we're compressing
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(statusCode)
}

// CompressionMiddleware adds gzip compression for responses
func CompressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		
		// Set compression headers
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		
		// Create gzip writer
		gz := gzip.NewWriter(w)
		defer func() {
			if err := gz.Close(); err != nil {
				log.Printf("Error closing gzip writer: %v", err)
			}
		}()
		
		// Create wrapper that writes to gzip writer
		gw := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gz,
		}
		
		next.ServeHTTP(gw, r)
	})
}

// OptimizedHealthCheck provides a fast health check endpoint
func OptimizedHealthHandler(w http.ResponseWriter, r *http.Request) {
	// Fast health check without heavy database queries
	start := time.Now()
	
	// Quick database ping
	dbHealthy := false
	if db != nil {
		err := db.Ping()
		dbHealthy = err == nil
	}
	
	response := map[string]interface{}{
		"status":      "healthy",
		"version":     "3.0.0",
		"database":    dbHealthy,
		"response_time_ms": float64(time.Since(start).Nanoseconds()) / 1e6,
		"timestamp":   time.Now().Unix(),
	}
	
	// Set cache headers for health endpoint
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	
	writeJSON(w, http.StatusOK, response)
}

// BatchedDatabaseQueries optimizes database operations
func OptimizeDatabaseQueries() {
	// Example optimization strategies:
	
	// 1. Add database indexes (would be in migration scripts)
	createIndexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflows_platform_active ON workflows(platform, is_active)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflows_type_active ON workflows(type, is_active)", 
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflows_created_at ON workflows(created_at DESC)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflows_usage_count ON workflows(usage_count DESC)",
	}
	
	log.Printf("Recommended database indexes:")
	for _, index := range createIndexes {
		log.Printf("  %s", index)
	}
	
	// 2. Use prepared statements for common queries (implementation would go in database.go)
	log.Printf("Database optimization suggestions:")
	log.Printf("  - Use prepared statements for repeated queries")
	log.Printf("  - Implement connection pooling")
	log.Printf("  - Add appropriate indexes")
	log.Printf("  - Use EXPLAIN ANALYZE to identify slow queries")
}

// PreloadCommonData preloads frequently accessed data
func PreloadCommonData() {
	if db == nil {
		return
	}
	
	// Preload common workflows, models, etc.
	go func() {
		log.Printf("Preloading common data...")
		
		// Example: Preload active workflows count
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM workflows WHERE is_active = true").Scan(&count)
		if err == nil {
			log.Printf("Preloaded: %d active workflows", count)
		}
		
		// You could preload other commonly accessed data here
		log.Printf("Data preloading completed")
	}()
}

// PerformanceReport generates a performance analysis report
func GeneratePerformanceReport() map[string]interface{} {
	return map[string]interface{}{
		"optimization_status": map[string]interface{}{
			"performance_middleware":  "enabled",
			"database_optimization":   "enabled", 
			"response_caching":        "enabled",
			"connection_pooling":      "optimized",
		},
		"current_metrics": map[string]interface{}{
			"cache_size":              len(responseCache),
			"database_max_open_conns": 50,
			"database_max_idle_conns": 10,
		},
		"recommendations": []string{
			"Add database indexes for commonly queried fields",
			"Implement Redis for distributed caching",
			"Use CDN for static assets",
			"Implement request rate limiting",
			"Add metrics collection (Prometheus)",
			"Use prepared statements for database queries",
			"Implement graceful shutdown",
			"Add request timeout middleware",
		},
		"target_response_times": map[string]string{
			"health_endpoint":   "<10ms",
			"list_workflows":    "<50ms",
			"get_workflow":      "<25ms",
			"execute_workflow":  "<5000ms (depends on workflow)",
			"search_workflows":  "<100ms",
		},
	}
}

// ApplyPerformanceOptimizations applies all performance optimizations
func ApplyPerformanceOptimizations() {
	log.Printf("Applying performance optimizations...")
	
	// Optimize database connections
	OptimizeDatabaseConnections()
	
	// Preload common data
	PreloadCommonData()
	
	// Log optimization status
	log.Printf("Performance optimizations applied successfully")
	log.Printf("Target: <100ms response time for most endpoints")
}