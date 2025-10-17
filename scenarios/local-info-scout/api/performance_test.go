package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkHealthHandler benchmarks the health check endpoint
func BenchmarkHealthHandler(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(healthHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if w.Code != 200 {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkSearchHandler benchmarks the search endpoint
func BenchmarkSearchHandler(b *testing.B) {
	req := createTestSearchRequest("restaurant", 40.7128, -74.0060, 5.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})
		if w.Code != 200 {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkCategoriesHandler benchmarks the categories endpoint
func BenchmarkCategoriesHandler(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(categoriesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/categories",
		})
		if w.Code != 200 {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkDiscoverHandler benchmarks the discover endpoint
func BenchmarkDiscoverHandler(b *testing.B) {
	req := SearchRequest{
		Lat:    40.7128,
		Lon:    -74.0060,
		Radius: 5.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/discover",
			Body:   req,
		})
		if w.Code != 200 {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkPlaceDetailsHandler benchmarks the place details endpoint
func BenchmarkPlaceDetailsHandler(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := makeHTTPRequest(placeDetailsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/places/123",
		})
		if w.Code != 200 {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkParseNaturalLanguageQuery benchmarks natural language parsing
func BenchmarkParseNaturalLanguageQuery(b *testing.B) {
	queries := []string{
		"vegan restaurants within 2 miles",
		"coffee shops nearby",
		"pharmacies open now",
		"parks and recreation areas",
		"organic grocery stores",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		parseNaturalLanguageQuery(query)
	}
}

// BenchmarkApplySmartFilters benchmarks the filtering logic
func BenchmarkApplySmartFilters(b *testing.B) {
	places := getMockPlaces()
	req := SearchRequest{
		Query:     "restaurant",
		Lat:       40.7128,
		Lon:       -74.0060,
		Radius:    5.0,
		MinRating: 4.0,
		MaxPrice:  2,
		OpenNow:   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applySmartFilters(places, req)
	}
}

// BenchmarkGetCacheKey benchmarks cache key generation
func BenchmarkGetCacheKey(b *testing.B) {
	req := SearchRequest{
		Lat:      40.7128,
		Lon:      -74.0060,
		Radius:   5.0,
		Category: "restaurant",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getCacheKey(req)
	}
}

// BenchmarkIsChainStore benchmarks chain store detection
func BenchmarkIsChainStore(b *testing.B) {
	names := []string{
		"McDonald's",
		"Local Coffee Shop",
		"Starbucks",
		"Mom's Diner",
		"Subway",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := names[i%len(names)]
		isChainStore(name)
	}
}

// BenchmarkSortByRelevance benchmarks the sorting algorithm
func BenchmarkSortByRelevance(b *testing.B) {
	places := []Place{
		createTestPlace("1", "Place 1", "restaurant", 0.5, 4.8),
		createTestPlace("2", "Place 2", "restaurant", 0.8, 4.2),
		createTestPlace("3", "Place 3", "restaurant", 0.3, 4.5),
		createTestPlace("4", "Place 4", "restaurant", 1.2, 3.8),
		createTestPlace("5", "Place 5", "restaurant", 0.7, 4.6),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a copy to sort
		testPlaces := make([]Place, len(places))
		copy(testPlaces, places)
		sortByRelevance(testPlaces)
	}
}

// TestSearchHandlerPerformance tests search performance under load
func TestSearchHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := createTestSearchRequest("restaurant", 40.7128, -74.0060, 5.0)

	start := time.Now()
	iterations := 100

	for i := 0; i < iterations; i++ {
		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		if w.Code != 200 {
			t.Fatalf("Request %d failed with status %d", i, w.Code)
		}
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	t.Logf("Average request time: %v", avgDuration)

	// Performance threshold: average request should complete in < 100ms
	if avgDuration > 100*time.Millisecond {
		t.Logf("Warning: Average request time (%v) exceeds 100ms threshold", avgDuration)
	}
}

// TestConcurrentSearchRequests tests concurrent request handling
func TestConcurrentSearchRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	concurrency := 10
	requestsPerWorker := 10

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*requestsPerWorker)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			req := createTestSearchRequest("restaurant", 40.7128, -74.0060, 5.0)

			for j := 0; j < requestsPerWorker; j++ {
				w := makeHTTPRequest(searchHandler, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/search",
					Body:   req,
				})

				if w.Code != 200 {
					errors <- &testError{
						workerID: workerID,
						request:  j,
						status:   w.Code,
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	totalRequests := concurrency * requestsPerWorker

	t.Logf("Concurrent test: %d workers × %d requests = %d total requests in %v",
		concurrency, requestsPerWorker, totalRequests, duration)

	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent request error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Failed %d out of %d concurrent requests", errorCount, totalRequests)
	}
}

// testError represents a test error
type testError struct {
	workerID int
	request  int
	status   int
}

func (e *testError) Error() string {
	return fmt.Sprintf("Worker %d, Request %d failed with status %d",
		e.workerID, e.request, e.status)
}

// TestDiscoverHandlerPerformance tests discover endpoint performance
func TestDiscoverHandlerPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := SearchRequest{
		Lat:    40.7128,
		Lon:    -74.0060,
		Radius: 5.0,
	}

	start := time.Now()
	iterations := 50

	for i := 0; i < iterations; i++ {
		w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/discover",
			Body:   req,
		})

		if w.Code != 200 {
			t.Fatalf("Request %d failed with status %d", i, w.Code)
		}
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	t.Logf("Discover average request time: %v", avgDuration)

	// Discovery should be fast since it combines multiple operations
	if avgDuration > 150*time.Millisecond {
		t.Logf("Warning: Average discover time (%v) exceeds 150ms threshold", avgDuration)
	}
}

// TestMemoryUsage tests memory usage patterns
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a large number of places
	places := make([]Place, 1000)
	for i := 0; i < 1000; i++ {
		places[i] = createTestPlace(
			fmt.Sprintf("place-%d", i),
			fmt.Sprintf("Place %d", i),
			"restaurant",
			float64(i%10),
			4.5,
		)
	}

	req := SearchRequest{
		Lat:    40.7128,
		Lon:    -74.0060,
		Radius: 5.0,
	}

	start := time.Now()

	// Apply filters multiple times
	for i := 0; i < 100; i++ {
		_ = applySmartFilters(places, req)
	}

	duration := time.Since(start)
	t.Logf("Filtered 1000 places 100 times in %v", duration)

	// Should complete in reasonable time
	if duration > 1*time.Second {
		t.Errorf("Memory/performance issue: took %v to filter 1000 places 100 times", duration)
	}
}
