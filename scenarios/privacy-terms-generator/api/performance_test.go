package main

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestHealthHandlerPerformance tests health endpoint performance
func TestHealthHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		start := time.Now()

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		})

		healthHandler(w, req)

		duration := time.Since(start)

		// Health check should be fast (< 1 second)
		if duration > time.Second {
			t.Errorf("Health check took too long: %v", duration)
		}

		t.Logf("Health check completed in %v", duration)
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		concurrency := 10
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()

				w, req := makeHTTPRequest(HTTPTestRequest{
					Method: "GET",
					Path:   "/api/health",
				})

				healthHandler(w, req)

				if w.Code != 200 {
					errors <- nil // placeholder error
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)

		// Check for errors
		errorCount := 0
		for range errors {
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("%d/%d concurrent requests failed", errorCount, concurrency)
		}

		// All concurrent requests should complete within reasonable time
		maxExpected := time.Second * 5
		if duration > maxExpected {
			t.Errorf("Concurrent requests took too long: %v (expected < %v)", duration, maxExpected)
		}

		t.Logf("%d concurrent health checks completed in %v", concurrency, duration)
	})
}

// TestGenerateHandlerPerformance tests document generation performance
func TestGenerateHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SingleRequest", func(t *testing.T) {
		genReq := TestData.GenerateRequest("Test Business", "privacy-policy", []string{"US"})

		start := time.Now()

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/generate",
			Body:   genReq,
		})

		generateHandler(w, req)

		duration := time.Since(start)

		// Should respond within reasonable time (even if it fails)
		maxExpected := time.Second * 30
		if duration > maxExpected {
			t.Errorf("Generate request took too long: %v (expected < %v)", duration, maxExpected)
		}

		t.Logf("Generate request completed in %v", duration)
	})

	t.Run("SequentialRequests", func(t *testing.T) {
		iterations := 5
		totalDuration := time.Duration(0)

		for i := 0; i < iterations; i++ {
			genReq := TestData.GenerateRequest("Test Business", "privacy-policy", []string{"US"})

			start := time.Now()

			w, req := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/legal/generate",
				Body:   genReq,
			})

			generateHandler(w, req)

			duration := time.Since(start)
			totalDuration += duration
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("%d sequential requests completed with average time: %v", iterations, avgDuration)

		// Average should be reasonable
		if avgDuration > time.Second*30 {
			t.Errorf("Average request time too high: %v", avgDuration)
		}
	})
}

// TestSearchClausesHandlerPerformance tests search performance
func TestSearchClausesHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		searchReq := TestData.SearchClauseRequest("data retention", 10)

		start := time.Now()

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body:   searchReq,
		})

		searchClausesHandler(w, req)

		duration := time.Since(start)

		// Search should complete within reasonable time
		maxExpected := time.Second * 10
		if duration > maxExpected {
			t.Errorf("Search took too long: %v (expected < %v)", duration, maxExpected)
		}

		t.Logf("Search completed in %v", duration)
	})

	t.Run("LargeResultSet", func(t *testing.T) {
		searchReq := TestData.SearchClauseRequest("privacy", 1000) // Large limit

		start := time.Now()

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/legal/clauses/search",
			Body:   searchReq,
		})

		searchClausesHandler(w, req)

		duration := time.Since(start)

		// Should handle large result sets
		maxExpected := time.Second * 15
		if duration > maxExpected {
			t.Errorf("Large result search took too long: %v (expected < %v)", duration, maxExpected)
		}

		t.Logf("Large result search completed in %v", duration)
	})
}

// TestTemplateFreshnessHandlerPerformance tests template freshness performance
func TestTemplateFreshnessHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		start := time.Now()

		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/legal/templates/freshness",
		})

		templateFreshnessHandler(w, req)

		duration := time.Since(start)

		// Should be very fast since it's a simple response
		if duration > time.Millisecond*100 {
			t.Errorf("Template freshness check took too long: %v", duration)
		}

		t.Logf("Template freshness check completed in %v", duration)
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		concurrency := 20
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				w, req := makeHTTPRequest(HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/legal/templates/freshness",
				})

				templateFreshnessHandler(w, req)

				if w.Code != 200 {
					errors <- nil // placeholder error
				}
			}()
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)

		// Check for errors
		errorCount := 0
		for range errors {
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("%d/%d concurrent requests failed", errorCount, concurrency)
		}

		// Should handle concurrent requests efficiently
		maxExpected := time.Second * 2
		if duration > maxExpected {
			t.Errorf("Concurrent requests took too long: %v (expected < %v)", duration, maxExpected)
		}

		t.Logf("%d concurrent template freshness checks completed in %v", concurrency, duration)
	})
}

// TestCORSMiddlewarePerformance tests CORS overhead
func TestCORSMiddlewarePerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := corsMiddleware(testHandler)

	t.Run("Overhead", func(t *testing.T) {
		// Measure without CORS
		start := time.Now()
		w1, req1 := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		})
		testHandler(w1, req1)
		withoutCORS := time.Since(start)

		// Measure with CORS
		start = time.Now()
		w2, req2 := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		})
		wrappedHandler(w2, req2)
		withCORS := time.Since(start)

		overhead := withCORS - withoutCORS

		// CORS middleware should add minimal overhead
		if overhead > time.Millisecond*10 {
			t.Errorf("CORS middleware overhead too high: %v", overhead)
		}

		t.Logf("CORS middleware overhead: %v", overhead)
	})
}

// BenchmarkHealthHandler benchmarks the health endpoint
func BenchmarkHealthHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		})

		healthHandler(w, req)
	}
}

// BenchmarkTemplateFreshnessHandler benchmarks template freshness
func BenchmarkTemplateFreshnessHandler(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/legal/templates/freshness",
		})

		templateFreshnessHandler(w, req)
	}
}

// BenchmarkCORSMiddleware benchmarks CORS middleware
func BenchmarkCORSMiddleware(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := corsMiddleware(testHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, req := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		})

		wrappedHandler(w, req)
	}
}
