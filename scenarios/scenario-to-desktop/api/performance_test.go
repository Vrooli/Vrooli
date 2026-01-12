package main

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestHealthEndpointPerformance tests health check performance
func TestHealthEndpointPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("SequentialRequests", func(t *testing.T) {
		requestCount := 100
		start := time.Now()

		for i := 0; i < requestCount; i++ {
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Completed %d health checks in %v (avg: %v per request)", requestCount, elapsed, avgTime)

		// Performance target: < 10ms average per request
		if avgTime > 10*time.Millisecond {
			t.Logf("WARNING: Average response time (%v) exceeds 10ms target", avgTime)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		requestCount := 100
		concurrency := 10

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < requestCount/concurrency; j++ {
					req := httptest.NewRequest("GET", "/api/v1/health", nil)
					w := httptest.NewRecorder()
					server.router.ServeHTTP(w, req)

					if w.Code != 200 {
						t.Errorf("Worker %d request %d failed", workerID, j)
					}
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Completed %d concurrent health checks in %v (avg: %v per request)", requestCount, elapsed, avgTime)

		// Performance target: < 5ms average with concurrency
		if avgTime > 5*time.Millisecond {
			t.Logf("WARNING: Average concurrent response time (%v) exceeds 5ms target", avgTime)
		}
	})

	t.Run("SustainedLoad", func(t *testing.T) {
		duration := 5 * time.Second
		var requestCount int
		var failCount int
		var mu sync.Mutex

		start := time.Now()
		done := make(chan bool)

		// Spawn multiple workers
		workerCount := 5
		for i := 0; i < workerCount; i++ {
			go func() {
				for {
					select {
					case <-done:
						return
					default:
						req := httptest.NewRequest("GET", "/api/v1/health", nil)
						w := httptest.NewRecorder()
						server.router.ServeHTTP(w, req)

						mu.Lock()
						requestCount++
						if w.Code != 200 {
							failCount++
						}
						mu.Unlock()
					}
				}
			}()
		}

		// Run for specified duration
		time.Sleep(duration)
		close(done)
		elapsed := time.Since(start)

		// Allow workers to finish
		time.Sleep(100 * time.Millisecond)

		t.Logf("Sustained load: %d requests in %v (%.0f req/s, %d failures)",
			requestCount, elapsed, float64(requestCount)/elapsed.Seconds(), failCount)

		if failCount > 0 {
			t.Errorf("Expected zero failures under sustained load, got %d", failCount)
		}
	})
}

// TestStatusEndpointPerformance tests status endpoint performance
func TestStatusEndpointPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("SequentialRequests", func(t *testing.T) {
		requestCount := 50
		start := time.Now()

		for i := 0; i < requestCount; i++ {
			req := httptest.NewRequest("GET", "/api/v1/status", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Completed %d status checks in %v (avg: %v per request)", requestCount, elapsed, avgTime)

		// Performance target: < 20ms average per request (more complex than health)
		if avgTime > 20*time.Millisecond {
			t.Logf("WARNING: Average response time (%v) exceeds 20ms target", avgTime)
		}
	})
}

// TestTemplateListingPerformance tests template listing performance
func TestTemplateListingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SequentialRequests", func(t *testing.T) {
		requestCount := 30
		start := time.Now()

		for i := 0; i < requestCount; i++ {
			req := httptest.NewRequest("GET", "/api/v1/templates", nil)
			w := httptest.NewRecorder()
			env.Server.router.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Completed %d template listings in %v (avg: %v per request)", requestCount, elapsed, avgTime)

		// Performance target: < 15ms average per request
		if avgTime > 15*time.Millisecond {
			t.Logf("WARNING: Average response time (%v) exceeds 15ms target", avgTime)
		}
	})
}

// TestValidationPerformance tests configuration validation performance
func TestValidationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)
	baseConfig := func(name string) *DesktopConfig {
		return &DesktopConfig{
			AppName:      name,
			Framework:    "electron",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
			ServerType:   "static",
			ServerPath:   "./dist",
			APIEndpoint:  "http://localhost:3000",
		}
	}

	t.Run("ValidConfigValidation", func(t *testing.T) {
		validationCount := 10000
		start := time.Now()

		for i := 0; i < validationCount; i++ {
			config := baseConfig(fmt.Sprintf("App%d", i))

			err := server.validateDesktopConfig(config)
			if err != nil {
				t.Errorf("Validation %d failed: %v", i, err)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(validationCount)

		t.Logf("Completed %d validations in %v (avg: %v per validation)", validationCount, elapsed, avgTime)

		// Performance target: < 1µs average per validation
		if avgTime > 1*time.Microsecond {
			t.Logf("Average validation time: %v", avgTime)
		}
	})

	t.Run("InvalidConfigValidation", func(t *testing.T) {
		validationCount := 1000
		start := time.Now()

		for i := 0; i < validationCount; i++ {
			config := baseConfig(fmt.Sprintf("App%d", i))
			config.Framework = "invalid"

			err := server.validateDesktopConfig(config)
			if err == nil {
				t.Errorf("Expected validation %d to fail", i)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(validationCount)

		t.Logf("Completed %d invalid validations in %v (avg: %v per validation)", validationCount, elapsed, avgTime)
	})
}

// TestWebhookPerformance tests webhook handling performance
// NOTE: This test validates webhook endpoint exists and responds correctly.
// Detailed webhook testing with build state manipulation is done in build/handler_test.go
func TestWebhookPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("WebhookEndpointResponds", func(t *testing.T) {
		requestCount := 50
		start := time.Now()

		for i := 0; i < requestCount; i++ {
			body := map[string]interface{}{
				"status": "completed",
			}

			req := createJSONRequest("POST", "/api/v1/desktop/webhook/build-complete", body)
			req.Header.Set("X-Build-ID", fmt.Sprintf("test-build-%d", i))
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// Expect 404 for non-existent builds, which confirms route works
			if w.Code != 200 && w.Code != 404 {
				t.Errorf("Webhook request %d returned unexpected status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Processed %d webhook requests in %v (avg: %v per webhook)", requestCount, elapsed, avgTime)

		// Performance target: < 10ms average per webhook
		if avgTime > 10*time.Millisecond {
			t.Logf("WARNING: Average webhook time (%v) exceeds 10ms target", avgTime)
		}
	})
}
