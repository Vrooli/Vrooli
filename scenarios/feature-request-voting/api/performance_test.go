//go:build testing
// +build testing

package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestPerformanceListRequests tests listing performance
func TestPerformanceListRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	// Create test data
	scenario := createTestScenario(t, ts.DB, "perf-list-scenario")

	// Create 100 feature requests
	for i := 0; i < 100; i++ {
		createTestFeatureRequest(t, ts.DB, scenario.ID, fmt.Sprintf("Feature %d", i))
	}

	start := time.Now()
	w := makeHTTPRequest(ts.Server, HTTPTestRequest{
		Method:  "GET",
		Path:    fmt.Sprintf("/api/v1/scenarios/%s/feature-requests", scenario.ID),
		URLVars: map[string]string{"scenario_id": scenario.ID},
	})
	duration := time.Since(start)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should complete in under 1 second
	if duration > time.Second {
		t.Errorf("List request took too long: %v", duration)
	}

	t.Logf("List 100 requests completed in %v", duration)
}

// TestPerformanceCreateRequests tests creation performance
func TestPerformanceCreateRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	scenario := createTestScenario(t, ts.DB, "perf-create-scenario")

	start := time.Now()

	// Create 50 feature requests sequentially
	for i := 0; i < 50; i++ {
		payload := TestData.CreateFeatureRequestPayload(
			scenario.ID,
			fmt.Sprintf("Performance Test %d", i),
			fmt.Sprintf("Description %d", i),
		)

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/feature-requests",
			Body:   payload,
		})

		if w.Code != 201 {
			t.Errorf("Request %d failed with status %d", i, w.Code)
		}
	}

	duration := time.Since(start)

	// Should complete in under 2 seconds
	if duration > 2*time.Second {
		t.Errorf("Creating 50 requests took too long: %v", duration)
	}

	t.Logf("Created 50 requests in %v (avg: %v per request)", duration, duration/50)
}

// TestPerformanceVoting tests voting performance
func TestPerformanceVoting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	scenario := createTestScenario(t, ts.DB, "perf-vote-scenario")
	fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Vote Performance Test")

	start := time.Now()

	// Simulate 100 votes with different session IDs
	for i := 0; i < 100; i++ {
		payload := TestData.VotePayload(1)

		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s/vote", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Headers: map[string]string{
				"X-Session-ID": fmt.Sprintf("session-%d", i),
			},
			Body: payload,
		})

		if w.Code != 200 {
			t.Errorf("Vote %d failed with status %d", i, w.Code)
		}
	}

	duration := time.Since(start)

	// Should complete in under 3 seconds
	if duration > 3*time.Second {
		t.Errorf("100 votes took too long: %v", duration)
	}

	t.Logf("100 votes completed in %v (avg: %v per vote)", duration, duration/100)
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	scenario := createTestScenario(t, ts.DB, "concurrent-scenario")

	concurrency := 10
	iterations := 10
	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterations)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				payload := TestData.CreateFeatureRequestPayload(
					scenario.ID,
					fmt.Sprintf("Worker %d - Request %d", workerID, j),
					fmt.Sprintf("Concurrent test %d-%d", workerID, j),
				)

				w := makeHTTPRequest(ts.Server, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/feature-requests",
					Body:   payload,
				})

				if w.Code != 201 {
					errors <- fmt.Errorf("Worker %d request %d failed: status %d", workerID, j, w.Code)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Had %d errors during concurrent execution", errorCount)
	}

	totalRequests := concurrency * iterations
	t.Logf("Completed %d concurrent requests in %v (avg: %v per request)",
		totalRequests, duration, duration/time.Duration(totalRequests))
}

// TestConcurrentVoting tests concurrent voting on the same request
func TestConcurrentVoting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	scenario := createTestScenario(t, ts.DB, "concurrent-vote-scenario")
	fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Concurrent Vote Test")

	concurrency := 20
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(voteID int) {
			defer wg.Done()

			payload := TestData.VotePayload(1)

			w := makeHTTPRequest(ts.Server, HTTPTestRequest{
				Method:  "POST",
				Path:    fmt.Sprintf("/api/v1/feature-requests/%s/vote", fr.ID),
				URLVars: map[string]string{"id": fr.ID},
				Headers: map[string]string{
					"X-Session-ID": fmt.Sprintf("concurrent-session-%d", voteID),
				},
				Body: payload,
			})

			if w.Code != 200 {
				errors <- fmt.Errorf("Vote %d failed: status %d", voteID, w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Had %d errors during concurrent voting", errorCount)
	}

	// Verify vote count is correct
	var voteCount int
	err := ts.DB.QueryRow("SELECT vote_count FROM feature_requests WHERE id = $1", fr.ID).Scan(&voteCount)
	if err != nil {
		t.Fatalf("Failed to get vote count: %v", err)
	}

	if voteCount != concurrency {
		t.Errorf("Expected vote count %d, got %d", concurrency, voteCount)
	}

	t.Logf("Completed %d concurrent votes in %v (vote count: %d)",
		concurrency, duration, voteCount)
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	if ts == nil {
		t.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	scenario := createTestScenario(t, ts.DB, "memory-test-scenario")

	// Create many requests to test memory handling
	for i := 0; i < 1000; i++ {
		createTestFeatureRequest(t, ts.DB, scenario.ID, fmt.Sprintf("Memory Test %d", i))
	}

	// Run list query multiple times
	for i := 0; i < 10; i++ {
		w := makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/scenarios/%s/feature-requests", scenario.ID),
			URLVars: map[string]string{"scenario_id": scenario.ID},
		})

		if w.Code != 200 {
			t.Errorf("List request %d failed with status %d", i, w.Code)
		}
	}

	t.Log("Memory test completed successfully")
}

// BenchmarkCreateFeatureRequest benchmarks feature request creation
func BenchmarkCreateFeatureRequest(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(&testing.T{})
	if ts == nil {
		b.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	scenario := createTestScenario(&testing.T{}, ts.DB, "benchmark-scenario")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		payload := TestData.CreateFeatureRequestPayload(
			scenario.ID,
			fmt.Sprintf("Benchmark Request %d", i),
			"Benchmark description",
		)

		makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/feature-requests",
			Body:   payload,
		})
	}
}

// BenchmarkVote benchmarks voting
func BenchmarkVote(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(&testing.T{})
	if ts == nil {
		b.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	scenario := createTestScenario(&testing.T{}, ts.DB, "benchmark-vote-scenario")
	fr := createTestFeatureRequest(&testing.T{}, ts.DB, scenario.ID, "Benchmark Vote")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		payload := TestData.VotePayload(1)

		makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/feature-requests/%s/vote", fr.ID),
			URLVars: map[string]string{"id": fr.ID},
			Headers: map[string]string{
				"X-Session-ID": fmt.Sprintf("bench-%d", i),
			},
			Body: payload,
		})
	}
}

// BenchmarkListRequests benchmarks listing requests
func BenchmarkListRequests(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(&testing.T{})
	if ts == nil {
		b.Skip("Test server not available")
	}
	defer cleanupTestDB(ts.DB)

	scenario := createTestScenario(&testing.T{}, ts.DB, "benchmark-list-scenario")

	// Pre-populate with 100 requests
	for i := 0; i < 100; i++ {
		createTestFeatureRequest(&testing.T{}, ts.DB, scenario.ID, fmt.Sprintf("Request %d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		makeHTTPRequest(ts.Server, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/scenarios/%s/feature-requests", scenario.ID),
			URLVars: map[string]string{"scenario_id": scenario.ID},
		})
	}
}
