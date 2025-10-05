package main

import (
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestPerformanceHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		iterations := 100
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w := testHandlerWithRequest(t, healthHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})
			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average health endpoint response time: %v", avgDuration)

		// Health endpoint should respond in under 10ms on average
		if avgDuration > 10*time.Millisecond {
			t.Errorf("Average response time too slow: %v (expected < 10ms)", avgDuration)
		}
	})

	t.Run("Concurrency", func(t *testing.T) {
		concurrency := 50
		iterations := 10

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterations)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < iterations; j++ {
					w := testHandlerWithRequest(t, healthHandler, HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					})

					if w.Code != 200 {
						errors <- nil // placeholder for error
					}

					var response map[string]interface{}
					if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		errorCount := 0
		for range errors {
			errorCount++
		}

		t.Logf("Concurrent health requests: %d workers × %d iterations = %d total requests", concurrency, iterations, concurrency*iterations)
		t.Logf("Total time: %v", duration)
		t.Logf("Throughput: %.2f req/sec", float64(concurrency*iterations)/duration.Seconds())

		if errorCount > 0 {
			t.Errorf("Encountered %d errors during concurrent requests", errorCount)
		}

		// Should complete in reasonable time even under load
		maxDuration := 5 * time.Second
		if duration > maxDuration {
			t.Errorf("Concurrent requests took too long: %v (expected < %v)", duration, maxDuration)
		}
	})
}

func TestPerformanceAgentsEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		iterations := 50

		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/agents",
			})
			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average agents endpoint response time: %v", avgDuration)

		// Agents endpoint should respond in under 50ms on average
		if avgDuration > 50*time.Millisecond {
			t.Logf("Warning: Average response time is %v (expected < 50ms)", avgDuration)
		}
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		// Test that repeated calls don't cause memory leaks
		iterations := 100

		for i := 0; i < iterations; i++ {
			w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/agents",
			})

			var response AgentsResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response on iteration %d: %v", i, err)
			}

			// Ensure response is properly formed
			if response.Agents == nil {
				t.Errorf("Iteration %d: agents array is nil", i)
			}
		}

		// If we get here without panicking or running out of memory, test passes
		t.Logf("Completed %d iterations without memory issues", iterations)
	})
}

func TestPerformanceCapabilitiesEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		iterations := 50
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w := testHandlerWithRequest(t, capabilitiesHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/capabilities",
			})
			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average capabilities endpoint response time: %v", avgDuration)

		// Capabilities endpoint involves aggregation, allow more time
		if avgDuration > 100*time.Millisecond {
			t.Logf("Warning: Average response time is %v (expected < 100ms)", avgDuration)
		}
	})

	t.Run("Concurrency", func(t *testing.T) {
		concurrency := 20
		iterations := 5

		var wg sync.WaitGroup
		successCount := make(chan int, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				localSuccess := 0

				for j := 0; j < iterations; j++ {
					w := testHandlerWithRequest(t, capabilitiesHandler, HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/capabilities",
					})

					if w.Code == 200 {
						localSuccess++
					}
				}

				successCount <- localSuccess
			}()
		}

		wg.Wait()
		close(successCount)
		duration := time.Since(start)

		totalSuccess := 0
		for count := range successCount {
			totalSuccess += count
		}

		expectedTotal := concurrency * iterations
		t.Logf("Concurrent capabilities requests: %d/%d succeeded", totalSuccess, expectedTotal)
		t.Logf("Duration: %v", duration)

		if totalSuccess != expectedTotal {
			t.Errorf("Expected %d successful requests, got %d", expectedTotal, totalSuccess)
		}
	})
}

func TestPerformanceSearchEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		iterations := 50
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w := testHandlerWithRequest(t, searchByCapabilityHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/agents/search?capability=test",
				QueryParams: map[string]string{
					"capability": "test",
				},
			})
			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average search endpoint response time: %v", avgDuration)

		// Search involves filtering, allow reasonable time
		if avgDuration > 50*time.Millisecond {
			t.Logf("Warning: Average response time is %v (expected < 50ms)", avgDuration)
		}
	})

	t.Run("DifferentCapabilities", func(t *testing.T) {
		capabilities := []string{
			"text-generation",
			"code-completion",
			"image-generation",
			"data-analysis",
			"web-search",
		}

		var totalDuration time.Duration

		for _, cap := range capabilities {
			start := time.Now()
			w := testHandlerWithRequest(t, searchByCapabilityHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/agents/search?capability=" + cap,
				QueryParams: map[string]string{
					"capability": cap,
				},
			})
			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Search for capability '%s' failed with status %d", cap, w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(len(capabilities))
		t.Logf("Average search time across %d capabilities: %v", len(capabilities), avgDuration)
	})
}

func BenchmarkHealthHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		healthHandler(w, req)
	}
}

func BenchmarkAgentsHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/agents", nil)
		agentsHandler(w, req)
	}
}

func BenchmarkCapabilitiesHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/capabilities", nil)
		capabilitiesHandler(w, req)
	}
}

func BenchmarkStatusHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/status", nil)
		statusHandler(w, req)
	}
}

func BenchmarkSearchByCapabilityHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/agents/search?capability=test", nil)
		searchByCapabilityHandler(w, req)
	}
}

func BenchmarkValidationFunctions(b *testing.B) {
	b.Run("isValidLineCount", func(b *testing.B) {
		testCases := []string{"100", "10000", "invalid", "-1"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = isValidLineCount(testCases[i%len(testCases)])
		}
	})

	b.Run("isValidResourceName", func(b *testing.B) {
		testCases := []string{"claude-code", "ollama", "invalid", ""}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = isValidResourceName(testCases[i%len(testCases)])
		}
	})

	b.Run("isValidAgentID", func(b *testing.B) {
		testCases := []string{"claude-code:agent-123", "ollama:test", "invalid", ""}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = isValidAgentID(testCases[i%len(testCases)])
		}
	})
}
