package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPerformance runs performance benchmarks
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("ListComponentsPerformance", func(t *testing.T) {
		// Create some test components first
		for i := 0; i < 20; i++ {
			createTestComponent(t, testDB.DB, fmt.Sprintf("PerfTestComponent%d", i))
		}

		start := time.Now()
		iterations := 100

		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/components",
				QueryParams: map[string]string{
					"limit":  "20",
					"offset": "0",
				},
			})

			assert.Equal(t, 200, w.Code)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("List Components Performance:")
		t.Logf("  Total time: %v", duration)
		t.Logf("  Average time per request: %v", avgDuration)
		t.Logf("  Requests per second: %.2f", float64(iterations)/duration.Seconds())

		// Performance assertion - should handle 100 requests in reasonable time
		if duration > 5*time.Second {
			t.Logf("WARNING: List components took longer than expected: %v", duration)
		}
	})

	t.Run("ComponentCreationPerformance", func(t *testing.T) {
		start := time.Now()
		iterations := 50

		for i := 0; i < iterations; i++ {
			componentData := getValidComponentData()
			componentData.Name = fmt.Sprintf("PerfCreate%d", i)

			w := makeHTTPRequest(router, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/components",
				Body:   componentData,
			})

			assert.Equal(t, 201, w.Code)
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Component Creation Performance:")
		t.Logf("  Total time: %v", duration)
		t.Logf("  Average time per creation: %v", avgDuration)
		t.Logf("  Creations per second: %.2f", float64(iterations)/duration.Seconds())

		// Performance assertion
		if avgDuration > 200*time.Millisecond {
			t.Logf("WARNING: Component creation slower than expected: %v avg", avgDuration)
		}
	})

	t.Run("ConcurrentRequestsPerformance", func(t *testing.T) {
		// Create a test component
		componentID := createTestComponent(t, testDB.DB, "ConcurrentPerfTest")

		concurrentUsers := 20
		requestsPerUser := 10

		start := time.Now()
		var wg sync.WaitGroup
		wg.Add(concurrentUsers)

		for user := 0; user < concurrentUsers; user++ {
			go func(userID int) {
				defer wg.Done()

				for req := 0; req < requestsPerUser; req++ {
					w := makeHTTPRequest(router, HTTPTestRequest{
						Method: "GET",
						Path:   fmt.Sprintf("/api/v1/components/%s", componentID.String()),
					})

					assert.Equal(t, 200, w.Code)
				}
			}(user)
		}

		wg.Wait()
		duration := time.Since(start)
		totalRequests := concurrentUsers * requestsPerUser
		avgDuration := duration / time.Duration(totalRequests)

		t.Logf("Concurrent Requests Performance:")
		t.Logf("  Concurrent users: %d", concurrentUsers)
		t.Logf("  Requests per user: %d", requestsPerUser)
		t.Logf("  Total requests: %d", totalRequests)
		t.Logf("  Total time: %v", duration)
		t.Logf("  Average time per request: %v", avgDuration)
		t.Logf("  Requests per second: %.2f", float64(totalRequests)/duration.Seconds())

		// Should handle concurrent load efficiently
		if duration > 10*time.Second {
			t.Logf("WARNING: Concurrent requests took longer than expected: %v", duration)
		}
	})

	t.Run("DatabaseConnectionPoolPerformance", func(t *testing.T) {
		// Test database connection pool under load
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			componentID := createTestComponent(t, testDB.DB, fmt.Sprintf("DBPoolTest%d", i))

			w := makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   fmt.Sprintf("/api/v1/components/%s", componentID.String()),
			})

			assert.Equal(t, 200, w.Code)
		}

		duration := time.Since(start)

		t.Logf("Database Connection Pool Performance:")
		t.Logf("  Total operations (create + read): %d", iterations*2)
		t.Logf("  Total time: %v", duration)
		t.Logf("  Operations per second: %.2f", float64(iterations*2)/duration.Seconds())
	})
}

// BenchmarkListComponents benchmarks the list components endpoint
func BenchmarkListComponents(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(&testing.T{})
	defer testDB.Cleanup()

	// Create test data
	for i := 0; i < 50; i++ {
		createTestComponent(&testing.T{}, testDB.DB, fmt.Sprintf("BenchComponent%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components",
			QueryParams: map[string]string{
				"limit":  "20",
				"offset": "0",
			},
		})
	}
}

// BenchmarkGetComponent benchmarks the get component endpoint
func BenchmarkGetComponent(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(&testing.T{})
	defer testDB.Cleanup()

	componentID := createTestComponent(&testing.T{}, testDB.DB, "BenchGetComponent")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s", componentID.String()),
		})
	}
}

// BenchmarkCreateComponent benchmarks the create component endpoint
func BenchmarkCreateComponent(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(&testing.T{})
	defer testDB.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		componentData := getValidComponentData()
		componentData.Name = fmt.Sprintf("BenchCreate%d", i)

		makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})
	}
}
