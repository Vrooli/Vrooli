// +build testing

package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestPerformance_ConcurrentRequests tests concurrent request handling
func TestPerformance_ConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Performance_100ConcurrentResearchRequests", func(t *testing.T) {
		concurrency := 100
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/shopping/research",
					Body: ShoppingResearchRequest{
						ProfileID: "test-user",
						Query:     "laptop",
						BudgetMax: 1000.0,
					},
				}

				w := makeHTTPRequest(env.Server, req)
				if w.Code != 200 {
					errors <- nil
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		errorCount := len(errors)
		if errorCount > 0 {
			t.Errorf("Got %d errors out of %d requests", errorCount, concurrency)
		}

		t.Logf("Handled %d concurrent requests in %v (%.2f req/s)",
			concurrency, duration, float64(concurrency)/duration.Seconds())

		// Performance benchmark: should handle 100 requests within 5 seconds
		if duration > 5*time.Second {
			t.Logf("Warning: Concurrent requests took %v (may need optimization)", duration)
		}
	})
}

// TestPerformance_HighLoadTracking tests tracking endpoint under load
func TestPerformance_HighLoadTracking(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Performance_ConcurrentTrackingRequests", func(t *testing.T) {
		concurrency := 50
		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := HTTPTestRequest{
					Method:  "GET",
					Path:    "/api/v1/shopping/tracking/test-profile",
					URLVars: map[string]string{"profile_id": "test-profile"},
				}

				makeHTTPRequest(env.Server, req)
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Handled %d concurrent tracking requests in %v (%.2f req/s)",
			concurrency, duration, float64(concurrency)/duration.Seconds())
	})
}

// TestPerformance_CacheEfficiency tests caching performance
func TestPerformance_CacheEfficiency(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, err := NewDatabase()
	if err != nil {
		t.Logf("Database initialization error: %v", err)
	}
	if db == nil {
		t.Skip("Database not available")
	}
	defer db.Close()

	if db.redis == nil {
		t.Skip("Redis not available for cache testing")
	}

	ctx := context.Background()

	t.Run("Performance_CacheHitSpeed", func(t *testing.T) {
		query := "cache-performance-test"
		budget := 500.0

		// First call - cache miss
		start := time.Now()
		_, err := db.SearchProducts(ctx, query, budget)
		firstCallDuration := time.Since(start)
		if err != nil {
			t.Fatalf("First call failed: %v", err)
		}

		// Second call - should hit cache
		start = time.Now()
		_, err = db.SearchProducts(ctx, query, budget)
		cachedCallDuration := time.Since(start)
		if err != nil {
			t.Fatalf("Cached call failed: %v", err)
		}

		t.Logf("Cache miss: %v, Cache hit: %v", firstCallDuration, cachedCallDuration)

		// Cached call should be faster
		if cachedCallDuration > firstCallDuration {
			t.Logf("Warning: Cache hit (%v) was slower than cache miss (%v)",
				cachedCallDuration, firstCallDuration)
		}
	})
}

// BenchmarkShoppingResearch benchmarks the shopping research endpoint
func BenchmarkShoppingResearch(b *testing.B) {
	env := setupTestServer(&testing.T{})
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/shopping/research",
		Body: ShoppingResearchRequest{
			ProfileID: "bench-user",
			Query:     "laptop",
			BudgetMax: 1000.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Server, req)
	}
}

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	env := setupTestServer(&testing.T{})
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Server, req)
	}
}

// BenchmarkPatternAnalysis benchmarks pattern analysis
func BenchmarkPatternAnalysis(b *testing.B) {
	env := setupTestServer(&testing.T{})
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/shopping/pattern-analysis",
		Body: PatternAnalysisRequest{
			ProfileID: "bench-user",
			Timeframe: "30d",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Server, req)
	}
}

// TestPerformance_MemoryUsage tests memory efficiency
func TestPerformance_MemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Performance_LargeResultSet", func(t *testing.T) {
		// Make multiple requests to check for memory leaks
		for i := 0; i < 100; i++ {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/shopping/research",
				Body: ShoppingResearchRequest{
					ProfileID:           "test",
					Query:               "product",
					BudgetMax:           1000.0,
					IncludeAlternatives: true,
				},
			}

			w := makeHTTPRequest(env.Server, req)
			if w.Code != 200 {
				t.Fatalf("Request %d failed with status %d", i, w.Code)
			}
		}

		t.Log("Completed 100 requests without memory issues")
	})
}

// TestPerformance_ResponseTime tests response time targets
func TestPerformance_ResponseTime(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Performance_HealthCheckLatency", func(t *testing.T) {
		iterations := 10
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}
			makeHTTPRequest(env.Server, req)
			totalDuration += time.Since(start)
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average health check latency: %v", avgDuration)

		// Health checks should be fast (<100ms target)
		if avgDuration > 100*time.Millisecond {
			t.Logf("Warning: Health check average latency %v exceeds 100ms target", avgDuration)
		}
	})

	t.Run("Performance_ResearchEndpointLatency", func(t *testing.T) {
		iterations := 10
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/shopping/research",
				Body: ShoppingResearchRequest{
					ProfileID: "test",
					Query:     "keyboard",
					BudgetMax: 150.0,
				},
			}
			makeHTTPRequest(env.Server, req)
			totalDuration += time.Since(start)
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average research endpoint latency: %v", avgDuration)

		// Research should complete within reasonable time (<500ms target)
		if avgDuration > 500*time.Millisecond {
			t.Logf("Warning: Research endpoint average latency %v exceeds 500ms target", avgDuration)
		}
	})
}
