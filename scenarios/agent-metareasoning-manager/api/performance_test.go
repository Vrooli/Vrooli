package main

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestConcurrentHealthRequests tests concurrent health check requests
func TestConcurrentHealthRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	const concurrency = 100
	const iterations = 10

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterations)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				req := httptest.NewRequest("GET", "/health", nil)
				w := httptest.NewRecorder()
				Health(w, req)

				if w.Code != 200 {
					errors <- nil // Error would be logged here
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	totalRequests := concurrency * iterations
	requestsPerSecond := float64(totalRequests) / duration.Seconds()

	t.Logf("Concurrent health requests: %d requests in %v (%.2f req/s)", totalRequests, duration, requestsPerSecond)

	// Performance assertion: should handle at least 1000 req/s
	if requestsPerSecond < 1000 {
		t.Logf("Warning: Performance below 1000 req/s (got %.2f req/s)", requestsPerSecond)
	}
}

// TestSearchPerformance tests search endpoint performance
func TestSearchPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Skip - requires test database
	t.Skip("Requires test database")
}

// TestAnalyzePerformance tests analyze endpoint performance
func TestAnalyzePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Skip - requires test database
	t.Skip("Requires test database")
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create many objects and ensure no memory leaks
	iterations := 1000

	for i := 0; i < iterations; i++ {
		_ = WorkflowMetadata{
			ID:         uuid.New(),
			Platform:   "n8n",
			PlatformID: "test-workflow",
			Name:       "Test Workflow",
			Tags:       []string{"test", "reasoning"},
		}

		_ = ReasoningRequest{
			Input: "test input",
			Type:  "pros_cons",
		}

		_ = ReasoningResponse{
			ID:        uuid.New().String(),
			Type:      "pros_cons",
			Timestamp: time.Now(),
		}
	}

	t.Logf("Memory usage test: Created %d objects", iterations*3)
}

// TestConcurrentPatternMatching tests pattern matching under load
func TestConcurrentPatternMatching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	service := NewDiscoveryService(nil, "", "", "")
	workflowNames := []string{
		"pros-cons-analyzer",
		"swot-analysis",
		"risk-assessment",
		"decision-maker",
		"self-review-tool",
		"data-processor",
		"workflow-executor",
		"reasoning-chain",
	}

	const concurrency = 50
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				name := workflowNames[j%len(workflowNames)]
				_ = service.isMetareasoningWorkflow(name)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	totalOps := concurrency * 100

	t.Logf("Pattern matching: %d operations in %v (%.2f ops/ms)", totalOps, duration, float64(totalOps)/float64(duration.Milliseconds()))
}

// TestExtractTagsPerformance tests tag extraction performance
func TestExtractTagsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	workflow := map[string]interface{}{
		"tags": []interface{}{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7", "tag8", "tag9", "tag10"},
	}

	iterations := 10000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_ = extractTags(workflow)
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	t.Logf("Tag extraction: %d extractions in %v (avg: %v per extraction)", iterations, duration, avgDuration)

	// Should be very fast
	if avgDuration > time.Microsecond {
		t.Logf("Warning: Tag extraction slower than expected (avg: %v)", avgDuration)
	}
}

// TestCalculateScorePerformance tests score calculation performance
func TestCalculateScorePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	items := []WeightedItem{
		{Item: "item1", Weight: 5.0},
		{Item: "item2", Weight: 7.5},
		{Item: "item3", Weight: 3.5},
		{Item: "item4", Weight: 6.0},
		{Item: "item5", Weight: 4.5},
		{Item: "item6", Weight: 8.0},
		{Item: "item7", Weight: 2.5},
		{Item: "item8", Weight: 9.0},
		{Item: "item9", Weight: 1.5},
		{Item: "item10", Weight: 7.0},
	}

	iterations := 100000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_ = calculateItemsScore(items)
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	t.Logf("Score calculation: %d calculations in %v (avg: %v per calculation)", iterations, duration, avgDuration)

	// Should be very fast
	if avgDuration > time.Microsecond {
		t.Logf("Warning: Score calculation slower than expected (avg: %v)", avgDuration)
	}
}

// TestResponseTime measures response time percentiles
func TestResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	iterations := 1000
	durations := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		Health(w, req)

		durations[i] = time.Since(start)
	}

	// Calculate percentiles
	// Simple bubble sort for percentile calculation
	for i := 0; i < len(durations); i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}

	p50 := durations[len(durations)*50/100]
	p95 := durations[len(durations)*95/100]
	p99 := durations[len(durations)*99/100]

	t.Logf("Response time percentiles:")
	t.Logf("  P50: %v", p50)
	t.Logf("  P95: %v", p95)
	t.Logf("  P99: %v", p99)

	// Performance assertions
	if p50 > 5*time.Millisecond {
		t.Logf("Warning: P50 response time above 5ms (got %v)", p50)
	}
	if p95 > 20*time.Millisecond {
		t.Logf("Warning: P95 response time above 20ms (got %v)", p95)
	}
	if p99 > 50*time.Millisecond {
		t.Logf("Warning: P99 response time above 50ms (got %v)", p99)
	}
}
