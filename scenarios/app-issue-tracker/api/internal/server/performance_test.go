//go:build testing
// +build testing

package server

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestPerformance_CreateIssues tests bulk issue creation performance
func TestPerformance_CreateIssues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	const numIssues = 100
	start := time.Now()

	for i := 0; i < numIssues; i++ {
		req := HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/issues",
			Body: map[string]interface{}{
				"title": fmt.Sprintf("Performance Test Issue %d", i),
				"targets": []map[string]interface{}{
					{"type": "scenario", "id": "perf-test"},
				},
			},
		}

		w := makeHTTPRequest(env.Server.createIssueHandler, req)
		if w.Code != http.StatusOK {
			t.Errorf("Failed to create issue %d: status %d", i, w.Code)
		}
	}

	duration := time.Since(start)
	avgTime := duration / time.Duration(numIssues)

	t.Logf("Created %d issues in %v (avg: %v per issue)", numIssues, duration, avgTime)

	if avgTime > 100*time.Millisecond {
		t.Logf("WARNING: Average issue creation time %v exceeds target of 100ms", avgTime)
	}
}

// TestPerformance_ConcurrentReads tests concurrent read performance
func TestPerformance_ConcurrentReads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test issues
	const numIssues = 10
	issueIDs := make([]string, numIssues)
	for i := 0; i < numIssues; i++ {
		issueID := fmt.Sprintf("issue-perf-%d", i)
		issue := createTestIssue(issueID, fmt.Sprintf("Perf Test %d", i), "bug", "medium", "perf-test")
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}
		issueIDs[i] = issueID
	}

	// Concurrent reads
	const numReaders = 50
	const readsPerReader = 20

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(numReaders)

	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer wg.Done()

			for j := 0; j < readsPerReader; j++ {
				issueID := issueIDs[j%len(issueIDs)]
				req := HTTPTestRequest{
					Method:  http.MethodGet,
					Path:    "/api/v1/issues/" + issueID,
					URLVars: map[string]string{"id": issueID},
				}

				w := makeHTTPRequest(env.Server.getIssueHandler, req)
				if w.Code != http.StatusOK {
					t.Errorf("Reader %d: Failed to get issue %s: status %d", readerID, issueID, w.Code)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalReads := numReaders * readsPerReader
	avgTime := duration / time.Duration(totalReads)

	t.Logf("Performed %d concurrent reads in %v (avg: %v per read)", totalReads, duration, avgTime)

	if avgTime > 50*time.Millisecond {
		t.Logf("WARNING: Average read time %v exceeds target of 50ms", avgTime)
	}
}

// TestPerformance_ConcurrentWrites tests concurrent write performance
func TestPerformance_ConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	const numWriters = 20
	const writesPerWriter = 5

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(numWriters)

	errors := make(chan error, numWriters*writesPerWriter)

	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			defer wg.Done()

			for j := 0; j < writesPerWriter; j++ {
				req := HTTPTestRequest{
					Method: http.MethodPost,
					Path:   "/api/v1/issues",
					Body: map[string]interface{}{
						"title": fmt.Sprintf("Concurrent Write %d-%d", writerID, j),
						"targets": []map[string]interface{}{
							{"type": "scenario", "id": "concurrent-test"},
						},
					},
				}

				w := makeHTTPRequest(env.Server.createIssueHandler, req)
				if w.Code != http.StatusOK {
					errors <- fmt.Errorf("writer %d: failed to create issue: status %d", writerID, w.Code)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	totalWrites := numWriters * writesPerWriter

	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("%d/%d writes failed", errorCount, totalWrites)
	}

	avgTime := duration / time.Duration(totalWrites)
	t.Logf("Performed %d concurrent writes in %v (avg: %v per write)", totalWrites, duration, avgTime)

	if avgTime > 200*time.Millisecond {
		t.Logf("WARNING: Average write time %v exceeds target of 200ms", avgTime)
	}
}

// TestPerformance_ListIssues tests list performance with large datasets
func TestPerformance_ListIssues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create a large number of issues
	const numIssues = 200
	for i := 0; i < numIssues; i++ {
		issue := createTestIssue(
			fmt.Sprintf("issue-list-perf-%d", i),
			fmt.Sprintf("List Perf Test %d", i),
			"bug",
			"medium",
			"list-perf-test",
		)
		folder := "open"
		if i%3 == 0 {
			folder = "active"
		} else if i%5 == 0 {
			folder = "completed"
		}
		if _, err := env.Server.saveIssue(issue, folder); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}
	}

	start := time.Now()

	req := HTTPTestRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/issues",
	}

	w := makeHTTPRequest(env.Server.getIssuesHandler, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Failed to list issues: status %d", w.Code)
	}

	duration := time.Since(start)

	t.Logf("Listed %d issues in %v", numIssues, duration)

	if duration > 1*time.Second {
		t.Logf("WARNING: List time %v exceeds target of 1s for %d issues", duration, numIssues)
	}
}

