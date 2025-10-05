// +build testing

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPerformanceGetCompetitors tests the performance of getting competitors
func TestPerformanceGetCompetitors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	// Create test data
	for i := 0; i < 100; i++ {
		testComp := setupTestCompetitor(t, env, fmt.Sprintf("Test Competitor %d", i))
		defer testComp.Cleanup()
	}

	t.Run("Sequential_100Requests", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 100; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/competitors",
			}

			w := testHandlerWithRequest(t, getCompetitorsHandler, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / 100

		t.Logf("Total duration: %v", duration)
		t.Logf("Average request duration: %v", avgDuration)

		// Should complete in reasonable time (< 5 seconds total)
		if duration > 5*time.Second {
			t.Errorf("Performance degraded: 100 requests took %v (expected < 5s)", duration)
		}
	})

	t.Run("Concurrent_100Requests", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount int32
		var failureCount int32

		start := time.Now()

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/competitors",
				}

				w := testHandlerWithRequest(t, getCompetitorsHandler, req)

				if w.Code == 200 {
					atomic.AddInt32(&successCount, 1)
				} else {
					atomic.AddInt32(&failureCount, 1)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Total duration: %v", duration)
		t.Logf("Successful requests: %d", successCount)
		t.Logf("Failed requests: %d", failureCount)

		// Should complete in reasonable time (< 3 seconds)
		if duration > 3*time.Second {
			t.Errorf("Performance degraded: 100 concurrent requests took %v (expected < 3s)", duration)
		}

		// Should have high success rate
		if successCount < 95 {
			t.Errorf("Success rate too low: %d/100 (expected >= 95)", successCount)
		}
	})
}

// TestPerformanceAddCompetitor tests the performance of adding competitors
func TestPerformanceAddCompetitor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	t.Run("Sequential_50Inserts", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 50; i++ {
			body := TestData.CreateCompetitorRequest(fmt.Sprintf("Perf Test Competitor %d", i))

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/competitors",
				Body:   body,
			}

			w := testHandlerWithRequest(t, addCompetitorHandler, req)

			if w.Code != 201 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / 50

		t.Logf("Total duration: %v", duration)
		t.Logf("Average insert duration: %v", avgDuration)

		// Should complete in reasonable time (< 3 seconds total)
		if duration > 3*time.Second {
			t.Errorf("Performance degraded: 50 inserts took %v (expected < 3s)", duration)
		}
	})

	t.Run("Concurrent_50Inserts", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount int32

		start := time.Now()

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()

				body := TestData.CreateCompetitorRequest(fmt.Sprintf("Concurrent Perf Competitor %d", iteration))

				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/competitors",
					Body:   body,
				}

				w := testHandlerWithRequest(t, addCompetitorHandler, req)

				if w.Code == 201 {
					atomic.AddInt32(&successCount, 1)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Total duration: %v", duration)
		t.Logf("Successful inserts: %d", successCount)

		// Should complete in reasonable time (< 2 seconds)
		if duration > 2*time.Second {
			t.Errorf("Performance degraded: 50 concurrent inserts took %v (expected < 2s)", duration)
		}

		// Should have high success rate
		if successCount < 48 {
			t.Errorf("Success rate too low: %d/50 (expected >= 48)", successCount)
		}
	})
}

// TestPerformanceGetTargets tests the performance of getting monitoring targets
func TestPerformanceGetTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	// Create test competitor
	testComp := setupTestCompetitor(t, env, "Performance Test Competitor")
	defer testComp.Cleanup()

	// Create multiple targets
	for i := 0; i < 50; i++ {
		testTarget := setupTestTarget(t, env, testComp.Competitor.ID)
		defer testTarget.Cleanup()
	}

	t.Run("Sequential_50Requests", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 50; i++ {
			req := HTTPTestRequest{
				Method:  "GET",
				Path:    fmt.Sprintf("/api/competitors/%s/targets", testComp.Competitor.ID),
				URLVars: map[string]string{"id": testComp.Competitor.ID},
			}

			w := testHandlerWithRequest(t, getTargetsHandler, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / 50

		t.Logf("Total duration: %v", duration)
		t.Logf("Average request duration: %v", avgDuration)

		// Should complete in reasonable time (< 2 seconds total)
		if duration > 2*time.Second {
			t.Errorf("Performance degraded: 50 requests took %v (expected < 2s)", duration)
		}
	})
}

