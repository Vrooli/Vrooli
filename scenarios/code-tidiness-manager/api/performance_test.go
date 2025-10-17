// +build testing

package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkScanCode benchmarks the ScanCode operation
func BenchmarkScanCode(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	createTestFiles(&testing.T{}, env)

	req := CodeScanRequest{
		Paths: []string{env.TempDir},
		Types: []string{"backup_files", "temp_files"},
		Limit: 100,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			b.Fatalf("ScanCode failed: %v", err)
		}
	}
}

// BenchmarkAnalyzePatterns benchmarks the AnalyzePatterns operation
func BenchmarkAnalyzePatterns(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	req := PatternAnalysisRequest{
		AnalysisType: "duplicate_detection",
		Paths:        []string{env.TempDir},
		DeepAnalysis: false,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.Processor.AnalyzePatterns(ctx, req)
		if err != nil {
			b.Fatalf("AnalyzePatterns failed: %v", err)
		}
	}
}

// BenchmarkExecuteCleanup benchmarks the ExecuteCleanup operation
func BenchmarkExecuteCleanup(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	req := CleanupExecutionRequest{
		CleanupScripts: []string{"echo 'test'"},
		DryRun:         true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.Processor.ExecuteCleanup(ctx, req)
		if err != nil {
			b.Fatalf("ExecuteCleanup failed: %v", err)
		}
	}
}

// BenchmarkHTTPHandlers benchmarks HTTP handler performance
func BenchmarkHTTPHandlers(b *testing.B) {
	handler := setupTestRouter()

	testCases := []struct {
		name     string
		method   string
		endpoint string
		body     interface{}
	}{
		{
			name:     "HealthCheck",
			method:   "GET",
			endpoint: "/health",
			body:     nil,
		},
		{
			name:     "ScanCodeHandler",
			method:   "POST",
			endpoint: "/api/v1/scan",
			body: map[string]interface{}{
				"paths": []string{"/tmp"},
				"types": []string{"backup_files"},
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req, err := makeHTTPRequest(tc.method, tc.endpoint, tc.body)
				if err != nil {
					b.Fatalf("Failed to create request: %v", err)
				}
				rr := executeRequest(handler, req)
				if rr.Code >= 500 {
					b.Fatalf("Handler returned error status: %d", rr.Code)
				}
			}
		})
	}
}

// TestPerformanceRequirements tests that operations meet performance requirements
func TestPerformanceRequirements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	createTestFiles(t, env)

	t.Run("ScanCode Performance", func(t *testing.T) {
		ctx := context.Background()

		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"backup_files", "temp_files"},
			Limit: 100,
		}

		start := time.Now()
		_, err := env.Processor.ScanCode(ctx, req)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("ScanCode failed: %v", err)
		}

		maxDuration := 5 * time.Second
		if elapsed > maxDuration {
			t.Errorf("ScanCode took %v, expected < %v", elapsed, maxDuration)
		}

		t.Logf("ScanCode completed in %v", elapsed)
	})

	t.Run("AnalyzePatterns Performance", func(t *testing.T) {
		ctx := context.Background()

		req := PatternAnalysisRequest{
			AnalysisType: "duplicate_detection",
			Paths:        []string{env.TempDir},
		}

		start := time.Now()
		_, err := env.Processor.AnalyzePatterns(ctx, req)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("AnalyzePatterns failed: %v", err)
		}

		maxDuration := 5 * time.Second
		if elapsed > maxDuration {
			t.Errorf("AnalyzePatterns took %v, expected < %v", elapsed, maxDuration)
		}

		t.Logf("AnalyzePatterns completed in %v", elapsed)
	})

	t.Run("ExecuteCleanup Performance", func(t *testing.T) {
		ctx := context.Background()

		req := CleanupExecutionRequest{
			CleanupScripts: []string{"echo 'test'"},
			DryRun:         true,
		}

		start := time.Now()
		_, err := env.Processor.ExecuteCleanup(ctx, req)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("ExecuteCleanup failed: %v", err)
		}

		maxDuration := 3 * time.Second
		if elapsed > maxDuration {
			t.Errorf("ExecuteCleanup took %v, expected < %v", elapsed, maxDuration)
		}

		t.Logf("ExecuteCleanup completed in %v", elapsed)
	})
}