// TestPerformance_UpdateOperations tests update performance
func TestPerformance_UpdateOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test issues
	const numIssues = 50
	issueIDs := make([]string, numIssues)
	for i := 0; i < numIssues; i++ {
		issueID := fmt.Sprintf("issue-update-perf-%d", i)
		issue := createTestIssue(issueID, fmt.Sprintf("Update Perf %d", i), "bug", "low", "update-perf")
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}
		issueIDs[i] = issueID
	}

	start := time.Now()

	for _, issueID := range issueIDs {
		req := HTTPTestRequest{
			Method:  http.MethodPut,
			Path:    "/api/v1/issues/" + issueID,
			URLVars: map[string]string{"id": issueID},
			Body: map[string]interface{}{
				"priority": "high",
				"status":   "active",
			},
		}

		w := makeHTTPRequest(env.Server.updateIssueHandler, req)
		if w.Code != http.StatusOK {
			t.Errorf("Failed to update issue %s: status %d", issueID, w.Code)
		}
	}

	duration := time.Since(start)
	avgTime := duration / time.Duration(numIssues)

	t.Logf("Updated %d issues in %v (avg: %v per update)", numIssues, duration, avgTime)

	if avgTime > 150*time.Millisecond {
		t.Logf("WARNING: Average update time %v exceeds target of 150ms", avgTime)
	}
}

// BenchmarkCreateIssue benchmarks issue creation
func BenchmarkCreateIssue(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/issues",
			Body: map[string]interface{}{
				"title": fmt.Sprintf("Benchmark Issue %d", i),
				"targets": []map[string]interface{}{
					{"type": "scenario", "id": "benchmark-test"},
				},
			},
		}

		w := makeHTTPRequest(env.Server.createIssueHandler, req)
		if w.Code != http.StatusOK {
			b.Fatalf("Failed to create issue: status %d", w.Code)
		}
	}
}

// BenchmarkGetIssue benchmarks issue retrieval
func BenchmarkGetIssue(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	// Create a test issue
	issue := createTestIssue("benchmark-get", "Benchmark Get", "bug", "medium", "benchmark")
	if _, err := env.Server.saveIssue(issue, "open"); err != nil {
		b.Fatalf("Failed to create test issue: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method:  http.MethodGet,
			Path:    "/api/v1/issues/benchmark-get",
			URLVars: map[string]string{"id": "benchmark-get"},
		}

		w := makeHTTPRequest(env.Server.getIssueHandler, req)
		if w.Code != http.StatusOK {
			b.Fatalf("Failed to get issue: status %d", w.Code)
		}
	}
}

// BenchmarkListIssues benchmarks issue listing
func BenchmarkListIssues(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	// Create test issues
	for i := 0; i < 50; i++ {
		issue := createTestIssue(
			fmt.Sprintf("benchmark-list-%d", i),
			fmt.Sprintf("Benchmark List %d", i),
			"bug",
			"medium",
			"benchmark",
		)
		if _, err := env.Server.saveIssue(issue, "open"); err != nil {
			b.Fatalf("Failed to create test issue: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/api/v1/issues",
		}

		w := makeHTTPRequest(env.Server.getIssuesHandler, req)
		if w.Code != http.StatusOK {
			b.Fatalf("Failed to list issues: status %d", w.Code)
		}
	}
}
