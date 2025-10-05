package main

import (
	"sync"
	"testing"
	"time"
)

// TestHandlerPerformance tests the performance of various handlers
func TestHandlerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("HealthEndpoint_Latency", func(t *testing.T) {
		maxDuration := 500 * time.Millisecond // Increased timeout for CI environments

		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(t, testApp.App, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Health check took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("FeaturesList_Latency", func(t *testing.T) {
		maxDuration := 500 * time.Millisecond

		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/features",
		}

		w := makeHTTPRequest(t, testApp.App, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Features list took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("RICECalculation_Performance", func(t *testing.T) {
		features := createTestFeatures(50)
		maxDuration := 1 * time.Second

		start := time.Now()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/rice",
			Body:   features,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("RICE calculation for 50 features took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("Prioritization_Performance", func(t *testing.T) {
		features := createTestFeatures(100)
		maxDuration := 2 * time.Second

		start := time.Now()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/prioritize",
			Body: map[string]interface{}{
				"features": features,
				"strategy": "rice",
			},
		}

		w := makeHTTPRequest(t, testApp.App, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Prioritization of 100 features took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("RoadmapGeneration_Performance", func(t *testing.T) {
		maxDuration := 1 * time.Second

		start := time.Now()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roadmap/generate",
			Body: map[string]interface{}{
				"feature_ids":     []string{"f1", "f2", "f3", "f4", "f5"},
				"start_date":      time.Now(),
				"duration_months": 12,
				"team_capacity":   200,
			},
		}

		w := makeHTTPRequest(t, testApp.App, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Roadmap generation took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("SprintPlanning_Performance", func(t *testing.T) {
		maxDuration := 2 * time.Second

		start := time.Now()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/sprint/plan",
			Body: map[string]interface{}{
				"team_size":      10,
				"velocity":       8,
				"duration_weeks": 2,
			},
		}

		w := makeHTTPRequest(t, testApp.App, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Sprint planning took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("ROICalculation_Performance", func(t *testing.T) {
		feature := createTestFeature("Performance Test Feature")
		maxDuration := 500 * time.Millisecond

		start := time.Now()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roi/calculate",
			Body:   feature,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("ROI calculation took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("Dashboard_Performance", func(t *testing.T) {
		maxDuration := 1 * time.Second

		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/dashboard",
		}

		w := makeHTTPRequest(t, testApp.App, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Dashboard load took %v, expected less than %v", duration, maxDuration)
		}
	})
}

// TestConcurrentRequests tests the system's ability to handle concurrent requests
func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("ConcurrentHealth_10", func(t *testing.T) {
		concurrency := 10
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}

				w := makeHTTPRequest(t, testApp.App, req)

				if w.Code != 200 {
					errors <- nil
				}
			}()
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)

		if len(errors) > 0 {
			t.Errorf("Some concurrent requests failed")
		}

		maxDuration := 2 * time.Second
		if duration > maxDuration {
			t.Errorf("Concurrent requests took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("ConcurrentFeatureCreation_20", func(t *testing.T) {
		concurrency := 20
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				feature := createTestFeature("Concurrent Feature")
				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/features",
					Body:   feature,
				}

				w := makeHTTPRequest(t, testApp.App, req)

				if w.Code == 200 {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		duration := time.Since(start)

		if successCount != concurrency {
			t.Errorf("Expected %d successful creations, got %d", concurrency, successCount)
		}

		maxDuration := 5 * time.Second
		if duration > maxDuration {
			t.Errorf("Concurrent feature creation took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("ConcurrentPrioritization_15", func(t *testing.T) {
		concurrency := 15
		var wg sync.WaitGroup

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				features := createTestFeatures(10)
				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/features/prioritize",
					Body: map[string]interface{}{
						"features": features,
						"strategy": "rice",
					},
				}

				makeHTTPRequest(t, testApp.App, req)
			}()
		}

		wg.Wait()

		duration := time.Since(start)

		maxDuration := 5 * time.Second
		if duration > maxDuration {
			t.Errorf("Concurrent prioritization took %v, expected less than %v", duration, maxDuration)
		}
	})

	t.Run("ConcurrentROICalculation_25", func(t *testing.T) {
		concurrency := 25
		var wg sync.WaitGroup
		results := make(chan int, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				feature := createTestFeature("ROI Feature")
				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/roi/calculate",
					Body:   feature,
				}

				w := makeHTTPRequest(t, testApp.App, req)
				results <- w.Code
			}()
		}

		wg.Wait()
		close(results)

		duration := time.Since(start)

		successCount := 0
		for code := range results {
			if code == 200 {
				successCount++
			}
		}

		if successCount != concurrency {
			t.Errorf("Expected %d successful calculations, got %d", concurrency, successCount)
		}

		maxDuration := 8 * time.Second
		if duration > maxDuration {
			t.Errorf("Concurrent ROI calculations took %v, expected less than %v", duration, maxDuration)
		}
	})
}

// TestMemoryUsage tests memory efficiency of handlers
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("LargeFeatureList_Memory", func(t *testing.T) {
		// Create a large feature list
		features := createTestFeatures(500)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/rice",
			Body:   features,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// If we get here without OOM, the test passes
	})

	t.Run("MultipleRoadmaps_Memory", func(t *testing.T) {
		// Generate multiple roadmaps in sequence
		for i := 0; i < 10; i++ {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/roadmap/generate",
				Body: map[string]interface{}{
					"feature_ids":     []string{"f1", "f2", "f3", "f4", "f5"},
					"start_date":      time.Now(),
					"duration_months": 6,
					"team_capacity":   100,
				},
			}

			w := makeHTTPRequest(t, testApp.App, req)

			if w.Code != 200 {
				t.Errorf("Roadmap generation %d failed with status %d", i+1, w.Code)
			}
		}
	})
}

// BenchmarkRICECalculation benchmarks RICE score calculation
func BenchmarkRICECalculation(b *testing.B) {
	testApp := setupTestApp(&testing.T{})
	defer testApp.Cleanup()

	feature := &Feature{
		Reach:      5000,
		Impact:     4,
		Confidence: 0.85,
		Effort:     8,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testApp.App.calculateRICE(feature)
	}
}

// BenchmarkFeatureSorting benchmarks feature sorting
func BenchmarkFeatureSorting(b *testing.B) {
	testApp := setupTestApp(&testing.T{})
	defer testApp.Cleanup()

	features := createTestFeatures(100)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testApp.App.sortFeaturesByRICE(features)
	}
}

// BenchmarkPrioritization benchmarks the full prioritization process
func BenchmarkPrioritization(b *testing.B) {
	testApp := setupTestApp(&testing.T{})
	defer testApp.Cleanup()

	features := createTestFeatures(50)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Calculate RICE scores
		for j := range features {
			features[j].Score = testApp.App.calculateRICE(&features[j])
		}

		// Sort features
		testApp.App.sortFeaturesByRICE(features)
	}
}

// BenchmarkROICalculation benchmarks ROI calculation
func BenchmarkROICalculation(b *testing.B) {
	testApp := setupTestApp(&testing.T{})
	defer testApp.Cleanup()

	feature := &Feature{
		ID:         "bench-feature",
		Reach:      10000,
		Impact:     5,
		Confidence: 0.9,
		Effort:     10,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testApp.App.calculateROI(feature)
	}
}

// BenchmarkSprintOptimization benchmarks sprint optimization
func BenchmarkSprintOptimization(b *testing.B) {
	testApp := setupTestApp(&testing.T{})
	defer testApp.Cleanup()

	features := createTestFeatures(30)
	capacity := 100

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testApp.App.optimizeSprint(capacity, features)
	}
}

// BenchmarkDashboardMetrics benchmarks dashboard metrics fetching
func BenchmarkDashboardMetrics(b *testing.B) {
	testApp := setupTestApp(&testing.T{})
	defer testApp.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testApp.App.fetchDashboardMetrics()
	}
}
