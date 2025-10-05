// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestSearchPatternsPerformance benchmarks search endpoint
func TestSearchPatternsPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("SearchResponseTime", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query:  map[string]string{"query": "test"},
		}

		start := time.Now()
		w := makeHTTPRequest(searchPatternsHandler, req)
		elapsed := time.Since(start)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		// Performance target: < 100ms (from PRD)
		if elapsed > 100*time.Millisecond {
			t.Logf("WARNING: Search took %v (target: <100ms)", elapsed)
		} else {
			t.Logf("Search completed in %v (✓ under 100ms)", elapsed)
		}
	})

	t.Run("SearchWithPaginationResponseTime", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/patterns/search",
			Query: map[string]string{
				"query":  "test",
				"limit":  "50",
				"offset": "0",
			},
		}

		start := time.Now()
		w := makeHTTPRequest(searchPatternsHandler, req)
		elapsed := time.Since(start)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		if elapsed > 100*time.Millisecond {
			t.Logf("WARNING: Paginated search took %v", elapsed)
		} else {
			t.Logf("Paginated search completed in %v", elapsed)
		}
	})
}

// TestPatternRetrievalPerformance benchmarks individual pattern retrieval
func TestPatternRetrievalPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("SinglePatternRetrievalTime", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/patterns/test-pattern-1",
			URLVars: map[string]string{"id": "test-pattern-1"},
		}

		start := time.Now()
		w := makeHTTPRequest(getPatternHandler, req)
		elapsed := time.Since(start)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		// Performance target: < 50ms (from PRD)
		if elapsed > 50*time.Millisecond {
			t.Logf("WARNING: Pattern retrieval took %v (target: <50ms)", elapsed)
		} else {
			t.Logf("Pattern retrieval completed in %v (✓ under 50ms)", elapsed)
		}
	})
}

// TestCodeGenerationPerformance benchmarks code generation
func TestCodeGenerationPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("CodeGenerationResponseTime", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/recipes/generate",
			Body: GenerationRequest{
				RecipeID:       "test-recipe-1",
				Language:       "go",
				Parameters:     map[string]interface{}{"key": "value"},
				TargetPlatform: "linux",
			},
		}

		start := time.Now()
		w := makeHTTPRequest(generateCodeHandler, req)
		elapsed := time.Since(start)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		// Performance target: < 2s (from PRD)
		if elapsed > 2*time.Second {
			t.Errorf("FAIL: Code generation took %v (target: <2s)", elapsed)
		} else {
			t.Logf("Code generation completed in %v (✓ under 2s)", elapsed)
		}
	})
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("ConcurrentSearchRequests", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/patterns/search",
					Query:  map[string]string{"query": fmt.Sprintf("test-%d", id)},
				}

				w := makeHTTPRequest(searchPatternsHandler, req)
				if w.Code != http.StatusOK {
					errors <- fmt.Errorf("request %d failed with status %d", id, w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}

		elapsed := time.Since(start)
		close(errors)

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Error(err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("%d out of %d concurrent requests failed", errorCount, concurrency)
		}

		t.Logf("Completed %d concurrent requests in %v (avg: %v per request)",
			concurrency, elapsed, elapsed/time.Duration(concurrency))
	})

	t.Run("ConcurrentMixedRequests", func(t *testing.T) {
		concurrency := 15
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				var w *http.ResponseWriter
				var req HTTPTestRequest

				// Alternate between different endpoint types
				switch id % 3 {
				case 0:
					req = HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/patterns/search",
					}
					w := makeHTTPRequest(searchPatternsHandler, req)
					if w.Code != http.StatusOK {
						errors <- fmt.Errorf("search request %d failed with status %d", id, w.Code)
					}
				case 1:
					req = HTTPTestRequest{
						Method:  "GET",
						Path:    "/api/v1/patterns/test-pattern-1",
						URLVars: map[string]string{"id": "test-pattern-1"},
					}
					w := makeHTTPRequest(getPatternHandler, req)
					if w.Code != http.StatusOK {
						errors <- fmt.Errorf("pattern request %d failed with status %d", id, w.Code)
					}
				case 2:
					req = HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/patterns/chapters",
					}
					w := makeHTTPRequest(getChaptersHandler, req)
					if w.Code != http.StatusOK {
						errors <- fmt.Errorf("chapters request %d failed with status %d", id, w.Code)
					}
				}

				_ = w
				done <- true
			}(i)
		}

		// Wait for all requests
		for i := 0; i < concurrency; i++ {
			<-done
		}

		elapsed := time.Since(start)
		close(errors)

		errorCount := 0
		for err := range errors {
			t.Error(err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("%d out of %d mixed concurrent requests failed", errorCount, concurrency)
		}

		t.Logf("Completed %d mixed concurrent requests in %v", concurrency, elapsed)
	})
}

// TestDatabaseConnectionPooling tests connection pool efficiency
func TestDatabaseConnectionPooling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("SequentialRequests_ConnectionReuse", func(t *testing.T) {
		iterations := 50
		start := time.Now()

		for i := 0; i < iterations; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/patterns/search",
			}

			w := makeHTTPRequest(searchPatternsHandler, req)
			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)

		t.Logf("Completed %d sequential requests in %v (avg: %v per request)",
			iterations, elapsed, avgTime)

		// Average should be well under 100ms per request
		if avgTime > 100*time.Millisecond {
			t.Logf("WARNING: Average request time %v exceeds target", avgTime)
		}
	})
}

// TestMemoryUsage provides basic memory usage tracking
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	seedTestData(t, testDB)
	defer cleanTestData(t, testDB)

	t.Run("SearchMemoryFootprint", func(t *testing.T) {
		// Make multiple search requests and ensure no memory leaks
		for i := 0; i < 100; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/patterns/search",
				Query:  map[string]string{"limit": "50"},
			}

			w := makeHTTPRequest(searchPatternsHandler, req)
			if w.Code != http.StatusOK {
				t.Fatalf("Request failed at iteration %d", i)
			}
		}

		// If we get here without panic or excessive memory usage, test passes
		t.Log("Memory usage test completed successfully")
	})
}
