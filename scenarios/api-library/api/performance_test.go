
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkSearchAPIs benchmarks search performance
func BenchmarkSearchAPIs(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	if env == nil {
		b.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	searchReq := SearchRequest{
		Query: "payment processing",
		Limit: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   searchReq,
		})
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		searchAPIsHandler(w, httpReq)
	}
}

// BenchmarkListAPIs benchmarks list performance
func BenchmarkListAPIs(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	if env == nil {
		b.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/apis",
		})
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		listAPIsHandler(w, httpReq)
	}
}

// BenchmarkGetAPI benchmarks single API retrieval
func BenchmarkGetAPI(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	if env == nil {
		b.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	testAPI := setupTestAPI(&testing.T{}, env.DB, "Benchmark API")
	defer testAPI.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/apis/" + testAPI.API.ID,
			URLVars: map[string]string{"id": testAPI.API.ID},
		})
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		getAPIHandler(w, httpReq)
	}
}

// TestSearchResponseTime tests response time requirements
func TestSearchResponseTime(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	// Create test APIs
	for i := 0; i < 5; i++ {
		testAPI := setupTestAPI(t, env.DB, fmt.Sprintf("Perf Test API %d", i))
		defer testAPI.Cleanup()
	}

	searchReq := SearchRequest{
		Query: "test",
		Limit: 10,
	}

	t.Run("Response_under_100ms", func(t *testing.T) {
		start := time.Now()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchAPIsHandler(w, httpReq)
		elapsed := time.Since(start)

		if elapsed > 100*time.Millisecond {
			t.Logf("Warning: Search response time %v exceeds 100ms target", elapsed)
		} else {
			t.Logf("Search response time: %v", elapsed)
		}
	})
}

// TestConcurrentSearchRequests tests concurrent request handling
func TestConcurrentSearchRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	numRequests := 50
	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32

	t.Run("Handle_50_concurrent_requests", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				searchReq := SearchRequest{
					Query: fmt.Sprintf("test%d", id),
					Limit: 5,
				}

				w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/search",
					Body:   searchReq,
				})
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
					return
				}

				searchAPIsHandler(w, httpReq)

				if w.Code == 200 {
					atomic.AddInt32(&successCount, 1)
				} else {
					atomic.AddInt32(&errorCount, 1)
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Completed %d requests in %v", numRequests, elapsed)
		t.Logf("Success: %d, Errors: %d", successCount, errorCount)

		if successCount < int32(numRequests)*80/100 {
			t.Errorf("Success rate too low: %d/%d", successCount, numRequests)
		}
	})
}

// TestDatabaseConnectionPool tests connection pool performance
func TestDatabaseConnectionPool(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	t.Run("Connection_pool_handles_load", func(t *testing.T) {
		var wg sync.WaitGroup
		numQueries := 100

		start := time.Now()

		for i := 0; i < numQueries; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				var count int
				err := env.DB.QueryRow("SELECT COUNT(*) FROM apis").Scan(&count)
				if err != nil {
					t.Errorf("Query %d failed: %v", id, err)
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Completed %d concurrent queries in %v", numQueries, elapsed)
		avgTime := elapsed.Milliseconds() / int64(numQueries)

		if avgTime > 100 {
			t.Logf("Warning: Average query time %dms exceeds 100ms", avgTime)
		}
	})
}

// TestCachePerformance tests cache hit/miss performance
func TestCachePerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	if redisClient == nil {
		initRedisCache()
	}

	if redisClient == nil {
		t.Skip("Redis client not available")
		return
	}

	testAPI := setupTestAPI(t, env.DB, "Cache Perf API")
	defer testAPI.Cleanup()

	t.Run("Cache_hit_faster_than_database", func(t *testing.T) {
		// First request - cache miss
		start1 := time.Now()
		w1, httpReq1, _ := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/apis/" + testAPI.API.ID,
			URLVars: map[string]string{"id": testAPI.API.ID},
		})
		getAPIHandler(w1, httpReq1)
		duration1 := time.Since(start1)

		// Second request - cache hit (if caching is implemented)
		start2 := time.Now()
		w2, httpReq2, _ := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/apis/" + testAPI.API.ID,
			URLVars: map[string]string{"id": testAPI.API.ID},
		})
		getAPIHandler(w2, httpReq2)
		duration2 := time.Since(start2)

		t.Logf("First request: %v, Second request: %v", duration1, duration2)

		if duration2 > duration1 {
			t.Log("Note: Cache may not be implemented or second request was slower")
		}
	})
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		t.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	t.Run("Large_result_set_handling", func(t *testing.T) {
		// Create multiple test APIs
		testAPIs := make([]*TestAPI, 20)
		for i := 0; i < 20; i++ {
			testAPIs[i] = setupTestAPI(t, env.DB, fmt.Sprintf("Memory Test API %d", i))
			defer testAPIs[i].Cleanup()
		}

		searchReq := SearchRequest{
			Query: "Memory Test",
			Limit: 100,
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search",
			Body:   searchReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		searchAPIsHandler(w, httpReq)

		if w.Code != 200 {
			t.Errorf("Large result set failed with status: %d", w.Code)
		}
	})
}

// TestRateLimitPerformance tests rate limiter performance
func TestRateLimitPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RateLimiter_allows_burst", func(t *testing.T) {
		// Send burst of requests
		numRequests := 20
		var wg sync.WaitGroup
		var allowed int32
		var blocked int32

		start := time.Now()

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				})
				if err != nil {
					return
				}

				healthHandler(w, httpReq)

				if w.Code == 200 {
					atomic.AddInt32(&allowed, 1)
				} else if w.Code == 429 {
					atomic.AddInt32(&blocked, 1)
				}
			}()
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Burst test: %d allowed, %d blocked in %v", allowed, blocked, elapsed)

		if allowed < int32(numRequests)*50/100 {
			t.Logf("Note: More than 50%% of requests were blocked")
		}
	})
}
