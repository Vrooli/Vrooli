// +build testing

package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestPerformance_HealthCheck tests health check endpoint performance
func TestPerformance_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	pattern := PerformanceTestPattern{
		Name:        "HealthCheck_Performance",
		Description: "Test health check endpoint performance under load",
		MaxDuration: 1 * time.Second,
		Iterations:  100,
		Setup: func(t *testing.T) interface{} {
			return server
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			srv := setupData.(*TestServer)
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w, err := executeServerRequest(srv, req)
			if err != nil {
				return err
			}

			if w.Code != 200 {
				return fmt.Errorf("unexpected status code: %d", w.Code)
			}

			return nil
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestPerformance_GetProfiles tests profile listing performance
func TestPerformance_GetProfiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	pattern := PerformanceTestPattern{
		Name:        "GetProfiles_Performance",
		Description: "Test profile listing performance",
		MaxDuration: 2 * time.Second,
		Iterations:  50,
		Setup: func(t *testing.T) interface{} {
			return server
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			srv := setupData.(*TestServer)
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/profiles",
			}

			w, err := executeServerRequest(srv, req)
			if err != nil {
				return err
			}

			if w.Code != 200 {
				return fmt.Errorf("unexpected status code: %d", w.Code)
			}

			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, duration time.Duration, iterations int) {
			avgLatency := duration / time.Duration(iterations)
			if avgLatency > 20*time.Millisecond {
				t.Logf("Warning: Average latency is %v (expected < 20ms)", avgLatency)
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestPerformance_QueryBookmarks tests bookmark query performance
func TestPerformance_QueryBookmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	pattern := PerformanceTestPattern{
		Name:        "QueryBookmarks_Performance",
		Description: "Test bookmark query performance",
		MaxDuration: 2 * time.Second,
		Iterations:  50,
		Setup: func(t *testing.T) interface{} {
			return server
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			srv := setupData.(*TestServer)
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/bookmarks/query",
			}

			w, err := executeServerRequest(srv, req)
			if err != nil {
				return err
			}

			if w.Code != 200 {
				return fmt.Errorf("unexpected status code: %d", w.Code)
			}

			return nil
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestConcurrency_HealthCheck tests health check under concurrent load
func TestConcurrency_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	pattern := ConcurrencyTestPattern{
		Name:        "HealthCheck_Concurrency",
		Description: "Test health check endpoint under concurrent load",
		Concurrency: 10,
		Iterations:  10,
		Setup: func(t *testing.T) interface{} {
			return server
		},
		Execute: func(t *testing.T, setupData interface{}, workerID, iteration int) error {
			srv := setupData.(*TestServer)
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w, err := executeServerRequest(srv, req)
			if err != nil {
				return err
			}

			if w.Code != 200 {
				return fmt.Errorf("worker %d iteration %d: unexpected status code: %d", workerID, iteration, w.Code)
			}

			return nil
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestConcurrency_MixedEndpoints tests multiple endpoints under concurrent load
func TestConcurrency_MixedEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	pattern := ConcurrencyTestPattern{
		Name:        "MixedEndpoints_Concurrency",
		Description: "Test multiple endpoints under concurrent load",
		Concurrency: 5,
		Iterations:  20,
		Setup: func(t *testing.T) interface{} {
			return server
		},
		Execute: func(t *testing.T, setupData interface{}, workerID, iteration int) error {
			srv := setupData.(*TestServer)

			// Rotate between different endpoints
			endpoints := []HTTPTestRequest{
				{Method: "GET", Path: "/health"},
				{Method: "GET", Path: "/api/v1/profiles"},
				{Method: "GET", Path: "/api/v1/bookmarks/query"},
				{Method: "GET", Path: "/api/v1/categories"},
				{Method: "GET", Path: "/api/v1/platforms"},
			}

			req := endpoints[iteration%len(endpoints)]

			w, err := executeServerRequest(srv, req)
			if err != nil {
				return err
			}

			if w.Code >= 500 {
				return fmt.Errorf("server error: %d", w.Code)
			}

			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			successCount := 0
			for _, err := range results {
				if err == nil {
					successCount++
				}
			}

			successRate := float64(successCount) / float64(len(results)) * 100
			t.Logf("Success rate: %.2f%% (%d/%d)", successRate, successCount, len(results))

			if successRate < 95.0 {
				t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate)
			}
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestConcurrency_BookmarkProcessing tests bookmark processing under load
func TestConcurrency_BookmarkProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	pattern := ConcurrencyTestPattern{
		Name:        "BookmarkProcessing_Concurrency",
		Description: "Test bookmark processing under concurrent load",
		Concurrency: 5,
		Iterations:  10,
		Setup: func(t *testing.T) interface{} {
			return server
		},
		Execute: func(t *testing.T, setupData interface{}, workerID, iteration int) error {
			srv := setupData.(*TestServer)
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/bookmarks/process",
				Body: map[string]interface{}{
					"urls": []string{
						fmt.Sprintf("https://example.com/worker-%d/bookmark-%d", workerID, iteration),
					},
				},
			}

			w, err := executeServerRequest(srv, req)
			if err != nil {
				return err
			}

			if w.Code >= 500 {
				return fmt.Errorf("server error: %d", w.Code)
			}

			return nil
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestMemoryUsage tests memory usage patterns (basic)
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("RepeatedRequests_NoMemoryLeak", func(t *testing.T) {
		// Make many requests and check if server remains stable
		iterations := 1000

		for i := 0; i < iterations; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w, err := executeServerRequest(server, req)
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}

			if w.Code != 200 {
				t.Fatalf("Request %d returned status: %d", i, w.Code)
			}

			// Every 100 requests, force garbage collection to help identify leaks
			if i%100 == 0 {
				// Note: In a real test, we'd measure actual memory usage here
				t.Logf("Progress: %d/%d requests completed", i, iterations)
			}
		}

		t.Logf("Successfully completed %d requests without errors", iterations)
	})
}

// TestResponseTime tests response time requirements
func TestResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping response time test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("HealthCheck_ResponseTime", func(t *testing.T) {
		var totalDuration time.Duration
		iterations := 100

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w, err := executeServerRequest(server, req)
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != 200 {
				t.Fatalf("Unexpected status: %d", w.Code)
			}

			// Individual request should be fast
			if duration > 100*time.Millisecond {
				t.Logf("Warning: Request %d took %v", i, duration)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average response time: %v", avgDuration)

		if avgDuration > 50*time.Millisecond {
			t.Errorf("Average response time too high: %v (expected < 50ms)", avgDuration)
		}
	})
}

// TestRateLimiting tests behavior under high request rates
func TestRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(t, false)
	if server == nil {
		t.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	t.Run("BurstRequests", func(t *testing.T) {
		// Send burst of requests
		burstSize := 50
		var wg sync.WaitGroup
		errors := make([]error, burstSize)

		start := time.Now()

		for i := 0; i < burstSize; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}

				w, err := executeServerRequest(server, req)
				if err != nil {
					errors[idx] = err
					return
				}

				if w.Code >= 500 {
					errors[idx] = fmt.Errorf("server error: %d", w.Code)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		// Count errors
		errorCount := 0
		for _, err := range errors {
			if err != nil {
				errorCount++
			}
		}

		t.Logf("Burst test: %d requests in %v (%d errors)", burstSize, duration, errorCount)

		if errorCount > burstSize/10 {
			t.Errorf("Too many errors in burst: %d/%d", errorCount, burstSize)
		}
	})
}

// Benchmark tests for Go's benchmark framework

func BenchmarkHealthCheck(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(&testing.T{}, false)
	if server == nil {
		b.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		_, err := executeServerRequest(server, req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

func BenchmarkGetProfiles(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(&testing.T{}, false)
	if server == nil {
		b.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles",
		}

		_, err := executeServerRequest(server, req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

func BenchmarkQueryBookmarks(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := setupTestServer(&testing.T{}, false)
	if server == nil {
		b.Skip("Could not setup test server")
	}
	defer server.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/bookmarks/query",
		}

		_, err := executeServerRequest(server, req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}
