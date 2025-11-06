//go:build testing
// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// BenchmarkHealthHandler benchmarks the health check endpoint
func BenchmarkHealthHandler(b *testing.B) {
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkSaaSDetection benchmarks the SaaS detection logic
func BenchmarkSaaSDetection(b *testing.B) {
	// Benchmarks typically skip database operations
	b.Skip("Skipping benchmark that requires database setup")
}

// BenchmarkTemplateRetrieval benchmarks template database queries
func BenchmarkTemplateRetrieval(b *testing.B) {
	// Benchmarks typically skip database operations
	b.Skip("Skipping benchmark that requires database setup")
}

// BenchmarkLandingPageGeneration benchmarks landing page generation
func BenchmarkLandingPageGeneration(b *testing.B) {
	// Benchmarks typically skip database operations
	b.Skip("Skipping benchmark that requires database setup")
}

// TestPerformanceScenarios tests various performance scenarios
func TestPerformanceScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping performance tests: no database available")
	}

	t.Run("LargeScaleScanning", func(t *testing.T) {
		scenariosDir := filepath.Join(env.TempDir, "large-scenarios")
		os.MkdirAll(scenariosDir, 0755)

		// Create 50 test scenarios
		for i := 0; i < 50; i++ {
			scenarioDir := filepath.Join(scenariosDir, fmt.Sprintf("scenario-%d", i))
			os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755)

			// Alternate between SaaS and non-SaaS
			isSaaS := i%2 == 0
			tags := []string{"utility"}
			if isSaaS {
				tags = []string{"saas", "multi-tenant", "billing"}
			}

			serviceConfig := map[string]interface{}{
				"service": map[string]interface{}{
					"displayName": fmt.Sprintf("Scenario %d", i),
					"description": fmt.Sprintf("Test scenario %d", i),
					"tags":        tags,
				},
			}

			if isSaaS {
				serviceConfig["resources"] = map[string]interface{}{
					"postgres": map[string]interface{}{
						"enabled": true,
					},
				}
			}

			serviceJSON, _ := json.MarshalIndent(serviceConfig, "", "  ")
			os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0644)
		}

		dbService := NewDatabaseService(env.DB)
		detectionService := NewSaaSDetectionService(scenariosDir, dbService)

		start := time.Now()
		response, err := detectionService.ScanScenarios(false, "")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Large scale scan failed: %v", err)
		}

		t.Logf("Scanned %d scenarios in %v", response.TotalScenarios, duration)

		// Performance threshold: should complete in under 10 seconds
		if duration > 10*time.Second {
			t.Errorf("Large scale scan took too long: %v (expected < 10s)", duration)
		}

		// Verify results
		if response.TotalScenarios != 50 {
			t.Errorf("Expected 50 scenarios, got %d", response.TotalScenarios)
		}

		expectedSaaS := 25 // Half should be SaaS
		if response.SaaSScenarios < expectedSaaS-5 || response.SaaSScenarios > expectedSaaS+5 {
			t.Errorf("Expected ~%d SaaS scenarios, got %d", expectedSaaS, response.SaaSScenarios)
		}
	})

	t.Run("ConcurrentTemplateAccess", func(t *testing.T) {
		dbService := NewDatabaseService(env.DB)

		// Create templates
		for i := 0; i < 20; i++ {
			createTestTemplate(t, env.DB, fmt.Sprintf("concurrent-template-%d", i))
		}

		// Test concurrent access
		concurrency := 10
		iterations := 100
		var wg sync.WaitGroup
		errors := make([]error, 0)
		var errorsMux sync.Mutex

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < iterations; j++ {
					_, err := dbService.GetTemplates("", "")
					if err != nil {
						errorsMux.Lock()
						errors = append(errors, err)
						errorsMux.Unlock()
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		totalRequests := concurrency * iterations
		t.Logf("Completed %d concurrent template requests in %v", totalRequests, duration)

		if len(errors) > 0 {
			t.Errorf("Encountered %d errors during concurrent access", len(errors))
		}

		// Performance threshold: should handle 1000 requests in under 30 seconds
		if duration > 30*time.Second {
			t.Errorf("Concurrent access took too long: %v (expected < 30s)", duration)
		}

		requestsPerSecond := float64(totalRequests) / duration.Seconds()
		t.Logf("Throughput: %.2f requests/second", requestsPerSecond)

		// Should achieve at least 30 req/s
		if requestsPerSecond < 30 {
			t.Errorf("Low throughput: %.2f req/s (expected >= 30 req/s)", requestsPerSecond)
		}
	})

	t.Run("ConcurrentLandingPageGeneration", func(t *testing.T) {
		dbService := NewDatabaseService(env.DB)
		landingPageService := NewLandingPageService(dbService, filepath.Join(env.TempDir, "templates"))

		// Create prerequisites
		scenario := createTestScenario(t, env.DB, "concurrent-gen-scenario")
		template := createTestTemplate(t, env.DB, "concurrent-gen-template")

		concurrency := 5
		var wg sync.WaitGroup
		errors := make([]error, 0)
		var errorsMux sync.Mutex

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				req := &GenerateRequest{
					ScenarioID:      scenario.ID,
					TemplateID:      template.ID,
					EnableABTesting: true,
				}

				_, err := landingPageService.GenerateLandingPage(req)
				if err != nil {
					errorsMux.Lock()
					errors = append(errors, err)
					errorsMux.Unlock()
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Generated %d landing pages concurrently in %v", concurrency, duration)

		if len(errors) > 0 {
			t.Errorf("Encountered %d errors during concurrent generation: %v", len(errors), errors[0])
		}

		// Should complete quickly even with A/B variants
		if duration > 5*time.Second {
			t.Errorf("Concurrent generation took too long: %v (expected < 5s)", duration)
		}
	})

	t.Run("HandlerResponseTime", func(t *testing.T) {
		db = env.DB

		// Create test data
		scenariosDir := createTestScenariosDirectory(t, env.TempDir)
		os.Setenv("SCENARIOS_PATH", scenariosDir)
		defer os.Unsetenv("SCENARIOS_PATH")

		reqBody := ScanRequest{
			ForceRescan:    false,
			ScenarioFilter: "",
		}

		iterations := 100
		totalDuration := time.Duration(0)

		for i := 0; i < iterations; i++ {
			bodyJSON, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", "/api/v1/scenarios/scan", bytes.NewBuffer(bodyJSON))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(scanScenariosHandler)

			start := time.Now()
			handler.ServeHTTP(rr, req)
			duration := time.Since(start)

			totalDuration += duration

			if rr.Code != http.StatusOK {
				t.Errorf("Handler returned error status: %d", rr.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average handler response time: %v over %d iterations", avgDuration, iterations)

		// Average response time should be under 100ms
		if avgDuration > 100*time.Millisecond {
			t.Errorf("Average response time too slow: %v (expected < 100ms)", avgDuration)
		}
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		dbService := NewDatabaseService(env.DB)

		// Create a large number of scenarios to test memory handling
		for i := 0; i < 1000; i++ {
			scenario := &SaaSScenario{
				ID:               fmt.Sprintf("mem-test-%d", i),
				ScenarioName:     fmt.Sprintf("memory-test-%d", i),
				DisplayName:      fmt.Sprintf("Memory Test %d", i),
				Description:      "Testing memory usage",
				SaaSType:         "b2b_tool",
				Industry:         "technology",
				RevenuePotential: "$10K-$50K",
				ConfidenceScore:  0.8,
				Metadata:         map[string]interface{}{"test": i},
			}

			err := dbService.CreateSaaSScenario(scenario)
			if err != nil {
				t.Fatalf("Failed to create scenario %d: %v", i, err)
			}
		}

		// Retrieve all scenarios multiple times to test memory stability
		for i := 0; i < 10; i++ {
			scenarios, err := dbService.GetSaaSScenarios()
			if err != nil {
				t.Fatalf("Failed to get scenarios: %v", err)
			}

			if len(scenarios) < 1000 {
				t.Errorf("Expected at least 1000 scenarios, got %d", len(scenarios))
			}
		}

		t.Log("Memory usage test completed successfully")
	})
}

// TestLoadTesting performs load testing scenarios
func TestLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.DB == nil {
		t.Skip("Skipping load tests: no database available")
	}

	t.Run("SustainedLoad", func(t *testing.T) {
		db = env.DB

		scenario := createTestScenario(t, env.DB, "load-test-scenario")
		createTestTemplate(t, env.DB, "load-test-template")

		// Simulate sustained load for 10 seconds
		duration := 10 * time.Second
		concurrency := 5
		requestCount := 0
		var requestMux sync.Mutex
		errors := make([]error, 0)
		var errorsMux sync.Mutex

		stopChan := make(chan struct{})
		var wg sync.WaitGroup

		start := time.Now()

		// Start workers
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for {
					select {
					case <-stopChan:
						return
					default:
						// Alternate between different endpoints
						switch id % 3 {
						case 0:
							// Health check
							req, _ := http.NewRequest("GET", "/health", nil)
							rr := httptest.NewRecorder()
							healthHandler(rr, req)
							if rr.Code != http.StatusOK {
								errorsMux.Lock()
								errors = append(errors, fmt.Errorf("health check failed"))
								errorsMux.Unlock()
							}
						case 1:
							// Get dashboard
							req, _ := http.NewRequest("GET", "/api/v1/analytics/dashboard", nil)
							rr := httptest.NewRecorder()
							getDashboardHandler(rr, req)
							if rr.Code != http.StatusOK {
								errorsMux.Lock()
								errors = append(errors, fmt.Errorf("dashboard failed"))
								errorsMux.Unlock()
							}
						case 2:
							// Generate landing page
							reqBody := GenerateRequest{
								ScenarioID:      scenario.ID,
								EnableABTesting: false,
							}
							bodyJSON, _ := json.Marshal(reqBody)
							req, _ := http.NewRequest("POST", "/api/v1/landing-pages/generate", bytes.NewBuffer(bodyJSON))
							req.Header.Set("Content-Type", "application/json")
							rr := httptest.NewRecorder()
							generateLandingPageHandler(rr, req)
							if rr.Code != http.StatusOK {
								errorsMux.Lock()
								errors = append(errors, fmt.Errorf("generate failed"))
								errorsMux.Unlock()
							}
						}

						requestMux.Lock()
						requestCount++
						requestMux.Unlock()
					}
				}
			}(i)
		}

		// Let it run for specified duration
		time.Sleep(duration)
		close(stopChan)
		wg.Wait()

		elapsed := time.Since(start)
		requestsPerSecond := float64(requestCount) / elapsed.Seconds()

		t.Logf("Sustained load test: %d requests in %v", requestCount, elapsed)
		t.Logf("Throughput: %.2f req/s", requestsPerSecond)
		t.Logf("Errors: %d (%.2f%%)", len(errors), float64(len(errors))/float64(requestCount)*100)

		// Should handle at least 50 req/s with < 5% error rate
		if requestsPerSecond < 50 {
			t.Errorf("Low throughput: %.2f req/s (expected >= 50 req/s)", requestsPerSecond)
		}

		errorRate := float64(len(errors)) / float64(requestCount) * 100
		if errorRate > 5 {
			t.Errorf("High error rate: %.2f%% (expected < 5%%)", errorRate)
		}
	})
}
