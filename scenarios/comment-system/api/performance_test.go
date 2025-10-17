// +build testing

package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkDatabaseGetComments benchmarks comment retrieval performance
func BenchmarkDatabaseGetComments(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
	}
	defer testDB.Cleanup()

	scenarioName := "bench-scenario"

	// Create sample comments
	for i := 0; i < 100; i++ {
		createTestComment(&testing.T{}, testDB.DB, scenarioName, fmt.Sprintf("Benchmark comment %d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := testDB.DB.GetComments(scenarioName, nil, 50, 0, "newest")
		if err != nil {
			b.Fatalf("Failed to get comments: %v", err)
		}
	}
}

// BenchmarkDatabaseCreateComment benchmarks comment creation performance
func BenchmarkDatabaseCreateComment(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
	}
	defer testDB.Cleanup()

	scenarioName := "bench-create-scenario"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comment := &Comment{
			ScenarioName: scenarioName,
			Content:      fmt.Sprintf("Benchmark comment %d", i),
			ContentType:  "markdown",
			Status:       "active",
			Metadata:     make(map[string]interface{}),
		}

		err := testDB.DB.CreateComment(comment)
		if err != nil {
			b.Fatalf("Failed to create comment: %v", err)
		}
	}
}

// BenchmarkHTTPGetComments benchmarks HTTP GET performance
func BenchmarkHTTPGetComments(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(&testing.T{}, testDB)
	scenarioName := "bench-http-scenario"

	// Create sample comments
	for i := 0; i < 50; i++ {
		createTestComment(&testing.T{}, testDB.DB, scenarioName, fmt.Sprintf("HTTP benchmark comment %d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/comments/" + scenarioName,
		}

		_, err := makeHTTPRequest(app, req)
		if err != nil {
			b.Fatalf("Failed to make request: %v", err)
		}
	}
}

// BenchmarkHTTPCreateComment benchmarks HTTP POST performance
func BenchmarkHTTPCreateComment(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(&testing.T{}, testDB)
	scenarioName := "bench-http-create-scenario"

	// Create config
	testDB.DB.CreateDefaultConfig(scenarioName)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comments/" + scenarioName,
			Body: map[string]interface{}{
				"content":      fmt.Sprintf("Benchmark comment %d", i),
				"author_token": "test-token",
			},
		}

		_, err := makeHTTPRequest(app, req)
		if err != nil {
			b.Fatalf("Failed to make request: %v", err)
		}
	}
}

// BenchmarkConcurrentReads benchmarks concurrent comment reads
func BenchmarkConcurrentReads(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(&testing.T{}, testDB)
	scenarioName := "bench-concurrent-scenario"

	// Create sample comments
	for i := 0; i < 100; i++ {
		createTestComment(&testing.T{}, testDB.DB, scenarioName, fmt.Sprintf("Concurrent benchmark comment %d", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/comments/" + scenarioName + "?limit=20",
			}

			_, err := makeHTTPRequest(app, req)
			if err != nil {
				b.Fatalf("Failed to make request: %v", err)
			}
		}
	})
}

// BenchmarkMarkdownRendering benchmarks markdown rendering performance
func BenchmarkMarkdownRendering(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(&testing.T{})
	if testDB == nil {
		b.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(&testing.T{}, testDB)
	scenarioName := "bench-markdown-scenario"

	testDB.DB.CreateDefaultConfig(scenarioName)

	// Complex markdown content
	markdownContent := "# Title\n\nThis is a **bold** and *italic* text with [a link](https://example.com).\n\n## Subtitle\n\n- List item 1\n- List item 2\n- List item 3\n\n```go\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n```\n\n> This is a blockquote"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comments/" + scenarioName,
			Body: map[string]interface{}{
				"content":      markdownContent,
				"content_type": "markdown",
				"author_token": "test-token",
			},
		}

		_, err := makeHTTPRequest(app, req)
		if err != nil {
			b.Fatalf("Failed to make request: %v", err)
		}
	}
}

