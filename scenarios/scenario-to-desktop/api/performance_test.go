package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestHealthEndpointPerformance tests health check performance
func TestHealthEndpointPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("SequentialRequests", func(t *testing.T) {
		requestCount := 100
		start := time.Now()

		for i := 0; i < requestCount; i++ {
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			w := httptest.NewRecorder()
			server.healthHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Completed %d health checks in %v (avg: %v per request)", requestCount, elapsed, avgTime)

		// Performance target: < 10ms average per request
		if avgTime > 10*time.Millisecond {
			t.Logf("WARNING: Average response time (%v) exceeds 10ms target", avgTime)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		requestCount := 100
		concurrency := 10

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < requestCount/concurrency; j++ {
					req := httptest.NewRequest("GET", "/api/v1/health", nil)
					w := httptest.NewRecorder()
					server.healthHandler(w, req)

					if w.Code != 200 {
						t.Errorf("Worker %d request %d failed", workerID, j)
					}
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Completed %d concurrent health checks in %v (avg: %v per request)", requestCount, elapsed, avgTime)

		// Performance target: < 5ms average with concurrency
		if avgTime > 5*time.Millisecond {
			t.Logf("WARNING: Average concurrent response time (%v) exceeds 5ms target", avgTime)
		}
	})

	t.Run("SustainedLoad", func(t *testing.T) {
		duration := 5 * time.Second
		var requestCount int
		var failCount int
		var mu sync.Mutex

		start := time.Now()
		done := make(chan bool)

		// Spawn multiple workers
		workerCount := 5
		for i := 0; i < workerCount; i++ {
			go func() {
				for {
					select {
					case <-done:
						return
					default:
						req := httptest.NewRequest("GET", "/api/v1/health", nil)
						w := httptest.NewRecorder()
						server.healthHandler(w, req)

						mu.Lock()
						requestCount++
						if w.Code != 200 {
							failCount++
						}
						mu.Unlock()
					}
				}
			}()
		}

		// Run for specified duration
		time.Sleep(duration)
		close(done)
		elapsed := time.Since(start)

		// Allow workers to finish
		time.Sleep(100 * time.Millisecond)

		t.Logf("Sustained load: %d requests in %v (%.0f req/s, %d failures)",
			requestCount, elapsed, float64(requestCount)/elapsed.Seconds(), failCount)

		if failCount > 0 {
			t.Errorf("Expected zero failures under sustained load, got %d", failCount)
		}
	})
}

// TestStatusEndpointPerformance tests status endpoint performance
func TestStatusEndpointPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	// Add build statuses for realistic testing
	for i := 0; i < 50; i++ {
		buildID := uuid.New().String()
		status := "building"
		if i%3 == 0 {
			status = "ready"
		} else if i%5 == 0 {
			status = "failed"
		}
		server.buildStatuses[buildID] = createTestBuildStatus(buildID, status)
	}

	t.Run("SequentialRequests", func(t *testing.T) {
		requestCount := 50
		start := time.Now()

		for i := 0; i < requestCount; i++ {
			req := httptest.NewRequest("GET", "/api/v1/status", nil)
			w := httptest.NewRecorder()
			server.statusHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Completed %d status checks in %v (avg: %v per request)", requestCount, elapsed, avgTime)

		// Performance target: < 20ms average per request (more complex than health)
		if avgTime > 20*time.Millisecond {
			t.Logf("WARNING: Average response time (%v) exceeds 20ms target", avgTime)
		}
	})
}

// TestTemplateListingPerformance tests template listing performance
func TestTemplateListingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SequentialRequests", func(t *testing.T) {
		requestCount := 30
		start := time.Now()

		for i := 0; i < requestCount; i++ {
			req := httptest.NewRequest("GET", "/api/v1/templates", nil)
			w := httptest.NewRecorder()
			env.Server.listTemplatesHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Completed %d template listings in %v (avg: %v per request)", requestCount, elapsed, avgTime)

		// Performance target: < 15ms average per request
		if avgTime > 15*time.Millisecond {
			t.Logf("WARNING: Average response time (%v) exceeds 15ms target", avgTime)
		}
	})
}

