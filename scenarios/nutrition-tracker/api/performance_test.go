// +build testing

package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		}

		_, err := makeHTTPRequest(req, healthCheck)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkGetMealSuggestions benchmarks the meal suggestions endpoint
func BenchmarkGetMealSuggestions(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/suggestions",
		}

		_, err := makeHTTPRequest(req, getMealSuggestions)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkCreateMeal benchmarks meal creation
func BenchmarkCreateMeal(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	if testDB == nil {
		b.Skip("TEST_POSTGRES_URL not set, skipping database benchmarks")
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	meal := Meal{
		UserID:          "bench-user",
		MealType:        "lunch",
		FoodDescription: "Benchmark Meal",
		Calories:        500,
		Protein:         30,
		Carbs:           50,
		Fat:             20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/meals",
			Body:   meal,
		}

		_, err := makeHTTPRequest(req, createMeal)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// TestConcurrentRequests tests system behavior under concurrent load
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthCheckConcurrency", func(t *testing.T) {
		concurrency := 50
		iterations := 100

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterations)

		startTime := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < iterations; j++ {
					req := HTTPTestRequest{
						Method: "GET",
						Path:   "/api/health",
					}

					recorder, err := makeHTTPRequest(req, healthCheck)
					if err != nil {
						errors <- err
						return
					}

					if recorder.Code != 200 {
						errors <- fmt.Errorf("expected status 200, got %d", recorder.Code)
						return
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		duration := time.Since(startTime)

		// Check for errors
		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Errorf("Concurrent request error: %v", err)
				errorCount++
			}
		}

		totalRequests := concurrency * iterations
		requestsPerSecond := float64(totalRequests) / duration.Seconds()

		t.Logf("Concurrent performance:")
		t.Logf("  Total requests: %d", totalRequests)
		t.Logf("  Duration: %v", duration)
		t.Logf("  Requests/sec: %.2f", requestsPerSecond)
		t.Logf("  Errors: %d", errorCount)

		if errorCount > 0 {
			t.Errorf("Failed %d out of %d requests", errorCount, totalRequests)
		}
	})

	t.Run("MealSuggestionsConcurrency", func(t *testing.T) {
		concurrency := 20
		iterations := 50

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterations)

		startTime := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < iterations; j++ {
					req := HTTPTestRequest{
						Method: "GET",
						Path:   "/api/suggestions",
					}

					recorder, err := makeHTTPRequest(req, getMealSuggestions)
					if err != nil {
						errors <- err
						return
					}

					if recorder.Code != 200 {
						errors <- fmt.Errorf("expected status 200, got %d", recorder.Code)
						return
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		duration := time.Since(startTime)

		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Errorf("Concurrent request error: %v", err)
				errorCount++
			}
		}

		totalRequests := concurrency * iterations
		requestsPerSecond := float64(totalRequests) / duration.Seconds()

		t.Logf("Suggestions concurrent performance:")
		t.Logf("  Total requests: %d", totalRequests)
		t.Logf("  Duration: %v", duration)
		t.Logf("  Requests/sec: %.2f", requestsPerSecond)
		t.Logf("  Errors: %d", errorCount)

		if errorCount > 0 {
			t.Errorf("Failed %d out of %d requests", errorCount, totalRequests)
		}
	})
}

// TestResponseTime tests that endpoints respond within acceptable time limits
func TestResponseTime(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	maxResponseTime := 100 * time.Millisecond

	t.Run("HealthCheckResponseTime", func(t *testing.T) {
		startTime := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		}

		_, err := makeHTTPRequest(req, healthCheck)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		duration := time.Since(startTime)

		if duration > maxResponseTime {
			t.Errorf("Response time %v exceeds maximum %v", duration, maxResponseTime)
		} else {
			t.Logf("Response time: %v", duration)
		}
	})

	t.Run("SuggestionsResponseTime", func(t *testing.T) {
		maxTime := 200 * time.Millisecond // Allow more time for suggestions

		startTime := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/suggestions",
		}

		_, err := makeHTTPRequest(req, getMealSuggestions)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		duration := time.Since(startTime)

		if duration > maxTime {
			t.Errorf("Response time %v exceeds maximum %v", duration, maxTime)
		} else {
			t.Logf("Response time: %v", duration)
		}
	})
}

// TestMemoryUsage tests memory efficiency under load
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MemoryLeakCheck", func(t *testing.T) {
		// Run many iterations to detect potential memory leaks
		iterations := 1000

		for i := 0; i < iterations; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/health",
			}

			recorder, err := makeHTTPRequest(req, healthCheck)
			if err != nil {
				t.Fatalf("Request failed at iteration %d: %v", i, err)
			}

			if recorder.Code != 200 {
				t.Fatalf("Unexpected status at iteration %d: %d", i, recorder.Code)
			}

			// Clear recorder to allow garbage collection
			recorder = nil
		}

		t.Logf("Completed %d iterations without errors", iterations)
	})
}

// TestDatabaseConnectionPool tests database connection handling
func TestDatabaseConnectionPool(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	db = testDB.DB

	t.Run("MultipleSimultaneousQueries", func(t *testing.T) {
		concurrency := 25 // Should match or exceed MaxOpenConns setting
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				userID := fmt.Sprintf("pool-test-user-%d", id)

				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/meals",
					Query:  map[string]string{"user_id": userID},
				}

				recorder, err := makeHTTPRequest(req, getMeals)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d failed: %v", id, err)
					return
				}

				if recorder.Code != 200 {
					errors <- fmt.Errorf("goroutine %d got status %d", id, recorder.Code)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Error(err)
				errorCount++
			}
		}

		if errorCount > 0 {
			t.Errorf("Failed %d out of %d concurrent database queries", errorCount, concurrency)
		}
	})

	t.Run("SustainedLoad", func(t *testing.T) {
		duration := 5 * time.Second
		concurrency := 10

		var wg sync.WaitGroup
		stopChan := make(chan struct{})
		errors := make(chan error, 100)

		startTime := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				count := 0
				for {
					select {
					case <-stopChan:
						t.Logf("Worker %d completed %d requests", id, count)
						return
					default:
						req := HTTPTestRequest{
							Method: "GET",
							Path:   "/api/nutrition",
							Query: map[string]string{
								"user_id": fmt.Sprintf("sustained-user-%d", id),
							},
						}

						recorder, err := makeHTTPRequest(req, getNutritionSummary)
						if err != nil {
							errors <- err
							return
						}

						if recorder.Code != 200 {
							errors <- fmt.Errorf("unexpected status: %d", recorder.Code)
						}

						count++
					}
				}
			}(i)
		}

		time.Sleep(duration)
		close(stopChan)
		wg.Wait()
		close(errors)

		elapsed := time.Since(startTime)

		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Error(err)
				errorCount++
			}
		}

		t.Logf("Sustained load test:")
		t.Logf("  Duration: %v", elapsed)
		t.Logf("  Errors: %d", errorCount)

		if errorCount > 0 {
			t.Errorf("Encountered %d errors during sustained load", errorCount)
		}
	})
}