// TestPerformanceGetAlerts tests the performance of getting alerts
func TestPerformanceGetAlerts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	// Create test alerts
	for i := 0; i < 100; i++ {
		_, err := env.DB.Exec(`
			INSERT INTO alerts (id, competitor_id, title, priority, url, category, summary, relevance_score, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, fmt.Sprintf("perf-alert-%d", i), "test-comp", fmt.Sprintf("Alert %d", i),
			"high", "http://example.com", "test", "Summary", 80, "unread")

		if err != nil {
			t.Skipf("Failed to create test alert: %v", err)
		}
	}

	t.Run("Sequential_100Requests", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 100; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/alerts",
			}

			w := testHandlerWithRequest(t, getAlertsHandler, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / 100

		t.Logf("Total duration: %v", duration)
		t.Logf("Average request duration: %v", avgDuration)

		// Should complete in reasonable time (< 5 seconds total)
		if duration > 5*time.Second {
			t.Errorf("Performance degraded: 100 requests took %v (expected < 5s)", duration)
		}
	})

	t.Run("WithFilters", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 50; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/alerts",
				QueryParams: map[string]string{
					"status":   "unread",
					"priority": "high",
				},
			}

			w := testHandlerWithRequest(t, getAlertsHandler, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)

		t.Logf("Total duration: %v", duration)

		// Should complete in reasonable time (< 3 seconds total)
		if duration > 3*time.Second {
			t.Errorf("Performance degraded: 50 filtered requests took %v (expected < 3s)", duration)
		}
	})
}

// TestPerformanceUpdateAlert tests the performance of updating alerts
func TestPerformanceUpdateAlert(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	// Create test alerts
	alertIDs := make([]string, 50)
	for i := 0; i < 50; i++ {
		alertID := fmt.Sprintf("update-perf-alert-%d", i)
		alertIDs[i] = alertID

		_, err := env.DB.Exec(`
			INSERT INTO alerts (id, competitor_id, title, priority, url, category, summary, relevance_score, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, alertID, "test-comp", fmt.Sprintf("Alert %d", i),
			"high", "http://example.com", "test", "Summary", 80, "unread")

		if err != nil {
			t.Skipf("Failed to create test alert: %v", err)
		}
	}

	t.Run("Sequential_50Updates", func(t *testing.T) {
		start := time.Now()

		for i, alertID := range alertIDs {
			body := map[string]interface{}{
				"status": "read",
			}

			req := HTTPTestRequest{
				Method:  "PATCH",
				Path:    fmt.Sprintf("/api/alerts/%s", alertID),
				URLVars: map[string]string{"id": alertID},
				Body:    body,
			}

			w := testHandlerWithRequest(t, updateAlertHandler, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / 50

		t.Logf("Total duration: %v", duration)
		t.Logf("Average update duration: %v", avgDuration)

		// Should complete in reasonable time (< 2 seconds total)
		if duration > 2*time.Second {
			t.Errorf("Performance degraded: 50 updates took %v (expected < 2s)", duration)
		}
	})
}

// TestMemoryUsage tests memory usage during operations
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	t.Run("LargeDatasetRetrieval", func(t *testing.T) {
		// Create large dataset
		for i := 0; i < 1000; i++ {
			testComp := setupTestCompetitor(t, env, fmt.Sprintf("Memory Test Competitor %d", i))
			defer testComp.Cleanup()
		}

		// Retrieve all at once
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/competitors",
		}

		start := time.Now()
		w := testHandlerWithRequest(t, getCompetitorsHandler, req)
		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Request failed with status %d", w.Code)
		}

		t.Logf("Retrieved 1000 competitors in %v", duration)

		// Should complete in reasonable time even with large dataset
		if duration > 2*time.Second {
			t.Errorf("Performance degraded: retrieving 1000 competitors took %v (expected < 2s)", duration)
		}
	})
}

// BenchmarkGetCompetitors benchmarks the get competitors handler
func BenchmarkGetCompetitors(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(&testing.T{})
	defer env.Cleanup()

	db = env.DB

	// Create test data
	for i := 0; i < 10; i++ {
		testComp := setupTestCompetitor(&testing.T{}, env, fmt.Sprintf("Benchmark Competitor %d", i))
		defer testComp.Cleanup()
	}

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/competitors",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testHandlerWithRequest(&testing.T{}, getCompetitorsHandler, req)
	}
}

// BenchmarkAddCompetitor benchmarks the add competitor handler
func BenchmarkAddCompetitor(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(&testing.T{})
	defer env.Cleanup()

	db = env.DB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := TestData.CreateCompetitorRequest(fmt.Sprintf("Benchmark Competitor %d", i))

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitors",
			Body:   body,
		}

		testHandlerWithRequest(&testing.T{}, addCompetitorHandler, req)
	}
}

// BenchmarkGetAlerts benchmarks the get alerts handler
func BenchmarkGetAlerts(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(&testing.T{})
	defer env.Cleanup()

	db = env.DB

	// Create test alerts
	for i := 0; i < 50; i++ {
		env.DB.Exec(`
			INSERT INTO alerts (id, competitor_id, title, priority, url, category, summary, relevance_score, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, fmt.Sprintf("bench-alert-%d", i), "test-comp", fmt.Sprintf("Alert %d", i),
			"high", "http://example.com", "test", "Summary", 80, "unread")
	}

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/alerts",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testHandlerWithRequest(&testing.T{}, getAlertsHandler, req)
	}
}
