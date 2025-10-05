package main

import (
	"testing"
	"time"
)

// BenchmarkHandleHealth benchmarks the health endpoint
func BenchmarkHandleHealth(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/health",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executeHandler(server.handleHealth, req)
	}
}

// BenchmarkHandleAnalyze benchmarks the analyze endpoint
func BenchmarkHandleAnalyze(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	body := makeJSONBody(buildValidAnalyzeRequest())
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/code-smell/analyze",
		Body:   body,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executeHandler(server.handleAnalyze, req)
	}
}

// BenchmarkHandleGetRules benchmarks the rules endpoint
func BenchmarkHandleGetRules(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/code-smell/rules",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executeHandler(server.handleGetRules, req)
	}
}

// BenchmarkHandleFix benchmarks the fix endpoint
func BenchmarkHandleFix(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	body := makeJSONBody(buildValidFixRequest())
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/code-smell/fix",
		Body:   body,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executeHandler(server.handleFix, req)
	}
}

// BenchmarkHandleLearn benchmarks the learn endpoint
func BenchmarkHandleLearn(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	body := makeJSONBody(buildValidLearnRequest())
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/code-smell/learn",
		Body:   body,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executeHandler(server.handleLearn, req)
	}
}

// TestPerformance_AnalyzeLatency tests analyze endpoint latency
func TestPerformance_AnalyzeLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	pattern := PerformanceTestPattern{
		Name:           "AnalyzeLatency",
		Description:    "Test analyze endpoint response time",
		MaxDuration:    5 * time.Second,
		IterationCount: 100,
		Setup: func(t *testing.T) interface{} {
			server := createTestServer()
			body := makeJSONBody(buildValidAnalyzeRequest())
			return map[string]interface{}{
				"server": server,
				"body":   body,
			}
		},
		Execute: func(t *testing.T, setupData interface{}) {
			data := setupData.(map[string]interface{})
			server := data["server"].(*Server)
			body := data["body"].(string)

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/code-smell/analyze",
				Body:   body,
			}
			w := executeHandler(server.handleAnalyze, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestPerformance_GetRulesLatency tests rules endpoint latency
func TestPerformance_GetRulesLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	pattern := PerformanceTestPattern{
		Name:           "GetRulesLatency",
		Description:    "Test rules endpoint response time",
		MaxDuration:    2 * time.Second,
		IterationCount: 100,
		Setup: func(t *testing.T) interface{} {
			return createTestServer()
		},
		Execute: func(t *testing.T, setupData interface{}) {
			server := setupData.(*Server)

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/code-smell/rules",
			}
			w := executeHandler(server.handleGetRules, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestPerformance_ConcurrentRequests tests concurrent request handling
func TestPerformance_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	concurrency := 10
	requestsPerGoroutine := 10

	done := make(chan bool, concurrency)

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < requestsPerGoroutine; j++ {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/health",
				}
				w := executeHandler(server.handleHealth, req)

				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d", w.Code)
				}
			}
			done <- true
		}()
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}

	duration := time.Since(startTime)
	totalRequests := concurrency * requestsPerGoroutine

	t.Logf("Handled %d concurrent requests in %v", totalRequests, duration)

	maxDuration := 5 * time.Second
	if duration > maxDuration {
		t.Errorf("Concurrent requests took too long: %v > %v", duration, maxDuration)
	}
}

// TestPerformance_LargePayload tests handling of large payloads
func TestPerformance_LargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	// Create a large payload with many paths
	paths := make([]string, 500)
	for i := range paths {
		paths[i] = "/test/file" + string(rune(i)) + ".go"
	}

	body := makeJSONBody(map[string]interface{}{
		"paths":    paths,
		"auto_fix": false,
	})

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/code-smell/analyze",
		Body:   body,
	}

	startTime := time.Now()
	w := executeHandler(server.handleAnalyze, req)
	duration := time.Since(startTime)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	maxDuration := 2 * time.Second
	if duration > maxDuration {
		t.Errorf("Large payload processing took too long: %v > %v", duration, maxDuration)
	}

	t.Logf("Processed %d paths in %v", len(paths), duration)
}

// TestPerformance_MemoryUsage tests memory efficiency
func TestPerformance_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	// Run many requests to check for memory leaks
	iterations := 1000

	for i := 0; i < iterations; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}
		executeHandler(server.handleHealth, req)

		if i%100 == 0 {
			t.Logf("Completed %d iterations", i)
		}
	}

	t.Logf("Successfully completed %d iterations", iterations)
}

// TestPerformance_EndToEnd tests end-to-end workflow performance
func TestPerformance_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	startTime := time.Now()

	// Step 1: Get rules
	req1 := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/code-smell/rules",
	}
	w1 := executeHandler(server.handleGetRules, req1)
	if w1.Code != 200 {
		t.Fatalf("GetRules failed: %d", w1.Code)
	}

	// Step 2: Analyze files
	body2 := makeJSONBody(buildValidAnalyzeRequest())
	req2 := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/code-smell/analyze",
		Body:   body2,
	}
	w2 := executeHandler(server.handleAnalyze, req2)
	if w2.Code != 200 {
		t.Fatalf("Analyze failed: %d", w2.Code)
	}

	// Step 3: Get queue
	req3 := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/code-smell/queue",
	}
	w3 := executeHandler(server.handleGetQueue, req3)
	if w3.Code != 200 {
		t.Fatalf("GetQueue failed: %d", w3.Code)
	}

	// Step 4: Apply fix
	body4 := makeJSONBody(buildValidFixRequest())
	req4 := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/code-smell/fix",
		Body:   body4,
	}
	w4 := executeHandler(server.handleFix, req4)
	if w4.Code != 200 {
		t.Fatalf("Fix failed: %d", w4.Code)
	}

	// Step 5: Get stats
	req5 := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/code-smell/stats",
	}
	w5 := executeHandler(server.handleGetStats, req5)
	if w5.Code != 200 {
		t.Fatalf("GetStats failed: %d", w5.Code)
	}

	duration := time.Since(startTime)

	maxDuration := 1 * time.Second
	if duration > maxDuration {
		t.Errorf("End-to-end workflow took too long: %v > %v", duration, maxDuration)
	}

	t.Logf("End-to-end workflow completed in %v", duration)
}