// TestGenerateDesktopPerformance tests desktop generation performance
func TestGenerateDesktopPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SequentialGeneration", func(t *testing.T) {
		requestCount := 10
		start := time.Now()

		for i := 0; i < requestCount; i++ {
			body := createValidGenerateRequest()
			body["app_name"] = fmt.Sprintf("PerfTest%d", i)
			body["output_path"] = filepath.Join(env.TempDir, fmt.Sprintf("output-%d", i))

			req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
			w := httptest.NewRecorder()
			env.Server.generateDesktopHandler(w, req)

			if w.Code != 201 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(requestCount)

		t.Logf("Completed %d desktop generations in %v (avg: %v per request)", requestCount, elapsed, avgTime)

		// Performance target: < 100ms average per generation request (not including async build)
		if avgTime > 100*time.Millisecond {
			t.Logf("WARNING: Average generation request time (%v) exceeds 100ms target", avgTime)
		}

		// Verify all builds were created
		if len(env.Server.buildStatuses) != requestCount {
			t.Errorf("Expected %d build statuses, got %d", requestCount, len(env.Server.buildStatuses))
		}
	})

	t.Run("ConcurrentGeneration", func(t *testing.T) {
		requestCount := 12 // Changed to 12 so it divides evenly by 3 workers
		concurrency := 3

		var wg sync.WaitGroup
		start := time.Now()
		successCount := 0
		var mu sync.Mutex

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < requestCount/concurrency; j++ {
					body := createValidGenerateRequest()
					body["app_name"] = fmt.Sprintf("ConcPerfTest%d-%d", workerID, j)
					body["output_path"] = filepath.Join(env.TempDir, fmt.Sprintf("output-%d-%d", workerID, j))

					req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
					w := httptest.NewRecorder()
					env.Server.generateDesktopHandler(w, req)

					if w.Code == 201 {
						mu.Lock()
						successCount++
						mu.Unlock()
					}
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Completed %d concurrent generations in %v (%d successful)", requestCount, elapsed, successCount)

		if successCount != requestCount {
			t.Errorf("Expected %d successful generations, got %d", requestCount, successCount)
		}
	})
}

// TestBuildStatusPerformance tests build status lookup performance
func TestBuildStatusPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	// Create many build statuses
	buildIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		buildID := uuid.New().String()
		buildIDs[i] = buildID
		server.buildStatuses[buildID] = createTestBuildStatus(buildID, "ready")
	}

	t.Run("SequentialLookups", func(t *testing.T) {
		start := time.Now()

		for _, buildID := range buildIDs {
			req := httptest.NewRequest("GET", "/api/v1/desktop/status/"+buildID, nil)
			req = mux.SetURLVars(req, map[string]string{"build_id": buildID})
			w := httptest.NewRecorder()
			server.getBuildStatusHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Lookup for build %s failed with status %d", buildID, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(len(buildIDs))

		t.Logf("Completed %d build status lookups in %v (avg: %v per lookup)", len(buildIDs), elapsed, avgTime)

		// Performance target: < 5ms average per lookup
		if avgTime > 5*time.Millisecond {
			t.Logf("WARNING: Average lookup time (%v) exceeds 5ms target", avgTime)
		}
	})

	t.Run("ConcurrentLookups", func(t *testing.T) {
		var wg sync.WaitGroup
		start := time.Now()

		// Lookup all builds concurrently
		for _, buildID := range buildIDs {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				req := httptest.NewRequest("GET", "/api/v1/desktop/status/"+id, nil)
				req = mux.SetURLVars(req, map[string]string{"build_id": id})
				w := httptest.NewRecorder()
				server.getBuildStatusHandler(w, req)

				if w.Code != 200 {
					t.Errorf("Concurrent lookup for build %s failed", id)
				}
			}(buildID)
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Completed %d concurrent build status lookups in %v", len(buildIDs), elapsed)

		// All lookups should complete quickly even when concurrent
		if elapsed > 1*time.Second {
			t.Logf("WARNING: Concurrent lookups took %v (> 1s)", elapsed)
		}
	})
}

// TestMemoryUsage tests memory usage under load
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("BuildStatusAccumulation", func(t *testing.T) {
		// Create many build statuses to test memory usage
		buildCount := 1000
		start := time.Now()

		for i := 0; i < buildCount; i++ {
			body := createValidGenerateRequest()
			body["app_name"] = fmt.Sprintf("MemTest%d", i)
			body["output_path"] = filepath.Join(env.TempDir, fmt.Sprintf("output-%d", i))

			req := createJSONRequest("POST", "/api/v1/desktop/generate", body)
			w := httptest.NewRecorder()
			env.Server.generateDesktopHandler(w, req)

			if w.Code != 201 {
				t.Errorf("Request %d failed", i)
			}
		}

		elapsed := time.Since(start)

		t.Logf("Created %d build statuses in %v", buildCount, elapsed)

		// Verify all builds exist
		if len(env.Server.buildStatuses) != buildCount {
			t.Errorf("Expected %d build statuses, got %d", buildCount, len(env.Server.buildStatuses))
		}

		// Test accessing random builds
		testCount := 100
		accessStart := time.Now()
		for i := 0; i < testCount; i++ {
			req := httptest.NewRequest("GET", "/api/v1/status", nil)
			w := httptest.NewRecorder()
			env.Server.statusHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Status check %d failed", i)
			}
		}
		accessElapsed := time.Since(accessStart)

		t.Logf("Accessed status %d times with %d builds in memory: %v (avg: %v)",
			testCount, buildCount, accessElapsed, accessElapsed/time.Duration(testCount))
	})
}

