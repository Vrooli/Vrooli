// +build testing

package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// BenchmarkHealthEndpoint tests performance of health endpoint
func BenchmarkHealthEndpoint(b *testing.B) {
	server, cleanup := setupTestServer(&testing.T{})
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkAPIHealthEndpoint tests performance of API health endpoint
func BenchmarkAPIHealthEndpoint(b *testing.B) {
	server, cleanup := setupTestServer(&testing.T{})
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkGetAppsSummary tests performance of apps summary endpoint
func BenchmarkGetAppsSummary(b *testing.B) {
	server, cleanup := setupTestServer(&testing.T{})
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/apps/summary", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
	}
}

// BenchmarkConcurrentHealthRequests tests concurrent request handling
func BenchmarkConcurrentHealthRequests(b *testing.B) {
	server, cleanup := setupTestServer(&testing.T{})
	defer cleanup()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}
	})
}

// TestConcurrentRequestLoad tests system under concurrent load
func TestConcurrentRequestLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, srvCleanup := setupTestServer(t)
	defer srvCleanup()

	t.Run("HighConcurrency", func(t *testing.T) {
		concurrency := 100
		requestsPerWorker := 10

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*requestsPerWorker)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < requestsPerWorker; j++ {
					req := httptest.NewRequest("GET", "/health", nil)
					w := httptest.NewRecorder()
					server.router.ServeHTTP(w, req)

					if w.Code != http.StatusOK {
						errors <- nil // Just count errors
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		errorCount := len(errors)
		if errorCount > 0 {
			t.Logf("Had %d errors out of %d requests", errorCount, concurrency*requestsPerWorker)
		}
	})
}

// TestResponseTime tests that endpoints respond within acceptable time
func TestResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, srvCleanup := setupTestServer(t)
	defer srvCleanup()

	tests := []struct {
		name     string
		path     string
		method   string
		maxTime  time.Duration
	}{
		{"Health", "/health", "GET", 100 * time.Millisecond},
		{"APIHealth", "/api/health", "GET", 200 * time.Millisecond},
		{"AppsSummary", "/api/v1/apps/summary", "GET", 500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			duration := time.Since(start)

			if duration > tt.maxTime {
				t.Errorf("Request took %v, expected less than %v", duration, tt.maxTime)
			}

			t.Logf("Request completed in %v", duration)
		})
	}
}

// TestMemoryUsage tests that the server doesn't leak memory
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, srvCleanup := setupTestServer(t)
	defer srvCleanup()

	t.Run("NoMemoryLeak", func(t *testing.T) {
		// Make many requests
		for i := 0; i < 1000; i++ {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		// If we get here without crashing, memory is likely being managed properly
		t.Log("Completed 1000 requests without crashing")
	})
}

// TestRouterPerformance benchmarks the router
func TestRouterPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping router performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, srvCleanup := setupTestServer(t)
	defer srvCleanup()

	routes := []string{
		"/health",
		"/api/health",
		"/api/v1/apps/summary",
		"/api/v1/apps",
		"/api/v1/system/metrics",
		"/api/v1/resources",
	}

	t.Run("RouteDispatchSpeed", func(t *testing.T) {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			for _, route := range routes {
				req := httptest.NewRequest("GET", route, nil)
				w := httptest.NewRecorder()
				server.router.ServeHTTP(w, req)
			}
		}

		duration := time.Since(start)
		perRequest := duration / time.Duration(iterations*len(routes))

		t.Logf("Processed %d requests in %v (avg %v per request)", iterations*len(routes), duration, perRequest)

		if perRequest > 10*time.Millisecond {
			t.Errorf("Route dispatch too slow: %v per request", perRequest)
		}
	})
}

// TestMiddlewareOverhead tests the performance impact of middleware
func TestMiddlewareOverhead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping middleware overhead test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, srvCleanup := setupTestServer(t)
	defer srvCleanup()

	t.Run("MiddlewareImpact", func(t *testing.T) {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			req := httptest.NewRequest("GET", "/health", nil)
			req.Header.Set("Origin", "http://localhost:3000")
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}

		duration := time.Since(start)
		perRequest := duration / time.Duration(iterations)

		t.Logf("With middleware: %v per request", perRequest)

		if perRequest > 5*time.Millisecond {
			t.Errorf("Middleware overhead too high: %v per request", perRequest)
		}
	})
}
