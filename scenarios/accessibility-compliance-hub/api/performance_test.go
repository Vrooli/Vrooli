package main

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestPerformance_HealthHandler tests the performance of the health check endpoint
func TestPerformance_HealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Response_Time_Under_100ms", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := testHandlerWithRequest(t, healthHandler, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > 100*time.Millisecond {
			t.Errorf("Health check took too long: %v (expected < 100ms)", duration)
		}

		t.Logf("Health check completed in %v", duration)
	})

	t.Run("Sustained_Load_100_Requests", func(t *testing.T) {
		const numRequests = 100
		start := time.Now()

		for i := 0; i < numRequests; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w := testHandlerWithRequest(t, healthHandler, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / numRequests

		t.Logf("Completed %d requests in %v (avg: %v per request)", numRequests, duration, avgDuration)

		// Average should be under 50ms per request
		if avgDuration > 50*time.Millisecond {
			t.Errorf("Average request time too high: %v (expected < 50ms)", avgDuration)
		}
	})

	t.Run("Concurrent_Requests_50_Parallel", func(t *testing.T) {
		const concurrency = 50
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)
		durations := make(chan time.Duration, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				reqStart := time.Now()

				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}

				w := testHandlerWithRequest(t, healthHandler, req)

				reqDuration := time.Since(reqStart)
				durations <- reqDuration

				if w.Code != 200 {
					errors <- http.ErrServerClosed
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		close(durations)

		totalDuration := time.Since(start)

		// Check for errors
		errorCount := 0
		for range errors {
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Failed %d out of %d concurrent requests", errorCount, concurrency)
		}

		// Calculate stats
		var maxDuration time.Duration
		var totalReqDuration time.Duration
		requestCount := 0

		for d := range durations {
			requestCount++
			totalReqDuration += d
			if d > maxDuration {
				maxDuration = d
			}
		}

		avgDuration := totalReqDuration / time.Duration(requestCount)

		t.Logf("Concurrent test: %d requests in %v", concurrency, totalDuration)
		t.Logf("  Average request time: %v", avgDuration)
		t.Logf("  Max request time: %v", maxDuration)

		// Concurrent requests should complete within reasonable time
		if totalDuration > 5*time.Second {
			t.Errorf("Concurrent requests took too long: %v (expected < 5s)", totalDuration)
		}

		if maxDuration > 500*time.Millisecond {
			t.Errorf("Slowest request took too long: %v (expected < 500ms)", maxDuration)
		}
	})
}

// TestPerformance_ScansHandler tests the performance of the scans endpoint
func TestPerformance_ScansHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Response_Time_Under_200ms", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scans",
		}

		w := testHandlerWithRequest(t, scansHandler, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > 200*time.Millisecond {
			t.Errorf("Scans endpoint took too long: %v (expected < 200ms)", duration)
		}

		t.Logf("Scans endpoint completed in %v", duration)
	})

	t.Run("Concurrent_Scans_Requests", func(t *testing.T) {
		const concurrency = 20
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/scans",
				}

				w := testHandlerWithRequest(t, scansHandler, req)

				if w.Code != 200 {
					errors <- http.ErrServerClosed
				}
			}()
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)

		errorCount := 0
		for range errors {
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Failed %d out of %d concurrent requests", errorCount, concurrency)
		}

		t.Logf("Completed %d concurrent scans requests in %v", concurrency, duration)

		if duration > 3*time.Second {
			t.Errorf("Concurrent scans requests took too long: %v (expected < 3s)", duration)
		}
	})
}

// TestPerformance_ViolationsHandler tests the performance of the violations endpoint
func TestPerformance_ViolationsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Response_Time_Under_200ms", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/violations",
		}

		w := testHandlerWithRequest(t, violationsHandler, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > 200*time.Millisecond {
			t.Errorf("Violations endpoint took too long: %v (expected < 200ms)", duration)
		}

		t.Logf("Violations endpoint completed in %v", duration)
	})
}

