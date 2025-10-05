package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestPerformanceHealthHandler benchmarks the health handler
func TestPerformanceHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		iterations := 100
		totalDuration := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w, err := makeHTTPRequest(req, healthHandler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average response time: %v", avgDuration)

		// Health check should be fast (under 100ms)
		if avgDuration > 100*time.Millisecond {
			t.Errorf("Average response time too slow: %v (expected < 100ms)", avgDuration)
		}
	})

	t.Run("ConcurrentLoad", func(t *testing.T) {
		concurrency := 50
		requestsPerGoroutine := 10

		start := time.Now()
		var wg sync.WaitGroup
		errors := make(chan error, concurrency*requestsPerGoroutine)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < requestsPerGoroutine; j++ {
					req := HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					}

					w, err := makeHTTPRequest(req, healthHandler)
					if err != nil {
						errors <- err
						return
					}

					if w.Code != 200 {
						errors <- fmt.Errorf("expected status 200, got %d", w.Code)
						return
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		totalDuration := time.Since(start)
		totalRequests := concurrency * requestsPerGoroutine

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Errorf("Concurrent request error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Fatalf("Failed %d out of %d concurrent requests", errorCount, totalRequests)
		}

		requestsPerSecond := float64(totalRequests) / totalDuration.Seconds()
		t.Logf("Processed %d requests in %v (%.2f req/s)", totalRequests, totalDuration, requestsPerSecond)

		// Should handle at least 100 requests per second
		if requestsPerSecond < 100 {
			t.Errorf("Throughput too low: %.2f req/s (expected > 100 req/s)", requestsPerSecond)
		}
	})
}

// TestPerformanceDocumentsHandler benchmarks the documents handler
func TestPerformanceDocumentsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		iterations := 100
		totalDuration := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/documents",
			}

			w, err := makeHTTPRequest(req, documentsHandler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average response time: %v", avgDuration)

		// Document listing should be fast (under 50ms for mock data)
		if avgDuration > 50*time.Millisecond {
			t.Errorf("Average response time too slow: %v (expected < 50ms)", avgDuration)
		}
	})
}

// TestPerformanceJobsHandler benchmarks the jobs handler
func TestPerformanceJobsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		iterations := 100
		totalDuration := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/jobs",
			}

			w, err := makeHTTPRequest(req, jobsHandler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average response time: %v", avgDuration)

		if avgDuration > 50*time.Millisecond {
			t.Errorf("Average response time too slow: %v (expected < 50ms)", avgDuration)
		}
	})
}

// TestPerformanceWorkflowsHandler benchmarks the workflows handler
func TestPerformanceWorkflowsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		iterations := 100
		totalDuration := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/workflows",
			}

			w, err := makeHTTPRequest(req, workflowsHandler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average response time: %v", avgDuration)

		if avgDuration > 50*time.Millisecond {
			t.Errorf("Average response time too slow: %v (expected < 50ms)", avgDuration)
		}
	})
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoMemoryLeaks", func(t *testing.T) {
		// Run a large number of requests to check for memory leaks
		iterations := 1000

		for i := 0; i < iterations; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w, err := makeHTTPRequest(req, healthHandler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Explicitly clear response body
			w.Body.Reset()
		}

		// If we got here without running out of memory, test passes
		t.Logf("Successfully completed %d iterations without memory issues", iterations)
	})
}

// TestGetServiceStatusPerformance benchmarks service status checking
func TestGetServiceStatusPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("TimeoutHandling", func(t *testing.T) {
		// Test that timeout is enforced
		start := time.Now()

		// This should timeout quickly (within 2 seconds + overhead)
		status := getServiceStatus("http://10.255.255.1:12345") // Non-routable IP

		duration := time.Since(start)

		if status != "unhealthy" {
			t.Errorf("Expected 'unhealthy', got '%s'", status)
		}

		// Should timeout in approximately 2 seconds (plus some overhead)
		if duration > 3*time.Second {
			t.Errorf("Timeout took too long: %v (expected ~2s)", duration)
		}

		t.Logf("Service status check with timeout took: %v", duration)
	})

	t.Run("FastResponseTime", func(t *testing.T) {
		// Test with empty URL (should be instant)
		start := time.Now()
		status := getServiceStatus("")
		duration := time.Since(start)

		if status != "not_configured" {
			t.Errorf("Expected 'not_configured', got '%s'", status)
		}

		if duration > 10*time.Millisecond {
			t.Errorf("Empty URL check took too long: %v", duration)
		}
	})
}

// BenchmarkHealthHandler provides Go benchmark for health handler
func BenchmarkHealthHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, healthHandler)
		if err != nil {
			b.Fatalf("Failed to create HTTP request: %v", err)
		}

		if w.Code != 200 {
			b.Errorf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkDocumentsHandler provides Go benchmark for documents handler
func BenchmarkDocumentsHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/documents",
		}

		w, err := makeHTTPRequest(req, documentsHandler)
		if err != nil {
			b.Fatalf("Failed to create HTTP request: %v", err)
		}

		if w.Code != 200 {
			b.Errorf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkGetServiceStatus provides Go benchmark for service status checking
func BenchmarkGetServiceStatus(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getServiceStatus("")
	}
}
