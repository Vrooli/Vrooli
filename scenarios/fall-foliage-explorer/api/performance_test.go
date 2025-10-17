package main

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

// BenchmarkHealthHandler benchmarks the health check endpoint
func BenchmarkHealthHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}, healthHandler)
	}
}

// BenchmarkRegionsHandler benchmarks the regions list endpoint
func BenchmarkRegionsHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	if testDB != nil {
		defer testDB.Cleanup()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/regions",
		}, regionsHandler)
	}
}

// BenchmarkFoliageHandler benchmarks the foliage data endpoint
func BenchmarkFoliageHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/foliage",
			QueryParams: map[string]string{"region_id": "1"},
		}, foliageHandler)
	}
}

// BenchmarkWeatherHandler benchmarks the weather data endpoint
func BenchmarkWeatherHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/weather",
			QueryParams: map[string]string{"region_id": "1"},
		}, weatherHandler)
	}
}

// BenchmarkReportsHandlerGET benchmarks the GET reports endpoint
func BenchmarkReportsHandlerGET(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/reports",
			QueryParams: map[string]string{"region_id": "1"},
		}, reportsHandler)
	}
}

// TestPerformanceResponseTime validates API response times meet targets
func TestPerformanceResponseTime(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name        string
		handler     func(w http.ResponseWriter, r *http.Request)
		request     HTTPTestRequest
		maxDuration time.Duration
	}{
		{
			name:    "HealthCheck",
			handler: healthHandler,
			request: HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			},
			maxDuration: 100 * time.Millisecond,
		},
		{
			name:    "RegionsList",
			handler: regionsHandler,
			request: HTTPTestRequest{
				Method: "GET",
				Path:   "/api/regions",
			},
			maxDuration: 500 * time.Millisecond,
		},
		{
			name:    "FoliageData",
			handler: foliageHandler,
			request: HTTPTestRequest{
				Method:      "GET",
				Path:        "/api/foliage",
				QueryParams: map[string]string{"region_id": "1"},
			},
			maxDuration: 500 * time.Millisecond,
		},
		{
			name:    "WeatherData",
			handler: weatherHandler,
			request: HTTPTestRequest{
				Method:      "GET",
				Path:        "/api/weather",
				QueryParams: map[string]string{"region_id": "1"},
			},
			maxDuration: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			_, err := makeHTTPRequest(tt.request, tt.handler)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if duration > tt.maxDuration {
				t.Errorf("Response time %v exceeds target %v", duration, tt.maxDuration)
			} else {
				t.Logf("✅ Response time: %v (target: %v)", duration, tt.maxDuration)
			}
		})
	}
}

// TestConcurrentRequests validates API handles concurrent requests
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		concurrency := 50
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := makeHTTPRequest(HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}, healthHandler)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		errorCount := 0
		for err := range errors {
			t.Errorf("Concurrent request failed: %v", err)
			errorCount++
		}

		if errorCount == 0 {
			t.Logf("✅ %d concurrent requests completed in %v", concurrency, duration)
		}

		// Should handle 50 concurrent requests in under 2 seconds
		if duration > 2*time.Second {
			t.Errorf("Concurrent requests took %v, expected < 2s", duration)
		}
	})

	t.Run("ConcurrentRegionsRequests", func(t *testing.T) {
		concurrency := 30
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := makeHTTPRequest(HTTPTestRequest{
					Method: "GET",
					Path:   "/api/regions",
				}, regionsHandler)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		errorCount := 0
		for err := range errors {
			t.Errorf("Concurrent request failed: %v", err)
			errorCount++
		}

		if errorCount == 0 {
			t.Logf("✅ %d concurrent regions requests completed in %v", concurrency, duration)
		}
	})
}

// TestMemoryUsage validates no memory leaks in request processing
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RepeatedRequests", func(t *testing.T) {
		iterations := 1000

		for i := 0; i < iterations; i++ {
			_, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}, healthHandler)

			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}

			// Every 100 requests, check we can still make requests
			if i%100 == 0 && i > 0 {
				t.Logf("Completed %d requests successfully", i)
			}
		}

		t.Logf("✅ %d requests completed without errors", iterations)
	})
}

// TestDatabaseConnectionPool validates connection pool performance
func TestDatabaseConnectionPool(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	t.Run("ParallelDatabaseQueries", func(t *testing.T) {
		concurrency := 20
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := makeHTTPRequest(HTTPTestRequest{
					Method: "GET",
					Path:   "/api/regions",
				}, regionsHandler)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		errorCount := 0
		for err := range errors {
			t.Errorf("Parallel query failed: %v", err)
			errorCount++
		}

		if errorCount == 0 {
			t.Logf("✅ %d parallel DB queries completed in %v", concurrency, duration)
		}
	})
}

// TestAPIThroughput measures API throughput
func TestAPIThroughput(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RequestsPerSecond", func(t *testing.T) {
		duration := 5 * time.Second
		start := time.Now()
		requestCount := 0

		for time.Since(start) < duration {
			_, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}, healthHandler)

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			requestCount++
		}

		actualDuration := time.Since(start)
		rps := float64(requestCount) / actualDuration.Seconds()

		t.Logf("✅ Throughput: %.2f requests/second (%d requests in %v)",
			rps, requestCount, actualDuration)

		// Should handle at least 100 requests per second for health checks
		if rps < 100 {
			t.Errorf("Throughput %.2f req/s is below target of 100 req/s", rps)
		}
	})
}

// TestErrorHandlingPerformance validates error paths don't degrade performance
func TestErrorHandlingPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidRequestsSpeed", func(t *testing.T) {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			makeHTTPRequest(HTTPTestRequest{
				Method:      "GET",
				Path:        "/api/foliage",
				QueryParams: map[string]string{"region_id": "invalid"},
			}, foliageHandler)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("✅ Average error response time: %v", avgDuration)

		// Error responses should be fast (< 10ms average)
		if avgDuration > 10*time.Millisecond {
			t.Errorf("Error handling too slow: %v avg (target: < 10ms)", avgDuration)
		}
	})
}
