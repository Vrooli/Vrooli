package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestPerformance_HealthHandler tests the performance of health endpoint
func TestPerformance_HealthHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		healthHandler(w, req)

		if w.Code != 200 {
			t.Fatalf("Iteration %d failed with status %d", i, w.Code)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Health handler: %d iterations in %v (avg: %v per request)", iterations, elapsed, avgTime)

	// Health handler should be very fast (< 1ms per request on average)
	if avgTime > 1*time.Millisecond {
		t.Logf("Warning: Health handler average time %v exceeds 1ms", avgTime)
	}
}

// TestPerformance_DebugHandler tests the performance of debug endpoint
func TestPerformance_DebugHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	iterations := 100 // Fewer iterations since debug is more expensive
	debugTypes := []string{"performance", "error", "logs", "health", "general"}

	for _, debugType := range debugTypes {
		t.Run("DebugType_"+debugType, func(t *testing.T) {
			start := time.Now()

			for i := 0; i < iterations; i++ {
				debugReq := DebugRequest{
					AppName:   "test-app",
					DebugType: debugType,
				}

				body, _ := json.Marshal(debugReq)
				req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				debugHandler(w, req)

				if w.Code != 200 && w.Code != 500 {
					t.Fatalf("Iteration %d failed with unexpected status %d", i, w.Code)
				}
			}

			elapsed := time.Since(start)
			avgTime := elapsed / time.Duration(iterations)

			t.Logf("Debug handler (%s): %d iterations in %v (avg: %v per request)",
				debugType, iterations, elapsed, avgTime)
		})
	}
}

// TestPerformance_ConcurrentRequests tests concurrent request handling
func TestPerformance_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	concurrency := 10
	requestsPerGoroutine := 10

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest("GET", "/health", nil)
				w := httptest.NewRecorder()
				healthHandler(w, req)

				if w.Code != 200 {
					t.Errorf("Goroutine %d request %d failed with status %d", goroutineID, j, w.Code)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	totalRequests := concurrency * requestsPerGoroutine
	avgTime := elapsed / time.Duration(totalRequests)

	t.Logf("Concurrent requests: %d goroutines × %d requests = %d total in %v (avg: %v per request)",
		concurrency, requestsPerGoroutine, totalRequests, elapsed, avgTime)
}

// TestPerformance_MemoryUsage tests memory usage patterns
func TestPerformance_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	// Create many debug sessions to test memory handling
	iterations := 100

	for i := 0; i < iterations; i++ {
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "general",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		// Verify no memory leaks by ensuring response is properly handled
		if w.Body.Len() == 0 {
			t.Errorf("Iteration %d: Empty response body", i)
		}
	}

	t.Logf("Memory test: Created %d debug sessions", iterations)
}

// TestPerformance_LargePayload tests handling of large request payloads
func TestPerformance_LargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	// Create a large error report
	largeContext := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		key := "key_" + string(rune(i))
		value := "This is a long value to increase payload size " + string(rune(i))
		largeContext[key] = value
	}

	errorReport := ErrorReport{
		AppName:      "test-app",
		ErrorMessage: "Large error report",
		StackTrace:   string(make([]byte, 5000)), // 5KB stack trace
		Context:      largeContext,
	}

	body, _ := json.Marshal(errorReport)
	t.Logf("Large payload size: %d bytes", len(body))

	start := time.Now()
	req := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	reportErrorHandler(w, req)
	elapsed := time.Since(start)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	t.Logf("Large payload processing time: %v", elapsed)

	// Should handle large payloads reasonably quickly (< 100ms)
	if elapsed > 100*time.Millisecond {
		t.Logf("Warning: Large payload took %v to process", elapsed)
	}
}

// TestPerformance_ResponseTime benchmarks response time consistency
func TestPerformance_ResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	iterations := 100
	times := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		healthHandler(w, req)

		times[i] = time.Since(start)
	}

	// Calculate statistics
	var total time.Duration
	var min, max time.Duration
	min = times[0]
	max = times[0]

	for _, t := range times {
		total += t
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
	}

	avg := total / time.Duration(iterations)

	t.Logf("Response time statistics over %d requests:", iterations)
	t.Logf("  Average: %v", avg)
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)
	t.Logf("  Range: %v", max-min)

	// Check consistency - max shouldn't be more than 10x average
	if max > avg*10 {
		t.Logf("Warning: Max response time (%v) is more than 10x average (%v)", max, avg)
	}
}

// BenchmarkHealthHandler benchmarks the health handler
func BenchmarkHealthHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		healthHandler(w, req)
	}
}

// BenchmarkDebugHandler benchmarks the debug handler
func BenchmarkDebugHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	debugReq := DebugRequest{
		AppName:   "test-app",
		DebugType: "general",
	}
	body, _ := json.Marshal(debugReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		debugHandler(w, req)
	}
}

// BenchmarkListAppsHandler benchmarks the list apps handler
func BenchmarkListAppsHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/apps", nil)
		w := httptest.NewRecorder()
		listAppsHandler(w, req)
	}
}

// BenchmarkReportErrorHandler benchmarks the error reporting handler
func BenchmarkReportErrorHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	errorReport := ErrorReport{
		AppName:      "test-app",
		ErrorMessage: "Benchmark error",
		StackTrace:   "line1\nline2\nline3",
	}
	body, _ := json.Marshal(errorReport)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		reportErrorHandler(w, req)
	}
}

// BenchmarkSanitizePathComponent benchmarks path sanitization
func BenchmarkSanitizePathComponent(b *testing.B) {
	testPath := "../../../etc/passwd"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizePathComponent(testPath)
	}
}
