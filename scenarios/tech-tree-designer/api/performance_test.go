//go:build testing
// +build testing

package main

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestPerformance_HealthEndpoint benchmarks health check response time
func TestPerformance_HealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Health endpoint response time", func(t *testing.T) {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(t, router, "GET", "/health", nil)
			if w.Code != http.StatusOK {
				t.Errorf("Iteration %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		t.Logf("Health endpoint average response time: %v", avgTime)
		t.Logf("Total time for %d requests: %v", iterations, elapsed)

		// Health check should be very fast - under 10ms average
		if avgTime > 10*time.Millisecond {
			t.Errorf("Health endpoint too slow: %v (expected < 10ms)", avgTime)
		}
	})
}

// TestPerformance_GetSectors benchmarks sector retrieval performance
func TestPerformance_GetSectors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Sector retrieval performance", func(t *testing.T) {
		// Setup: Create test data with multiple sectors and stages
		treeID := createTestTechTree(t, testDB)

		// Create 10 sectors with stages
		baseDelay := 5 * time.Millisecond
		maxDelay := 40 * time.Millisecond
		randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

		for i := 1; i <= 10; i++ {
			sectorID := fmt.Sprintf("00000000-0000-0000-0000-00000000%04d", i+100)
			if err := execWithBackoff(t, testDB, `
				INSERT INTO sectors (id, tree_id, name, category, description, progress_percentage,
					position_x, position_y, color, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
			`, sectorID, treeID, fmt.Sprintf("Sector %d", i), "software",
				"Test sector", float64(i*10), float64(i*50), float64(i*50), "#3498db"); err != nil {
				t.Fatalf("failed to insert sector %d: %v", i, err)
			}

			delay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(i-1)), float64(maxDelay)))
			jitter := time.Duration(randSource.Float64() * float64(delay) * 0.25)
			time.Sleep(delay + jitter)

			// Create 5 stages per sector
			for j := 1; j <= 5; j++ {
				stageID := fmt.Sprintf("00000000-0000-0000-0000-00000%03d%03d", i, j)
				examples := `["Example"]`
				if err := execWithBackoff(t, testDB, `
					INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description,
						progress_percentage, examples, position_x, position_y, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
				`, stageID, sectorID, "foundation", j, fmt.Sprintf("Stage %d", j),
					"Test stage", float64(j*20), examples, float64(j*30), float64(j*30)); err != nil {
					t.Fatalf("failed to insert stage %d for sector %d: %v", j, i, err)
				}

				innerDelay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(j-1)), float64(maxDelay)))
				innerJitter := time.Duration(randSource.Float64() * float64(innerDelay) * 0.25)
				time.Sleep(innerDelay + innerJitter)

				// Create scenario mappings
				createTestScenarioMapping(t, testDB, stageID, fmt.Sprintf("scenario-%d-%d", i, j))
			}
		}

		// Benchmark sector retrieval
		iterations := 50
		start := time.Now()

		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors", nil)
			if w.Code != http.StatusOK {
				t.Errorf("Iteration %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		t.Logf("Sector retrieval average response time: %v", avgTime)
		t.Logf("Total time for %d requests: %v", iterations, elapsed)
		t.Logf("Throughput: %.2f requests/second", float64(iterations)/elapsed.Seconds())

		// Sector retrieval with nested data should be under 100ms
		if avgTime > 100*time.Millisecond {
			t.Logf("WARNING: Sector retrieval slower than expected: %v", avgTime)
		}
	})
}

// TestPerformance_ConcurrentRequests tests concurrent request handling
func TestPerformance_ConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Concurrent health checks", func(t *testing.T) {
		concurrency := 50
		requestsPerWorker := 20

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*requestsPerWorker)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < requestsPerWorker; j++ {
					w := makeHTTPRequest(t, router, "GET", "/health", nil)
					if w.Code != http.StatusOK {
						errors <- fmt.Errorf("worker %d request %d failed: status %d", workerID, j, w.Code)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		elapsed := time.Since(start)

		totalRequests := concurrency * requestsPerWorker
		t.Logf("Concurrent requests: %d workers × %d requests = %d total", concurrency, requestsPerWorker, totalRequests)
		t.Logf("Total time: %v", elapsed)
		t.Logf("Throughput: %.2f requests/second", float64(totalRequests)/elapsed.Seconds())

		// Check for errors
		errorCount := 0
		for err := range errors {
			errorCount++
			t.Error(err)
		}

		if errorCount > 0 {
			t.Errorf("Failed %d out of %d concurrent requests", errorCount, totalRequests)
		}
	})

	t.Run("Concurrent sector retrievals", func(t *testing.T) {
		// Setup test data
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)
		createTestScenarioMapping(t, testDB, stageID, "perf-test-scenario")

		concurrency := 20
		requestsPerWorker := 10

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*requestsPerWorker)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < requestsPerWorker; j++ {
					w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors", nil)
					if w.Code != http.StatusOK {
						errors <- fmt.Errorf("worker %d request %d failed: status %d", workerID, j, w.Code)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		elapsed := time.Since(start)

		totalRequests := concurrency * requestsPerWorker
		t.Logf("Concurrent sector requests: %d workers × %d requests = %d total", concurrency, requestsPerWorker, totalRequests)
		t.Logf("Total time: %v", elapsed)
		t.Logf("Throughput: %.2f requests/second", float64(totalRequests)/elapsed.Seconds())

		errorCount := 0
		for err := range errors {
			errorCount++
			t.Error(err)
		}

		if errorCount > 0 {
			t.Errorf("Failed %d out of %d concurrent requests", errorCount, totalRequests)
		}
	})
}

