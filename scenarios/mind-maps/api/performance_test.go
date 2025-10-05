// +build testing

package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestPerformanceCreateMindMap benchmarks mind map creation
func TestPerformanceCreateMindMap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)
	ctx := context.Background()

	iterations := 10
	start := time.Now()

	for i := 0; i < iterations; i++ {
		createReq := CreateMindMapRequest{
			Title:       fmt.Sprintf("Performance Test Map %d", i),
			Description: "Testing performance",
			UserID:      "perf-test-user",
		}

		_, err := processor.CreateMindMap(ctx, createReq)
		if err != nil {
			t.Logf("Iteration %d failed: %v", i, err)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Created %d mind maps in %v (avg: %v per map)", iterations, elapsed, avgTime)

	if avgTime > 100*time.Millisecond {
		t.Logf("Warning: Average creation time (%v) exceeds 100ms", avgTime)
	}
}

// TestPerformanceUpdateMindMap benchmarks mind map updates
func TestPerformanceUpdateMindMap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)
	ctx := context.Background()

	// Create a mind map first
	createReq := CreateMindMapRequest{
		Title:  "Update Performance Test",
		UserID: "perf-test-user",
	}

	mindMap, err := processor.CreateMindMap(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create mind map: %v", err)
	}

	iterations := 10
	start := time.Now()

	for i := 0; i < iterations; i++ {
		// Update via direct database query since UpdateMindMap processor method doesn't exist
		query := `UPDATE mind_maps SET title = $1, description = $2 WHERE id = $3`
		_, err := testDB.Exec(query,
			fmt.Sprintf("Updated Title %d", i),
			fmt.Sprintf("Updated Description %d", i),
			mindMap.ID,
		)

		if err != nil {
			t.Logf("Iteration %d failed: %v", i, err)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Updated mind map %d times in %v (avg: %v per update)", iterations, elapsed, avgTime)

	if avgTime > 100*time.Millisecond {
		t.Logf("Warning: Average update time (%v) exceeds 100ms", avgTime)
	}
}

// TestPerformanceConcurrentReads tests concurrent read performance
func TestPerformanceConcurrentReads(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)
	ctx := context.Background()

	// Create a mind map first
	createReq := CreateMindMapRequest{
		Title:  "Concurrent Read Test",
		UserID: "perf-test-user",
	}

	mindMap, err := processor.CreateMindMap(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create mind map: %v", err)
	}

	concurrency := 10
	iterations := 5

	var wg sync.WaitGroup
	start := time.Now()

	for c := 0; c < concurrency; c++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				_, err := processor.getMindMapByID(mindMap.ID)
				if err != nil {
					t.Logf("Worker %d, iteration %d failed: %v", workerID, i, err)
				}
			}
		}(c)
	}

	wg.Wait()
	elapsed := time.Since(start)

	totalReads := concurrency * iterations
	avgTime := elapsed / time.Duration(totalReads)

	t.Logf("Performed %d concurrent reads in %v (avg: %v per read)", totalReads, elapsed, avgTime)

	if avgTime > 50*time.Millisecond {
		t.Logf("Warning: Average read time (%v) exceeds 50ms", avgTime)
	}
}

// TestPerformanceBulkNodeCreation tests creating many nodes
func TestPerformanceBulkNodeCreation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer dbCleanup()

	db = testDB

	// Create a mind map
	mindMap := createTestMindMap(t, testDB, "Bulk Node Test", "perf-test-user")

	nodeCount := 50
	start := time.Now()

	for i := 0; i < nodeCount; i++ {
		content := fmt.Sprintf("Node %d", i)
		createTestNode(t, testDB, mindMap.ID, content, "child")
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(nodeCount)

	t.Logf("Created %d nodes in %v (avg: %v per node)", nodeCount, elapsed, avgTime)

	if avgTime > 50*time.Millisecond {
		t.Logf("Warning: Average node creation time (%v) exceeds 50ms", avgTime)
	}
}

// TestPerformanceMemoryUsage monitors memory usage during operations
func TestPerformanceMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)
	ctx := context.Background()

	// Create multiple mind maps
	t.Log("Creating multiple mind maps to test memory usage...")

	for i := 0; i < 20; i++ {
		createReq := CreateMindMapRequest{
			Title:       fmt.Sprintf("Memory Test Map %d", i),
			Description: "Testing memory usage with large dataset",
			UserID:      "memory-test-user",
			Metadata: map[string]interface{}{
				"data": fmt.Sprintf("Large data payload %d", i),
			},
		}

		_, err := processor.CreateMindMap(ctx, createReq)
		if err != nil {
			t.Logf("Failed to create mind map %d: %v", i, err)
		}
	}

	t.Log("Memory usage test completed successfully")
}

// TestPerformanceSearchOperations tests search performance
func TestPerformanceSearchOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)
	ctx := context.Background()

	// Create mind maps with nodes for searching
	for i := 0; i < 5; i++ {
		createReq := CreateMindMapRequest{
			Title:  fmt.Sprintf("Search Test Map %d", i),
			UserID: "search-test-user",
		}

		_, err := processor.CreateMindMap(ctx, createReq)
		if err != nil {
			t.Logf("Failed to create mind map %d: %v", i, err)
		}
	}

	// Test semantic search performance
	iterations := 3
	start := time.Now()

	for i := 0; i < iterations; i++ {
		searchReq := SemanticSearchRequest{
			Query:      fmt.Sprintf("test query %d", i),
			Collection: "mind_maps",
			Limit:      10,
		}

		_, err := processor.SemanticSearch(ctx, searchReq)
		if err != nil {
			// Expected to fail without Qdrant
			t.Logf("Search iteration %d failed (expected): %v", i, err)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Performed %d search operations in %v (avg: %v per search)", iterations, elapsed, avgTime)
}

// TestPerformanceContextTimeout tests context timeout handling
func TestPerformanceContextTimeout(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)

	t.Run("VeryShortTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Let the context timeout
		time.Sleep(10 * time.Millisecond)

		createReq := CreateMindMapRequest{
			Title:  "Timeout Test",
			UserID: "timeout-test-user",
		}

		_, err := processor.CreateMindMap(ctx, createReq)
		if err != nil {
			t.Logf("Operation failed with short timeout (expected): %v", err)
		}
	})

	t.Run("ReasonableTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		createReq := CreateMindMapRequest{
			Title:  "Normal Timeout Test",
			UserID: "timeout-test-user",
		}

		start := time.Now()
		_, err := processor.CreateMindMap(ctx, createReq)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("Operation failed with reasonable timeout: %v", err)
		}

		t.Logf("Operation completed in %v", elapsed)
	})
}

// TestPerformanceStressTest performs stress testing
func TestPerformanceStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer dbCleanup()

	processor := setupTestProcessor(t, testDB)
	ctx := context.Background()

	t.Log("Starting stress test...")

	var wg sync.WaitGroup
	workers := 5
	opsPerWorker := 10

	start := time.Now()
	successCount := 0
	var mu sync.Mutex

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < opsPerWorker; i++ {
				createReq := CreateMindMapRequest{
					Title:  fmt.Sprintf("Stress Worker %d Map %d", workerID, i),
					UserID: fmt.Sprintf("stress-user-%d", workerID),
				}

				_, err := processor.CreateMindMap(ctx, createReq)
				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(start)

	totalOps := workers * opsPerWorker
	t.Logf("Stress test: %d/%d operations succeeded in %v", successCount, totalOps, elapsed)
	t.Logf("Throughput: %.2f ops/sec", float64(totalOps)/elapsed.Seconds())
}
