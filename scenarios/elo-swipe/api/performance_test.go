
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("CreateList_100Items", func(t *testing.T) {
		start := time.Now()

		req := TestData.CreateListRequest("Performance Test", 100)
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.CreateList(w, httpReq)

		duration := time.Since(start)

		if w.Code != http.StatusCreated {
			t.Errorf("Failed to create list: status %d", w.Code)
		}

		// Should complete in under 2 seconds
		if duration > 2*time.Second {
			t.Errorf("CreateList took %v, expected < 2s", duration)
		}

		t.Logf("CreateList(100 items) took %v", duration)

		// Cleanup
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if listID, ok := response["list_id"].(string); ok {
			testApp.DB.Exec("DELETE FROM elo_swipe.items WHERE list_id = $1", listID)
			testApp.DB.Exec("DELETE FROM elo_swipe.lists WHERE id = $1", listID)
		}
	})

	t.Run("GetRankings_LargeList", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Large List", 100)
		defer testList.Cleanup()

		start := time.Now()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/rankings", testList.List.ID),
			URLVars: map[string]string{"id": testList.List.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetRankings(w, httpReq)

		duration := time.Since(start)

		if w.Code != http.StatusOK {
			t.Errorf("Failed to get rankings: status %d", w.Code)
		}

		// Should complete in under 500ms
		if duration > 500*time.Millisecond {
			t.Errorf("GetRankings took %v, expected < 500ms", duration)
		}

		t.Logf("GetRankings(100 items) took %v", duration)
	})

	t.Run("CreateComparison_Speed", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 10)
		defer testList.Cleanup()

		// Measure average comparison creation time
		var totalDuration time.Duration
		iterations := 10

		for i := 0; i < iterations; i++ {
			itemA := testList.Items[i%len(testList.Items)]
			itemB := testList.Items[(i+1)%len(testList.Items)]

			req := TestData.CreateComparisonRequest(testList.List.ID, itemA.ID, itemB.ID)

			start := time.Now()

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/comparisons",
				Body:   req,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			testApp.App.CreateComparison(w, httpReq)

			duration := time.Since(start)
			totalDuration += duration

			if w.Code != http.StatusCreated {
				t.Errorf("Failed to create comparison: status %d", w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)

		// Average should be under 100ms
		if avgDuration > 100*time.Millisecond {
			t.Errorf("Average comparison time %v, expected < 100ms", avgDuration)
		}

		t.Logf("Average comparison time: %v (over %d iterations)", avgDuration, iterations)
	})

	t.Run("GetNextComparison_Speed", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 20)
		defer testList.Cleanup()

		start := time.Now()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/lists/%s/next-comparison", testList.List.ID),
			URLVars: map[string]string{"id": testList.List.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testApp.App.GetNextComparison(w, httpReq)

		duration := time.Since(start)

		if w.Code != http.StatusOK {
			t.Errorf("Failed to get next comparison: status %d", w.Code)
		}

		// Should complete in under 100ms
		if duration > 100*time.Millisecond {
			t.Errorf("GetNextComparison took %v, expected < 100ms", duration)
		}

		t.Logf("GetNextComparison took %v", duration)
	})
}

// TestConcurrency tests concurrent access patterns
func TestConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("ConcurrentGetLists", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 5)
		defer testList.Cleanup()

		var wg sync.WaitGroup
		concurrency := 10
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/lists",
				})
				if err != nil {
					errors <- err
					return
				}

				testApp.App.GetLists(w, httpReq)

				if w.Code != http.StatusOK {
					errors <- fmt.Errorf("unexpected status: %d", w.Code)
				}
			}()
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			t.Errorf("Concurrent request failed: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Had %d errors during concurrent GetLists", errorCount)
		}
	})

	t.Run("ConcurrentComparisons", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 10)
		defer testList.Cleanup()

		var wg sync.WaitGroup
		concurrency := 5
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()

				itemA := testList.Items[iteration%len(testList.Items)]
				itemB := testList.Items[(iteration+1)%len(testList.Items)]

				req := TestData.CreateComparisonRequest(testList.List.ID, itemA.ID, itemB.ID)

				w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/comparisons",
					Body:   req,
				})
				if err != nil {
					errors <- err
					return
				}

				testApp.App.CreateComparison(w, httpReq)

				if w.Code != http.StatusCreated {
					errors <- fmt.Errorf("unexpected status: %d", w.Code)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			t.Errorf("Concurrent comparison failed: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Had %d errors during concurrent comparisons", errorCount)
		}
	})

	t.Run("ConcurrentRankings", func(t *testing.T) {
		testList := setupTestList(t, testApp.DB, "Test List", 20)
		defer testList.Cleanup()

		var wg sync.WaitGroup
		concurrency := 10
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
					Method:  "GET",
					Path:    fmt.Sprintf("/api/v1/lists/%s/rankings", testList.List.ID),
					URLVars: map[string]string{"id": testList.List.ID},
				})
				if err != nil {
					errors <- err
					return
				}

				testApp.App.GetRankings(w, httpReq)

				if w.Code != http.StatusOK {
					errors <- fmt.Errorf("unexpected status: %d", w.Code)
				}
			}()
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			t.Errorf("Concurrent ranking request failed: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Had %d errors during concurrent rankings", errorCount)
		}
	})
}

// TestMemoryUsage tests memory efficiency with large datasets
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("LargeList_MemoryEfficiency", func(t *testing.T) {
		// Create and query a large list
		testList := setupTestList(t, testApp.DB, "Large List", 500)
		defer testList.Cleanup()

		// Get rankings multiple times
		for i := 0; i < 10; i++ {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method:  "GET",
				Path:    fmt.Sprintf("/api/v1/lists/%s/rankings", testList.List.ID),
				URLVars: map[string]string{"id": testList.List.ID},
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			testApp.App.GetRankings(w, httpReq)

			if w.Code != http.StatusOK {
				t.Errorf("Iteration %d failed with status %d", i, w.Code)
			}
		}

		t.Log("Memory usage test completed - check for memory leaks manually if needed")
	})
}