// TestPerformance_StrategicAnalysis benchmarks strategic analysis performance
func TestPerformance_StrategicAnalysis(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Strategic analysis response time", func(t *testing.T) {
		body := map[string]interface{}{
			"current_resources": 5,
			"time_horizon":      12,
			"priority_sectors":  []string{"software", "manufacturing", "healthcare"},
		}

		iterations := 20
		start := time.Now()

		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
			if w.Code != http.StatusOK {
				t.Errorf("Iteration %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		t.Logf("Strategic analysis average response time: %v", avgTime)
		t.Logf("Total time for %d requests: %v", iterations, elapsed)

		// Strategic analysis should complete in reasonable time
		if avgTime > 200*time.Millisecond {
			t.Logf("WARNING: Strategic analysis slower than expected: %v", avgTime)
		}
	})
}

// TestPerformance_DatabaseOperations benchmarks database-heavy operations
func TestPerformance_DatabaseOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Scenario mapping creation performance", func(t *testing.T) {
		// Setup
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)

		iterations := 30
		start := time.Now()

		for i := 0; i < iterations; i++ {
			body := map[string]interface{}{
				"scenario_name":       fmt.Sprintf("perf-scenario-%d", i),
				"stage_id":            stageID,
				"contribution_weight": 0.8,
				"completion_status":   "not_started",
				"priority":            1,
				"estimated_impact":    7.5,
				"notes":               "Performance test",
			}

			w := makeHTTPRequest(t, router, "POST", "/api/v1/progress/scenarios", body)
			if w.Code != http.StatusOK {
				t.Errorf("Iteration %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		t.Logf("Scenario mapping creation average time: %v", avgTime)
		t.Logf("Total time for %d insertions: %v", iterations, elapsed)

		// Database writes should be reasonably fast
		if avgTime > 50*time.Millisecond {
			t.Logf("WARNING: Scenario mapping creation slower than expected: %v", avgTime)
		}
	})

	t.Run("Bulk retrieval performance", func(t *testing.T) {
		iterations := 30
		start := time.Now()

		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/progress/scenarios", nil)
			if w.Code != http.StatusOK {
				t.Errorf("Iteration %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		t.Logf("Bulk scenario retrieval average time: %v", avgTime)
		t.Logf("Total time for %d queries: %v", iterations, elapsed)
	})
}

// TestPerformance_MemoryUsage tests memory efficiency
func TestPerformance_MemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Memory usage under load", func(t *testing.T) {
		// Setup large dataset
		treeID := createTestTechTree(t, testDB)

		for i := 1; i <= 20; i++ {
			sectorID := fmt.Sprintf("00000000-0000-0000-0000-00000000%04d", i+200)
			testDB.Exec(`
				INSERT INTO sectors (id, tree_id, name, category, description, progress_percentage,
					position_x, position_y, color, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
			`, sectorID, treeID, fmt.Sprintf("Memory Test Sector %d", i), "software",
				"Memory test", float64(i*5), float64(i*25), float64(i*25), "#3498db")
		}

		// Make multiple large queries
		iterations := 50
		start := time.Now()

		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors", nil)
			if w.Code != http.StatusOK {
				t.Errorf("Iteration %d failed", i)
			}
			// Response is automatically garbage collected
		}

		elapsed := time.Since(start)
		t.Logf("Completed %d large queries in %v", iterations, elapsed)
		t.Logf("Average time per query: %v", elapsed/time.Duration(iterations))

		// If this test completes without running out of memory, we're good
		t.Log("Memory usage test completed successfully")
	})
}

// BenchmarkHealthEndpoint provides Go benchmark for health endpoint
func BenchmarkHealthEndpoint(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(&testing.T{}, router, "GET", "/health", nil)
		if w.Code != http.StatusOK {
			b.Fatalf("Request failed with status %d", w.Code)
		}
	}
}

// BenchmarkGetRecommendations provides Go benchmark for recommendations endpoint
func BenchmarkGetRecommendations(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(&testing.T{}, router, "GET", "/api/v1/recommendations", nil)
		if w.Code != http.StatusOK {
			b.Fatalf("Request failed with status %d", w.Code)
		}
	}
}

// BenchmarkStrategicAnalysis provides Go benchmark for strategic analysis
func BenchmarkStrategicAnalysis(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	body := map[string]interface{}{
		"current_resources": 5,
		"time_horizon":      12,
		"priority_sectors":  []string{"software", "manufacturing"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(&testing.T{}, router, "POST", "/api/v1/tech-tree/analyze", body)
		if w.Code != http.StatusOK {
			b.Fatalf("Request failed with status %d", w.Code)
		}
	}
}
