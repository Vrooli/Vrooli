package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConcurrentFunnelCreation tests concurrent funnel creation
func TestConcurrentFunnelCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Concurrent_Creates", func(t *testing.T) {
		numConcurrent := 20
		var wg sync.WaitGroup
		var successCount int32
		var errorCount int32
		createdIDs := make([]string, 0, numConcurrent)
		var idsMutex sync.Mutex

		startTime := time.Now()

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				projectID := createTestProject(t, testServer.Server, fmt.Sprintf("Concurrent Project %d", index))

				funnelData := map[string]interface{}{
					"name":       fmt.Sprintf("Concurrent Test Funnel %d", index),
					"project_id": projectID,
					"steps": []map[string]interface{}{
						{
							"type":     "form",
							"position": 0,
							"title":    "Step 1",
							"content":  json.RawMessage(`{"fields":[{"name":"email","type":"email"}]}`),
						},
					},
				}

				req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
				recorder := httptest.NewRecorder()
				testServer.Server.router.ServeHTTP(recorder, req)

				if recorder.Code == http.StatusCreated {
					atomic.AddInt32(&successCount, 1)
					var response map[string]interface{}
					json.Unmarshal(recorder.Body.Bytes(), &response)

					idsMutex.Lock()
					createdIDs = append(createdIDs, response["id"].(string))
					idsMutex.Unlock()
				} else {
					atomic.AddInt32(&errorCount, 1)
					t.Logf("Concurrent create failed: %d - %s", recorder.Code, recorder.Body.String())
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(startTime)

		// Cleanup
		for _, id := range createdIDs {
			cleanupTestData(t, testServer.Server, id)
		}

		t.Logf("Created %d funnels concurrently in %v", successCount, duration)
		t.Logf("Success: %d, Errors: %d", successCount, errorCount)

		if successCount < int32(numConcurrent)*80/100 {
			t.Errorf("Expected at least 80%% success rate, got %d/%d", successCount, numConcurrent)
		}

		avgTime := duration / time.Duration(numConcurrent)
		if avgTime > 500*time.Millisecond {
			t.Logf("Warning: Average creation time %v exceeds 500ms target", avgTime)
		}
	})
}

// TestConcurrentFunnelExecution tests concurrent funnel sessions
func TestConcurrentFunnelExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Concurrent_Execution", func(t *testing.T) {
		// Create test funnel
		funnelID := createTestFunnel(t, testServer.Server, "Concurrent Execution Test")
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Get funnel and activate
		req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var funnel map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &funnel)
		slug := funnel["slug"].(string)

		updateData := map[string]interface{}{
			"name":   "Concurrent Execution Test",
			"status": "active",
		}
		req, _ = makeHTTPRequest("PUT", "/api/v1/funnels/"+funnelID, updateData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Run concurrent sessions
		numConcurrent := 50
		var wg sync.WaitGroup
		var successCount int32
		var errorCount int32

		startTime := time.Now()

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				// Execute funnel
				req, _ := makeHTTPRequest("GET", fmt.Sprintf("/api/v1/execute/%s", slug), nil)
				recorder := httptest.NewRecorder()
				testServer.Server.router.ServeHTTP(recorder, req)

				if recorder.Code == http.StatusOK {
					atomic.AddInt32(&successCount, 1)
				} else {
					atomic.AddInt32(&errorCount, 1)
					t.Logf("Concurrent execution failed: %d", recorder.Code)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(startTime)

		t.Logf("Executed %d concurrent sessions in %v", successCount, duration)
		t.Logf("Success: %d, Errors: %d", successCount, errorCount)

		if successCount < int32(numConcurrent)*90/100 {
			t.Errorf("Expected at least 90%% success rate, got %d/%d", successCount, numConcurrent)
		}

		avgTime := duration / time.Duration(numConcurrent)
		if avgTime > 200*time.Millisecond {
			t.Logf("Warning: Average execution time %v exceeds 200ms target", avgTime)
		}
	})
}