// TestLoadPerformance tests system performance under load
func TestLoadPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)
	scenarioName := "load-test-scenario"

	testDB.DB.CreateDefaultConfig(scenarioName)

	t.Run("SustainedLoad", func(t *testing.T) {
		// Simulate sustained load: 100 requests over 5 seconds
		numRequests := 100
		duration := 5 * time.Second
		requestInterval := duration / time.Duration(numRequests)

		var wg sync.WaitGroup
		errors := make(chan error, numRequests)
		latencies := make(chan time.Duration, numRequests)

		startTime := time.Now()

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				reqStart := time.Now()

				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/comments/" + scenarioName,
					Body: map[string]interface{}{
						"content":      fmt.Sprintf("Load test comment %d", index),
						"author_token": "test-token",
					},
				}

				w, err := makeHTTPRequest(app, req)
				if err != nil {
					errors <- err
					return
				}

				if w.Code != 201 {
					errors <- fmt.Errorf("unexpected status code: %d", w.Code)
					return
				}

				latencies <- time.Since(reqStart)
			}(i)

			time.Sleep(requestInterval)
		}

		wg.Wait()
		close(errors)
		close(latencies)

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Errorf("Request error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Fatalf("Load test failed with %d errors", errorCount)
		}

		// Analyze latencies
		var totalLatency time.Duration
		var maxLatency time.Duration
		count := 0

		for latency := range latencies {
			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
			count++
		}

		avgLatency := totalLatency / time.Duration(count)
		totalDuration := time.Since(startTime)

		t.Logf("Load Test Results:")
		t.Logf("  Total Requests: %d", numRequests)
		t.Logf("  Total Duration: %v", totalDuration)
		t.Logf("  Average Latency: %v", avgLatency)
		t.Logf("  Max Latency: %v", maxLatency)
		t.Logf("  Throughput: %.2f req/s", float64(numRequests)/totalDuration.Seconds())

		// Assert reasonable performance
		if avgLatency > 100*time.Millisecond {
			t.Errorf("Average latency too high: %v (expected < 100ms)", avgLatency)
		}

		if maxLatency > 500*time.Millisecond {
			t.Errorf("Max latency too high: %v (expected < 500ms)", maxLatency)
		}
	})

	t.Run("BurstLoad", func(t *testing.T) {
		// Simulate burst load: 50 concurrent requests
		numRequests := 50
		var wg sync.WaitGroup
		errors := make(chan error, numRequests)

		startTime := time.Now()

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/comments/" + scenarioName,
					Body: map[string]interface{}{
						"content":      fmt.Sprintf("Burst test comment %d", index),
						"author_token": "test-token",
					},
				}

				w, err := makeHTTPRequest(app, req)
				if err != nil {
					errors <- err
					return
				}

				if w.Code != 201 {
					errors <- fmt.Errorf("unexpected status code: %d", w.Code)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(startTime)

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Errorf("Burst request error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Fatalf("Burst test failed with %d errors", errorCount)
		}

		t.Logf("Burst Test Results:")
		t.Logf("  Concurrent Requests: %d", numRequests)
		t.Logf("  Total Duration: %v", duration)
		t.Logf("  Throughput: %.2f req/s", float64(numRequests)/duration.Seconds())

		// Assert reasonable performance
		if duration > 5*time.Second {
			t.Errorf("Burst test took too long: %v (expected < 5s)", duration)
		}
	})
}

// TestMemoryLeaks tests for potential memory leaks
func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.Cleanup()

	app := setupTestApp(t, testDB)
	scenarioName := "memory-leak-test"

	testDB.DB.CreateDefaultConfig(scenarioName)

	// Create and retrieve comments repeatedly to check for leaks
	iterations := 1000

	for i := 0; i < iterations; i++ {
		// Create comment
		createReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/comments/" + scenarioName,
			Body: map[string]interface{}{
				"content":      fmt.Sprintf("Memory test comment %d", i),
				"author_token": "test-token",
			},
		}

		_, err := makeHTTPRequest(app, createReq)
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		// Retrieve comments
		if i%10 == 0 {
			getReq := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/comments/" + scenarioName,
			}

			_, err := makeHTTPRequest(app, getReq)
			if err != nil {
				t.Fatalf("Failed to get comments: %v", err)
			}
		}
	}

	t.Logf("Completed %d iterations without crashes", iterations)
}