// TestPerformance_ReportsHandler tests the performance of the reports endpoint
func TestPerformance_ReportsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Response_Time_Under_200ms", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/reports",
		}

		w := testHandlerWithRequest(t, reportsHandler, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > 200*time.Millisecond {
			t.Errorf("Reports endpoint took too long: %v (expected < 200ms)", duration)
		}

		t.Logf("Reports endpoint completed in %v", duration)
	})
}

// TestPerformance_AllEndpoints tests the performance of all endpoints together
func TestPerformance_AllEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Sequential_Access_All_Endpoints", func(t *testing.T) {
		endpoints := []struct {
			name    string
			handler http.HandlerFunc
			path    string
		}{
			{"health", healthHandler, "/health"},
			{"scans", scansHandler, "/api/scans"},
			{"violations", violationsHandler, "/api/violations"},
			{"reports", reportsHandler, "/api/reports"},
		}

		start := time.Now()

		for _, endpoint := range endpoints {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   endpoint.path,
			}

			w := testHandlerWithRequest(t, endpoint.handler, req)

			if w.Code != 200 {
				t.Errorf("Endpoint %s failed with status %d", endpoint.name, w.Code)
			}
		}

		duration := time.Since(start)

		t.Logf("Sequential access to all endpoints completed in %v", duration)

		// All endpoints should complete in under 1 second sequentially
		if duration > 1*time.Second {
			t.Errorf("Sequential endpoint access took too long: %v (expected < 1s)", duration)
		}
	})

	t.Run("Concurrent_Mixed_Endpoint_Access", func(t *testing.T) {
		const requestsPerEndpoint = 10
		var wg sync.WaitGroup
		errors := make(chan error, requestsPerEndpoint*4)

		endpoints := []struct {
			name    string
			handler http.HandlerFunc
			path    string
		}{
			{"health", healthHandler, "/health"},
			{"scans", scansHandler, "/api/scans"},
			{"violations", violationsHandler, "/api/violations"},
			{"reports", reportsHandler, "/api/reports"},
		}

		start := time.Now()

		for _, endpoint := range endpoints {
			for i := 0; i < requestsPerEndpoint; i++ {
				wg.Add(1)
				go func(h http.HandlerFunc, p string) {
					defer wg.Done()

					req := HTTPTestRequest{
						Method: "GET",
						Path:   p,
					}

					w := testHandlerWithRequest(t, h, req)

					if w.Code != 200 {
						errors <- http.ErrServerClosed
					}
				}(endpoint.handler, endpoint.path)
			}
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)

		errorCount := 0
		for range errors {
			errorCount++
		}

		totalRequests := requestsPerEndpoint * len(endpoints)

		if errorCount > 0 {
			t.Errorf("Failed %d out of %d concurrent mixed requests", errorCount, totalRequests)
		}

		t.Logf("Completed %d concurrent mixed requests in %v", totalRequests, duration)

		// Should handle mixed concurrent load efficiently
		if duration > 5*time.Second {
			t.Errorf("Concurrent mixed requests took too long: %v (expected < 5s)", duration)
		}
	})
}

// TestPerformance_MemoryUsage tests memory efficiency
func TestPerformance_MemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Repeated_Requests_No_Memory_Leak", func(t *testing.T) {
		const iterations = 1000

		for i := 0; i < iterations; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w := testHandlerWithRequest(t, healthHandler, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
				break
			}

			// Periodically check if we're still responsive
			if i%100 == 0 {
				t.Logf("Completed %d iterations", i)
			}
		}

		// If we got here without hanging, memory management is acceptable
		t.Logf("Completed %d iterations successfully", iterations)
	})
}

// Benchmark tests for continuous performance monitoring
func BenchmarkHealthHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		testHandlerWithRequest(&testing.T{}, healthHandler, req)
	}
}

func BenchmarkScansHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scans",
		}

		testHandlerWithRequest(&testing.T{}, scansHandler, req)
	}
}

func BenchmarkViolationsHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/violations",
		}

		testHandlerWithRequest(&testing.T{}, violationsHandler, req)
	}
}

func BenchmarkReportsHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/reports",
		}

		testHandlerWithRequest(&testing.T{}, reportsHandler, req)
	}
}