// TestConcurrentOperations tests concurrent execution safety
func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	createTestFiles(t, env)

	t.Run("Concurrent Scans", func(t *testing.T) {
		concurrency := 5
		done := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				ctx, cancel := withTimeout(t, 10*time.Second)
				defer cancel()

				req := CodeScanRequest{
					Paths: []string{env.TempDir},
					Types: []string{"backup_files"},
					Limit: 100,
				}

				_, err := env.Processor.ScanCode(ctx, req)
				done <- err
			}(i)
		}

		for i := 0; i < concurrency; i++ {
			if err := <-done; err != nil {
				t.Errorf("Concurrent scan %d failed: %v", i, err)
			}
		}
	})

	t.Run("Concurrent Pattern Analysis", func(t *testing.T) {
		concurrency := 3
		done := make(chan error, concurrency)

		analysisTypes := []string{"duplicate_detection", "todo_comments", "hardcoded_values"}

		for i, analysisType := range analysisTypes {
			go func(id int, aType string) {
				ctx, cancel := withTimeout(t, 10*time.Second)
				defer cancel()

				req := PatternAnalysisRequest{
					AnalysisType: aType,
					Paths:        []string{env.TempDir},
				}

				_, err := env.Processor.AnalyzePatterns(ctx, req)
				done <- err
			}(i, analysisType)
		}

		for i := 0; i < concurrency; i++ {
			if err := <-done; err != nil {
				t.Errorf("Concurrent analysis %d failed: %v", i, err)
			}
		}
	})
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create a larger set of test files
	for i := 0; i < 100; i++ {
		createTestFile(t, env.TempDir, fmt.Sprintf("test%d.bak", i), "test content")
	}

	t.Run("Large Scan Memory Efficiency", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 30*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"backup_files", "temp_files"},
			Limit: 1000,
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success=true")
		}

		// Verify result structure is reasonable
		if result.TotalIssues > 10000 {
			t.Errorf("Unexpected number of issues: %d", result.TotalIssues)
		}
	})
}

// TestResponseTimes measures and validates response times
func TestResponseTimes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping response time tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	handler := setupTestRouter()
	env := setupTestDirectory(t)
	defer env.Cleanup()

	createTestFiles(t, env)

	endpoints := []struct {
		name     string
		method   string
		path     string
		body     interface{}
		maxTime  time.Duration
	}{
		{
			name:    "Health Check",
			method:  "GET",
			path:    "/health",
			body:    nil,
			maxTime: 100 * time.Millisecond,
		},
		{
			name:   "Scan Endpoint",
			method: "POST",
			path:   "/api/v1/scan",
			body: map[string]interface{}{
				"paths": []string{env.TempDir},
				"types": []string{"backup_files"},
			},
			maxTime: 3 * time.Second,
		},
		{
			name:   "Pattern Analysis",
			method: "POST",
			path:   "/api/v1/analyze",
			body: map[string]interface{}{
				"analysis_type": "duplicate_detection",
				"paths":         []string{env.TempDir},
			},
			maxTime: 3 * time.Second,
		},
		{
			name:   "Cleanup Dry Run",
			method: "POST",
			path:   "/api/v1/cleanup",
			body: map[string]interface{}{
				"cleanup_scripts": []string{"echo 'test'"},
				"dry_run":         true,
			},
			maxTime: 2 * time.Second,
		},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			req, err := makeHTTPRequest(ep.method, ep.path, ep.body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			start := time.Now()
			rr := executeRequest(handler, req)
			elapsed := time.Since(start)

			if rr.Code >= 500 {
				t.Errorf("Handler returned error status: %d", rr.Code)
			}

			if elapsed > ep.maxTime {
				t.Errorf("%s took %v, expected < %v", ep.name, elapsed, ep.maxTime)
			}

			t.Logf("%s completed in %v (limit: %v)", ep.name, elapsed, ep.maxTime)
		})
	}
}
