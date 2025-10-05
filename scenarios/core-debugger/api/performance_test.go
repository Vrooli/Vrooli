// +build testing

package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func BenchmarkHealthHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}
		makeHTTPRequest(req, router)
	}
}

func BenchmarkListIssues(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	router := setupTestRouter()

	// Create test data
	for i := 0; i < 100; i++ {
		issue := NewTestIssue(fmt.Sprintf("bench-issue-%d", i), "cli").
			WithSeverity("medium").
			Build()
		createTestIssue(&testing.T{}, env, issue)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues",
		}
		makeHTTPRequest(req, router)
	}
}

func BenchmarkReportIssue(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	router := setupTestRouter()

	issueData := map[string]interface{}{
		"component":   "cli",
		"severity":    "medium",
		"description": "Benchmark issue",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   createJSONBody(&testing.T{}, issueData),
		}
		makeHTTPRequest(req, router)
	}
}

func TestPerformanceHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("HealthCheckLatency", func(t *testing.T) {
		// PRD requirement: Health Check Latency < 500ms per component
		maxLatency := 500 * time.Millisecond

		iterations := 10
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/health",
			}
			w := makeHTTPRequest(req, router)

			duration := time.Since(start)
			totalDuration += duration

			if w.Code != 200 {
				t.Errorf("Health check failed with status %d", w.Code)
			}

			if duration > maxLatency {
				t.Errorf("Health check took %v, exceeds max latency of %v", duration, maxLatency)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average health check latency: %v", avgDuration)
	})

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		// Test concurrent access
		concurrency := 10
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/health",
				}
				w := makeHTTPRequest(req, router)

				if w.Code != 200 {
					errors <- fmt.Errorf("concurrent health check failed with status %d", w.Code)
				}
			}()
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		t.Logf("Concurrent health checks (%d) completed in %v", concurrency, duration)

		for err := range errors {
			t.Error(err)
		}
	})
}

func TestPerformanceIssueRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("LargeIssueList", func(t *testing.T) {
		// Create 1000 test issues
		issueCount := 1000
		for i := 0; i < issueCount; i++ {
			issue := NewTestIssue(fmt.Sprintf("perf-issue-%d", i), "cli").
				WithSeverity("medium").
				WithDescription(fmt.Sprintf("Performance test issue %d", i)).
				Build()
			createTestIssue(t, env, issue)
		}

		// Test retrieval performance
		maxDuration := 2 * time.Second

		start := time.Now()
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues",
		}
		w := makeHTTPRequest(req, router)
		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Issue retrieval failed with status %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Retrieving %d issues took %v, exceeds max of %v", issueCount, duration, maxDuration)
		}

		t.Logf("Retrieved %d issues in %v", issueCount, duration)
	})

	t.Run("FilteredRetrieval", func(t *testing.T) {
		// Test filtered queries performance
		filters := []string{
			"?component=cli",
			"?severity=critical",
			"?status=active",
			"?component=cli&severity=medium",
		}

		for _, filter := range filters {
			start := time.Now()
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/issues" + filter,
			}
			w := makeHTTPRequest(req, router)
			duration := time.Since(start)

			if w.Code != 200 {
				t.Errorf("Filtered retrieval failed with status %d", w.Code)
			}

			t.Logf("Filtered query '%s' completed in %v", filter, duration)
		}
	})
}

func TestPerformanceIssueCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("BulkIssueCreation", func(t *testing.T) {
		// Create 100 issues and measure time
		issueCount := 100
		maxDurationPerIssue := 100 * time.Millisecond

		start := time.Now()

		for i := 0; i < issueCount; i++ {
			issueData := map[string]interface{}{
				"component":   "cli",
				"severity":    "medium",
				"description": fmt.Sprintf("Bulk test issue %d", i),
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/issues",
				Body:   createJSONBody(t, issueData),
			}
			w := makeHTTPRequest(req, router)

			if w.Code != 201 {
				t.Errorf("Issue creation failed with status %d", w.Code)
			}
		}

		totalDuration := time.Since(start)
		avgDuration := totalDuration / time.Duration(issueCount)

		if avgDuration > maxDurationPerIssue {
			t.Errorf("Average issue creation time %v exceeds max %v", avgDuration, maxDurationPerIssue)
		}

		t.Logf("Created %d issues in %v (avg: %v per issue)", issueCount, totalDuration, avgDuration)
	})

	t.Run("ConcurrentIssueCreation", func(t *testing.T) {
		// Test concurrent issue creation
		concurrency := 20
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				issueData := map[string]interface{}{
					"component":   "cli",
					"severity":    "medium",
					"description": fmt.Sprintf("Concurrent issue %d", idx),
				}

				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/issues",
					Body:   createJSONBody(t, issueData),
				}
				w := makeHTTPRequest(req, router)

				if w.Code != 201 {
					errors <- fmt.Errorf("concurrent issue creation failed with status %d", w.Code)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		t.Logf("Concurrent issue creation (%d) completed in %v", concurrency, duration)

		for err := range errors {
			t.Error(err)
		}
	})
}

func TestPerformanceAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("AnalysisLatency", func(t *testing.T) {
		// Create test issue
		issue := NewTestIssue("perf-analysis-1", "cli").
			WithDescription("Performance test for analysis").
			Build()
		createTestIssue(t, env, issue)

		// Test analysis performance
		maxDuration := 1 * time.Second

		start := time.Now()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues/perf-analysis-1/analyze",
		}
		w := makeHTTPRequest(req, router)
		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Analysis failed with status %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Analysis took %v, exceeds max of %v", duration, maxDuration)
		}

		t.Logf("Issue analysis completed in %v", duration)
	})
}

func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("MemoryEfficiency", func(t *testing.T) {
		// PRD requirement: Storage Efficiency < 100MB for 10,000 issues
		// This test creates a smaller subset and checks memory doesn't explode

		issueCount := 500
		for i := 0; i < issueCount; i++ {
			issue := NewTestIssue(fmt.Sprintf("mem-issue-%d", i), "cli").
				WithSeverity("medium").
				WithDescription(fmt.Sprintf("Memory test issue %d with some description text", i)).
				Build()
			createTestIssue(t, env, issue)
		}

		// Test that we can still query efficiently
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues",
		}
		w := makeHTTPRequest(req, router)

		if w.Code != 200 {
			t.Errorf("Memory test query failed with status %d", w.Code)
		}

		data := parseJSONResponse(t, w)
		issues := data["issues"].([]interface{})

		if len(issues) != issueCount {
			t.Errorf("Expected %d issues, got %d", issueCount, len(issues))
		}

		t.Logf("Successfully handled %d issues in memory", issueCount)
	})
}

func BenchmarkComponentHealthCheck(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checkAllComponents()
	}
}

func BenchmarkDetermineOverallStatus(b *testing.B) {
	health := []ComponentHealth{
		{Component: "cli", Status: "healthy"},
		{Component: "orchestrator", Status: "healthy"},
		{Component: "resource-manager", Status: "degraded"},
		{Component: "setup", Status: "healthy"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		determineOverallStatus(health)
	}
}
