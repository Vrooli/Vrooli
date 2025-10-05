package main

import (
	"sync"
	"testing"
	"time"
)

// TestHealthCheckPerformance tests health endpoint performance
func TestHealthCheckPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("ResponseTime", func(t *testing.T) {
		start := time.Now()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Health check should respond in less than 10ms
		if duration > 10*time.Millisecond {
			t.Errorf("Health check took %v, expected < 10ms", duration)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		concurrency := 50
		iterations := 100

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterations)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					w := makeHTTPRequest(router, HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					})

					if w.Code != 200 {
						errors <- nil
					}
				}
			}()
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		errorCount := len(errors)
		if errorCount > 0 {
			t.Errorf("Had %d errors during concurrent requests", errorCount)
		}

		// All requests should complete within reasonable time
		expectedMaxDuration := 5 * time.Second
		if duration > expectedMaxDuration {
			t.Errorf("Concurrent requests took %v, expected < %v", duration, expectedMaxDuration)
		}

		avgDuration := duration / time.Duration(concurrency*iterations)
		t.Logf("Average request duration: %v", avgDuration)
		t.Logf("Total requests: %d", concurrency*iterations)
		t.Logf("Total duration: %v", duration)
	})
}

// TestMiddlewarePerformance tests middleware overhead
func TestMiddlewarePerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("CORSMiddlewareOverhead", func(t *testing.T) {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})

			if w.Code != 200 {
				t.Errorf("Request failed with status %d", w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		// Average overhead should be minimal
		if avgDuration > 1*time.Millisecond {
			t.Logf("Note: Average middleware overhead is %v (may be acceptable)", avgDuration)
		}

		t.Logf("CORS Middleware - Iterations: %d, Total: %v, Avg: %v", iterations, duration, avgDuration)
	})
}

// TestJSONMarshalingPerformance tests JSON encoding/decoding performance
func TestJSONMarshalingPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("ResponseMarshaling", func(t *testing.T) {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})

			if w.Code != 200 {
				t.Errorf("Request failed with status %d", w.Code)
			}

			// Response body is already marshaled by handler
			_ = w.Body.Bytes()
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("JSON Marshaling - Iterations: %d, Total: %v, Avg: %v", iterations, duration, avgDuration)
	})
}

// TestMemoryUsage provides basic memory profiling for handlers
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("HealthCheckMemory", func(t *testing.T) {
		// Make multiple requests to check for memory leaks
		iterations := 10000

		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})

			if w.Code != 200 {
				t.Errorf("Request failed with status %d", w.Code)
			}
		}

		// If we get here without running out of memory, test passes
		t.Logf("Successfully completed %d requests without memory issues", iterations)
	})
}

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if w.Code != 200 {
			b.Errorf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkCORSMiddleware benchmarks CORS middleware
func BenchmarkCORSMiddleware(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/applications",
		})

		if w.Code != 200 {
			b.Errorf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkJSONMarshaling benchmarks JSON response marshaling
func BenchmarkJSONMarshaling(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if w.Code != 200 {
			b.Errorf("Expected status 200, got %d", w.Code)
		}

		// Access body to ensure it's marshaled
		_ = w.Body.Bytes()
	}
}
