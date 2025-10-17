package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestRateLimiterPerformance tests rate limiter under load
func TestRateLimiterPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("ConcurrentAccess", func(t *testing.T) {
		rl := NewRateLimiter(1000, time.Minute)
		concurrency := 50
		requestsPerWorker := 100

		startTime := time.Now()
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				defer wg.Done()
				key := fmt.Sprintf("worker-%d", workerID)
				for j := 0; j < requestsPerWorker; j++ {
					rl.Allow(key)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(startTime)

		totalRequests := concurrency * requestsPerWorker
		requestsPerSecond := float64(totalRequests) / duration.Seconds()

		t.Logf("Processed %d requests in %v (%.2f req/s)", totalRequests, duration, requestsPerSecond)

		if requestsPerSecond < 10000 {
			t.Logf("Warning: Rate limiter performance below expected (%.2f req/s < 10000 req/s)", requestsPerSecond)
		}
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		rl := NewRateLimiter(100, time.Minute)

		// Create many keys
		for i := 0; i < 10000; i++ {
			key := fmt.Sprintf("key-%d", i)
			rl.Allow(key)
		}

		// Memory should be manageable
		t.Logf("Rate limiter with 10,000 keys created successfully")
	})

	t.Run("WindowCleanup", func(t *testing.T) {
		rl := NewRateLimiter(5, 100*time.Millisecond)

		// Fill up the rate limiter
		for i := 0; i < 5; i++ {
			rl.Allow("test-key")
		}

		// Wait for window to expire
		time.Sleep(150 * time.Millisecond)

		// Should be able to make requests again
		if !rl.Allow("test-key") {
			t.Error("Rate limiter did not clean up expired requests")
		}
	})
}

// TestHTTPRequestPerformance benchmarks HTTP request handling
func TestHTTPRequestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("MultipleRequests", func(t *testing.T) {
		mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"ok"}`))
		})

		requestCount := 100
		startTime := time.Now()

		for i := 0; i < requestCount; i++ {
			req := createTestHTTPRequest(mockServer.URL, "GET", nil, nil)
			_, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/network/http",
				Body:   req,
			})

			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}
		}

		duration := time.Since(startTime)
		avgDuration := duration / time.Duration(requestCount)

		t.Logf("Processed %d HTTP requests in %v (avg: %v)", requestCount, duration, avgDuration)

		if avgDuration > 100*time.Millisecond {
			t.Logf("Warning: Average request time exceeds 100ms: %v", avgDuration)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"ok"}`))
		})

		concurrency := 10
		requestsPerWorker := 10

		startTime := time.Now()
		var wg sync.WaitGroup
		wg.Add(concurrency)

		errors := make(chan error, concurrency*requestsPerWorker)

		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < requestsPerWorker; j++ {
					req := createTestHTTPRequest(mockServer.URL, "GET", nil, nil)
					_, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
						Method: "POST",
						Path:   "/api/v1/network/http",
						Body:   req,
					})
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(startTime)
		totalRequests := concurrency * requestsPerWorker

		errorCount := 0
		for range errors {
			errorCount++
		}

		t.Logf("Processed %d concurrent requests in %v (%d errors)", totalRequests, duration, errorCount)

		if errorCount > 0 {
			t.Errorf("Expected no errors, got %d", errorCount)
		}
	})
}

// TestDNSQueryPerformance benchmarks DNS query handling
func TestDNSQueryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("MultipleQueries", func(t *testing.T) {
		domains := []string{"google.com", "github.com", "amazon.com", "microsoft.com"}
		requestCount := 20

		startTime := time.Now()

		for i := 0; i < requestCount; i++ {
			domain := domains[i%len(domains)]
			req := createTestDNSRequest(domain, "A")
			_, err := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/network/dns",
				Body:   req,
			})

			if err != nil {
				t.Fatalf("Query %d failed: %v", i, err)
			}
		}

		duration := time.Since(startTime)
		avgDuration := duration / time.Duration(requestCount)

		t.Logf("Processed %d DNS queries in %v (avg: %v)", requestCount, duration, avgDuration)

		if avgDuration > 500*time.Millisecond {
			t.Logf("Warning: Average DNS query time exceeds 500ms: %v", avgDuration)
		}
	})
}

// TestConnectivityPerformance benchmarks connectivity testing
func TestConnectivityPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("MultipleTests", func(t *testing.T) {
		targets := []string{"127.0.0.1", "8.8.8.8"}
		requestCount := 10

		startTime := time.Now()

		for i := 0; i < requestCount; i++ {
			target := targets[i%len(targets)]
			req := createTestConnectivityRequest(target, "ping")
			_, err := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/network/test/connectivity",
				Body:   req,
			})

			if err != nil {
				t.Fatalf("Test %d failed: %v", i, err)
			}
		}

		duration := time.Since(startTime)
		avgDuration := duration / time.Duration(requestCount)

		t.Logf("Processed %d connectivity tests in %v (avg: %v)", requestCount, duration, avgDuration)
	})
}

// BenchmarkRateLimiter benchmarks the rate limiter
func BenchmarkRateLimiter(b *testing.B) {
	rl := NewRateLimiter(10000, time.Minute)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%100)
			rl.Allow(key)
			i++
		}
	})
}

// BenchmarkJSONSerialization benchmarks JSON encoding/decoding
func BenchmarkJSONSerialization(b *testing.B) {
	req := HTTPRequest{
		URL:    "https://example.com",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: map[string]interface{}{
			"key": "value",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(req)
		var decoded HTTPRequest
		json.Unmarshal(data, &decoded)
	}
}

// TestHealthEndpointPerformance tests health endpoint response time
func TestHealthEndpointPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("HealthCheckSpeed", func(t *testing.T) {
		iterations := 100
		maxDuration := 10 * time.Millisecond

		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w, _ := makeHTTPRequest(server.handleHealth, HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})
			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Health check %d failed with status %d", i, w.Code)
			}

			if duration > maxDuration {
				t.Logf("Health check %d took %v (exceeds %v)", i, duration, maxDuration)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average health check time: %v (over %d iterations)", avgDuration, iterations)

		if avgDuration > maxDuration {
			t.Errorf("Average health check time %v exceeds maximum %v", avgDuration, maxDuration)
		}
	})
}