// TestHighVolumeStepSubmission tests performance under load
func TestHighVolumeStepSubmission(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("HighVolume_Submissions", func(t *testing.T) {
		// Create and activate funnel
		funnelID := createTestFunnel(t, testServer.Server, "High Volume Test")
		defer cleanupTestData(t, testServer.Server, funnelID)

		sessionID, slug := createTestLead(t, testServer.Server, funnelID)

		// Get first step
		req, _ := makeHTTPRequest("GET", fmt.Sprintf("/api/v1/execute/%s?session_id=%s", slug, sessionID), nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var execResponse map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &execResponse)
		step := execResponse["step"].(map[string]interface{})
		stepID := step["id"].(string)

		// Measure submission performance
		numSubmissions := 10
		var totalDuration time.Duration

		for i := 0; i < numSubmissions; i++ {
			submitData := map[string]interface{}{
				"session_id":  sessionID,
				"step_id":     stepID,
				"response":    json.RawMessage(fmt.Sprintf(`{"email":"test%d@example.com"}`, i)),
				"duration_ms": 5000,
			}

			start := time.Now()
			req, _ := makeHTTPRequest("POST", fmt.Sprintf("/api/v1/execute/%s/submit", slug), submitData)
			recorder := httptest.NewRecorder()
			testServer.Server.router.ServeHTTP(recorder, req)
			duration := time.Since(start)

			totalDuration += duration

			if recorder.Code != http.StatusOK {
				t.Logf("Submission %d failed: %d", i, recorder.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(numSubmissions)
		t.Logf("Average submission time: %v", avgDuration)

		if avgDuration > 200*time.Millisecond {
			t.Logf("Warning: Average submission time %v exceeds 200ms target", avgDuration)
		}
	})
}

// TestAnalyticsPerformance tests analytics query performance
func TestAnalyticsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Analytics_Performance", func(t *testing.T) {
		// Create funnel with some data
		funnelID := createTestFunnel(t, testServer.Server, "Analytics Performance Test")
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Create multiple leads
		for i := 0; i < 5; i++ {
			createTestLead(t, testServer.Server, funnelID)
		}

		// Measure analytics query performance
		numQueries := 20
		var totalDuration time.Duration

		for i := 0; i < numQueries; i++ {
			start := time.Now()
			req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID+"/analytics", nil)
			recorder := httptest.NewRecorder()
			testServer.Server.router.ServeHTTP(recorder, req)
			duration := time.Since(start)

			totalDuration += duration

			if recorder.Code != http.StatusOK {
				t.Errorf("Analytics query %d failed: %d", i, recorder.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(numQueries)
		t.Logf("Average analytics query time: %v", avgDuration)

		if avgDuration > 500*time.Millisecond {
			t.Logf("Warning: Average analytics time %v exceeds 500ms target", avgDuration)
		}
	})
}

// TestDatabaseConnectionPool tests connection pool efficiency
func TestDatabaseConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Connection_Pool_Efficiency", func(t *testing.T) {
		numConcurrent := 30 // Exceeds default pool size
		var wg sync.WaitGroup
		var successCount int32

		startTime := time.Now()

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req, _ := makeHTTPRequest("GET", "/api/v1/funnels", nil)
				recorder := httptest.NewRecorder()
				testServer.Server.router.ServeHTTP(recorder, req)

				if recorder.Code == http.StatusOK {
					atomic.AddInt32(&successCount, 1)
				}
			}()
		}

		wg.Wait()
		duration := time.Since(startTime)

		t.Logf("Completed %d concurrent DB queries in %v", successCount, duration)

		if successCount != int32(numConcurrent) {
			t.Errorf("Expected all %d queries to succeed, got %d", numConcurrent, successCount)
		}

		if duration > 5*time.Second {
			t.Errorf("Connection pool handling took too long: %v", duration)
		}
	})
}

// TestMemoryUsage monitors memory usage under load
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Memory_Usage_Under_Load", func(t *testing.T) {
		// Create multiple funnels
		funnelIDs := make([]string, 0, 10)
		for i := 0; i < 10; i++ {
			funnelID := createTestFunnel(t, testServer.Server, fmt.Sprintf("Memory Test Funnel %d", i))
			funnelIDs = append(funnelIDs, funnelID)
		}

		// Cleanup all funnels
		defer func() {
			for _, id := range funnelIDs {
				cleanupTestData(t, testServer.Server, id)
			}
		}()

		// Generate load
		numRequests := 100
		var wg sync.WaitGroup

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				funnelID := funnelIDs[index%len(funnelIDs)]
				req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
				recorder := httptest.NewRecorder()
				testServer.Server.router.ServeHTTP(recorder, req)
			}(i)
		}

		wg.Wait()

		// Memory usage should remain stable
		// In production, this would check actual memory metrics
		t.Log("Memory usage test completed - manual verification recommended")
	})
}

// TestThroughput measures requests per second
func TestThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Throughput_Measurement", func(t *testing.T) {
		duration := 5 * time.Second
		var requestCount int32
		stopChan := make(chan bool)

		// Start concurrent requesters
		numWorkers := 10
		var wg sync.WaitGroup

		startTime := time.Now()

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for {
					select {
					case <-stopChan:
						return
					default:
						req, _ := makeHTTPRequest("GET", "/health", nil)
						recorder := httptest.NewRecorder()
						testServer.Server.router.ServeHTTP(recorder, req)

						if recorder.Code == http.StatusOK {
							atomic.AddInt32(&requestCount, 1)
						}
					}
				}
			}()
		}

		// Run for specified duration
		time.Sleep(duration)
		close(stopChan)
		wg.Wait()

		actualDuration := time.Since(startTime)
		rps := float64(requestCount) / actualDuration.Seconds()

		t.Logf("Throughput: %.2f requests/second (%d requests in %v)", rps, requestCount, actualDuration)

		// Target: > 100 requests/second for health checks
		if rps < 100 {
			t.Logf("Warning: Throughput %.2f rps is below 100 rps target", rps)
		}
	})
}

// TestResponseTimeDistribution analyzes response time distribution
func TestResponseTimeDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Response_Time_Distribution", func(t *testing.T) {
		numRequests := 100
		responseTimes := make([]time.Duration, numRequests)

		for i := 0; i < numRequests; i++ {
			start := time.Now()
			req, _ := makeHTTPRequest("GET", "/health", nil)
			recorder := httptest.NewRecorder()
			testServer.Server.router.ServeHTTP(recorder, req)
			responseTimes[i] = time.Since(start)
		}

		// Calculate statistics
		var total time.Duration
		var min, max time.Duration = responseTimes[0], responseTimes[0]

		for _, rt := range responseTimes {
			total += rt
			if rt < min {
				min = rt
			}
			if rt > max {
				max = rt
			}
		}

		avg := total / time.Duration(numRequests)

		// Calculate p95 and p99
		// Sort response times
		sorted := make([]time.Duration, len(responseTimes))
		copy(sorted, responseTimes)
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i] > sorted[j] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		p95 := sorted[int(float64(numRequests)*0.95)]
		p99 := sorted[int(float64(numRequests)*0.99)]

		t.Logf("Response Time Statistics:")
		t.Logf("  Min: %v", min)
		t.Logf("  Avg: %v", avg)
		t.Logf("  P95: %v", p95)
		t.Logf("  P99: %v", p99)
		t.Logf("  Max: %v", max)

		if p95 > 200*time.Millisecond {
			t.Logf("Warning: P95 response time %v exceeds 200ms target", p95)
		}
	})
}
