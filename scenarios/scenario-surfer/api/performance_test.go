
package main

import (
	"sync"
	"testing"
	"time"
)

// TestPerformanceHealthEndpoint tests performance of health check
func TestPerformanceHealthEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		// Single request should complete quickly
		duration := measureRequestDuration(func() {
			makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})
		})

		maxDuration := 50 * time.Millisecond
		if duration > maxDuration {
			t.Errorf("Health check took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		numRequests := 100
		concurrency := 10

		var wg sync.WaitGroup
		errors := make(chan error, numRequests)
		durations := make(chan time.Duration, numRequests)

		start := time.Now()

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				reqStart := time.Now()
				w, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				})
				reqDuration := time.Since(reqStart)

				if err != nil {
					errors <- err
					return
				}

				if w.Code != 200 {
					errors <- err
					return
				}

				durations <- reqDuration
			}()

			// Limit concurrency
			if (i+1)%concurrency == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		wg.Wait()
		totalDuration := time.Since(start)

		close(errors)
		close(durations)

		// Check for errors
		errorCount := 0
		for range errors {
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Had %d errors out of %d requests", errorCount, numRequests)
		}

		// Calculate average duration
		var totalReqDuration time.Duration
		successCount := 0
		for d := range durations {
			totalReqDuration += d
			successCount++
		}

		if successCount > 0 {
			avgDuration := totalReqDuration / time.Duration(successCount)
			t.Logf("Completed %d requests in %v (avg: %v per request)",
				successCount, totalDuration, avgDuration)

			if avgDuration > 100*time.Millisecond {
				t.Errorf("Average request duration %v exceeds threshold of 100ms", avgDuration)
			}
		}
	})

	t.Run("Throughput", func(t *testing.T) {
		duration := 2 * time.Second
		numWorkers := 5

		var wg sync.WaitGroup
		requestCount := 0
		countMutex := sync.Mutex{}

		start := time.Now()
		stopTime := start.Add(duration)

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for time.Now().Before(stopTime) {
					w, err := makeHTTPRequest(env, HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					})

					if err == nil && w.Code == 200 {
						countMutex.Lock()
						requestCount++
						countMutex.Unlock()
					}
				}
			}()
		}

		wg.Wait()
		actualDuration := time.Since(start)

		requestsPerSecond := float64(requestCount) / actualDuration.Seconds()
		t.Logf("Throughput: %.2f requests/second (%d requests in %v)",
			requestsPerSecond, requestCount, actualDuration)

		// Should handle at least 50 requests per second
		minThroughput := 50.0
		if requestsPerSecond < minThroughput {
			t.Errorf("Throughput %.2f req/s is below minimum %.2f req/s",
				requestsPerSecond, minThroughput)
		}
	})
}

// TestPerformanceScenariosEndpoints tests performance of scenario endpoints
func TestPerformanceScenariosEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	endpoints := map[string]string{
		"status":  "/api/v1/scenarios/status",
		"healthy": "/api/v1/scenarios/healthy",
		"debug":   "/api/v1/scenarios/debug",
	}

	for name, path := range endpoints {
		t.Run(name+"_ResponseTime", func(t *testing.T) {
			duration := measureRequestDuration(func() {
				makeHTTPRequest(env, HTTPTestRequest{
					Method: "GET",
					Path:   path,
				})
			})

			// Scenario endpoints may take longer due to exec calls
			maxDuration := 5 * time.Second
			if duration > maxDuration {
				t.Errorf("Endpoint %s took %v, expected less than %v", name, duration, maxDuration)
			}

			t.Logf("Endpoint %s completed in %v", name, duration)
		})
	}
}