// TestValidationPerformance tests configuration validation performance
func TestValidationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)
	baseConfig := func(name string) *DesktopConfig {
		return &DesktopConfig{
			AppName:      name,
			Framework:    "electron",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
			ServerType:   "static",
			ServerPath:   "./dist",
			APIEndpoint:  "http://localhost:3000",
		}
	}

	t.Run("ValidConfigValidation", func(t *testing.T) {
		validationCount := 10000
		start := time.Now()

		for i := 0; i < validationCount; i++ {
			config := baseConfig(fmt.Sprintf("App%d", i))

			err := server.validateDesktopConfig(config)
			if err != nil {
				t.Errorf("Validation %d failed: %v", i, err)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(validationCount)

		t.Logf("Completed %d validations in %v (avg: %v per validation)", validationCount, elapsed, avgTime)

		// Performance target: < 1µs average per validation
		if avgTime > 1*time.Microsecond {
			t.Logf("Average validation time: %v", avgTime)
		}
	})

	t.Run("InvalidConfigValidation", func(t *testing.T) {
		validationCount := 1000
		start := time.Now()

		for i := 0; i < validationCount; i++ {
			config := baseConfig(fmt.Sprintf("App%d", i))
			config.Framework = "invalid"

			err := server.validateDesktopConfig(config)
			if err == nil {
				t.Errorf("Expected validation %d to fail", i)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(validationCount)

		t.Logf("Completed %d invalid validations in %v (avg: %v per validation)", validationCount, elapsed, avgTime)
	})
}

// TestWebhookPerformance tests webhook handling performance
func TestWebhookPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	// Create build statuses
	buildIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		buildID := uuid.New().String()
		buildIDs[i] = buildID
		server.buildStatuses[buildID] = createTestBuildStatus(buildID, "building")
	}

	t.Run("SequentialWebhooks", func(t *testing.T) {
		start := time.Now()

		for _, buildID := range buildIDs {
			body := map[string]interface{}{
				"status": "completed",
			}

			req := createJSONRequest("POST", "/api/v1/desktop/webhook/build-complete", body)
			req.Header.Set("X-Build-ID", buildID)
			w := httptest.NewRecorder()
			server.buildCompleteWebhookHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Webhook for build %s failed with status %d", buildID, w.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(len(buildIDs))

		t.Logf("Processed %d webhooks in %v (avg: %v per webhook)", len(buildIDs), elapsed, avgTime)

		// Performance target: < 10ms average per webhook
		if avgTime > 10*time.Millisecond {
			t.Logf("WARNING: Average webhook time (%v) exceeds 10ms target", avgTime)
		}

		// Verify all statuses were updated
		completedCount := 0
		for _, status := range server.buildStatuses {
			if status.Status == "completed" {
				completedCount++
			}
		}

		if completedCount != len(buildIDs) {
			t.Errorf("Expected %d completed builds, got %d", len(buildIDs), completedCount)
		}
	})
}

// TestEndToEndPerformance tests complete workflow performance
func TestEndToEndPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("CompleteWorkflow", func(t *testing.T) {
		workflowCount := 5
		start := time.Now()

		for i := 0; i < workflowCount; i++ {
			// Step 1: Generate desktop app
			generateBody := createValidGenerateRequest()
			generateBody["app_name"] = fmt.Sprintf("E2ETest%d", i)
			generateBody["output_path"] = filepath.Join(env.TempDir, fmt.Sprintf("e2e-%d", i))

			generateReq := createJSONRequest("POST", "/api/v1/desktop/generate", generateBody)
			generateW := httptest.NewRecorder()
			env.Server.generateDesktopHandler(generateW, generateReq)

			if generateW.Code != 201 {
				t.Errorf("Workflow %d: Generate failed with status %d", i, generateW.Code)
				continue
			}

			// Extract build ID
			var generateResp map[string]interface{}
			if err := json.Unmarshal(generateW.Body.Bytes(), &generateResp); err != nil {
				t.Errorf("Workflow %d: Failed to parse response: %v", i, err)
				continue
			}

			buildID := generateResp["build_id"].(string)

			// Step 2: Check build status
			statusReq := httptest.NewRequest("GET", "/api/v1/desktop/status/"+buildID, nil)
			statusReq = mux.SetURLVars(statusReq, map[string]string{"build_id": buildID})
			statusW := httptest.NewRecorder()
			env.Server.getBuildStatusHandler(statusW, statusReq)

			if statusW.Code != 200 {
				t.Errorf("Workflow %d: Status check failed with status %d", i, statusW.Code)
			}

			// Step 3: Send webhook to complete
			webhookBody := map[string]interface{}{
				"status": "completed",
			}
			webhookReq := createJSONRequest("POST", "/api/v1/desktop/webhook/build-complete", webhookBody)
			webhookReq.Header.Set("X-Build-ID", buildID)
			webhookW := httptest.NewRecorder()
			env.Server.buildCompleteWebhookHandler(webhookW, webhookReq)

			if webhookW.Code != 200 {
				t.Errorf("Workflow %d: Webhook failed with status %d", i, webhookW.Code)
			}
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(workflowCount)

		t.Logf("Completed %d end-to-end workflows in %v (avg: %v per workflow)", workflowCount, elapsed, avgTime)

		// Performance target: < 200ms average per complete workflow
		if avgTime > 200*time.Millisecond {
			t.Logf("WARNING: Average workflow time (%v) exceeds 200ms target", avgTime)
		}
	})
}