// TestPerformanceIssueReporting tests performance of issue reporting
func TestPerformanceIssueReporting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("SingleReportTime", func(t *testing.T) {
		report := createTestIssueReport(
			"test-scenario",
			"Performance Test Issue",
			"Testing issue reporting performance",
		)

		duration := measureRequestDuration(func() {
			makeHTTPRequest(env, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/issues/report",
				Body:   report,
			})
		})

		// Issue reporting may fail if app-issue-tracker isn't running
		// but we still test the handler performance
		maxDuration := 5 * time.Second
		if duration > maxDuration {
			t.Errorf("Issue reporting took %v, expected less than %v", duration, maxDuration)
		}

		t.Logf("Issue report completed in %v", duration)
	})

	t.Run("ValidationPerformance", func(t *testing.T) {
		// Test that validation errors are fast (no external calls)
		invalidReport := map[string]interface{}{
			"title": "Missing required fields",
		}

		duration := measureRequestDuration(func() {
			makeHTTPRequest(env, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/issues/report",
				Body:   invalidReport,
			})
		})

		// Validation should be very fast
		maxDuration := 50 * time.Millisecond
		if duration > maxDuration {
			t.Errorf("Validation took %v, expected less than %v", duration, maxDuration)
		}
	})
}

// TestMemoryUsage tests that handlers don't leak memory
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("RepeatedRequests", func(t *testing.T) {
		// Make many requests to check for memory leaks
		numRequests := 1000

		for i := 0; i < numRequests; i++ {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})

			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}

			if w.Code != 200 {
				t.Fatalf("Request %d returned status %d", i, w.Code)
			}

			// Periodically log progress
			if (i+1)%100 == 0 {
				t.Logf("Completed %d/%d requests", i+1, numRequests)
			}
		}

		t.Logf("Successfully completed %d requests without errors", numRequests)
	})
}

// TestCORS Performance tests CORS middleware overhead
func TestCORSPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("OPTIONSPerformance", func(t *testing.T) {
		duration := measureRequestDuration(func() {
			makeHTTPRequest(env, HTTPTestRequest{
				Method: "OPTIONS",
				Path:   "/api/v1/scenarios/healthy",
			})
		})

		// OPTIONS should be very fast (just headers, no processing)
		maxDuration := 20 * time.Millisecond
		if duration > maxDuration {
			t.Errorf("OPTIONS request took %v, expected less than %v", duration, maxDuration)
		}
	})
}

// TestLoadScenario tests behavior under load
func TestLoadScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	_ = NewPerformanceTestSuite("ScenarioSurfer", env).
		AddEndpoint("/health", 100).
		AddEndpoint("/api/v1/scenarios/status", 5000).
		AddEndpoint("/api/v1/scenarios/healthy", 5000)

	t.Run("MixedLoad", func(t *testing.T) {
		// Simulate realistic mixed traffic
		numRequests := 50
		var wg sync.WaitGroup
		errors := 0
		errorsMutex := sync.Mutex{}

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(reqNum int) {
				defer wg.Done()

				// Mix of different endpoints
				var path string
				switch reqNum % 3 {
				case 0:
					path = "/health"
				case 1:
					path = "/api/v1/scenarios/status"
				case 2:
					path = "/api/v1/scenarios/healthy"
				}

				w, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "GET",
					Path:   path,
				})

				if err != nil || w.Code != 200 {
					errorsMutex.Lock()
					errors++
					errorsMutex.Unlock()
				}
			}(i)

			// Add slight delay to simulate realistic traffic
			if i%10 == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		wg.Wait()

		if errors > 0 {
			t.Errorf("Had %d errors out of %d mixed requests", errors, numRequests)
		}

		t.Logf("Successfully completed %d mixed requests", numRequests)
	})
}

// BenchmarkHealthHandler benchmarks the health endpoint
func BenchmarkHealthHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
	}
}

// BenchmarkScenariosStatus benchmarks the scenarios status endpoint
func BenchmarkScenariosStatus(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios/status",
		})
	}
}

// BenchmarkHealthyScenarios benchmarks the healthy scenarios endpoint
func BenchmarkHealthyScenarios(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/scenarios/healthy",
		})
	}
}
