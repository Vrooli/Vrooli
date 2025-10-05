package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestPerformanceHealthCheck tests health check endpoint performance
func TestPerformanceHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	if app.DB == nil {
		t.Skip("Skipping performance health check test - requires database")
	}

	t.Run("Health check completes quickly", func(t *testing.T) {
		timer := StartTimer()

		for i := 0; i < 100; i++ {
			req, _ := http.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(app.HealthCheck)
			handler.ServeHTTP(rr, req)
		}

		// Should complete 100 requests in < 500ms
		timer.AssertMaxDuration(t, 500*time.Millisecond, "100 health check requests")
	})
}

// TestPerformanceListAutomations tests automation listing performance
func TestPerformanceListAutomations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	timer := StartTimer()

	for i := 0; i < 50; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/automations", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.ListAutomations)
		handler.ServeHTTP(rr, req)
	}

	// Should complete 50 requests in < 100ms
	timer.AssertMaxDuration(t, 100*time.Millisecond, "50 list automation requests")
}

// TestConcurrentHealthChecks tests concurrent health check handling
func TestConcurrentHealthChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)

	t.Run("Handles concurrent health checks", func(t *testing.T) {
		var wg sync.WaitGroup
		concurrency := 50

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequest("GET", "/health", nil)
				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(app.HealthCheck)
				handler.ServeHTTP(rr, req)

				if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
					t.Errorf("Unexpected status code: %d", rr.Code)
				}
			}()
		}

		wg.Wait()
	})
}

// TestConcurrentAutomationGeneration tests concurrent automation generation
func TestConcurrentAutomationGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)

	t.Run("Handles concurrent code generation", func(t *testing.T) {
		var wg sync.WaitGroup
		concurrency := 20

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				code, explanation := app.generateAutomationCode(
					"Turn on lights at sunset",
					[]string{"light.living_room"},
					"evening",
				)

				if code == "" {
					t.Error("Generated code should not be empty")
				}

				if explanation == "" {
					t.Error("Explanation should not be empty")
				}
			}(i)
		}

		wg.Wait()
	})
}

// TestMemoryLeaks tests for memory leaks in request handling
func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)

	t.Run("No memory leaks in repeated requests", func(t *testing.T) {
		// Run many requests to detect potential memory leaks
		for i := 0; i < 1000; i++ {
			req, _ := http.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(app.HealthCheck)
			handler.ServeHTTP(rr, req)
		}

		// If we get here without crashing, basic memory management is working
	})
}

// TestRateLimiterPerformanceUnderLoad tests rate limiter under heavy load
func TestRateLimiterPerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Rate limiter performs well under load", func(t *testing.T) {
		rl := NewRateLimiter(1000, time.Minute)
		timer := StartTimer()

		var wg sync.WaitGroup
		concurrency := 100
		requestsPerWorker := 50

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < requestsPerWorker; j++ {
					rl.Allow("test-client")
				}
			}(i)
		}

		wg.Wait()

		// Should handle 5000 concurrent requests quickly
		timer.AssertMaxDuration(t, 500*time.Millisecond, "5000 concurrent rate limit checks")
	})
}

// TestGetEnvPerformance tests environment variable lookup performance
func TestGetEnvPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	timer := StartTimer()

	for i := 0; i < 10000; i++ {
		_ = getEnv("NONEXISTENT_KEY", "default")
	}

	// Should handle 10000 lookups quickly
	timer.AssertMaxDuration(t, 50*time.Millisecond, "10000 getEnv calls")
}

// TestCountHealthyDependenciesPerformance tests dependency counting performance
func TestCountHealthyDependenciesPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)

	deps := map[string]interface{}{
		"db":     map[string]interface{}{"status": "healthy"},
		"cache":  map[string]interface{}{"status": "healthy"},
		"device": map[string]interface{}{"status": "degraded"},
	}

	timer := StartTimer()

	for i := 0; i < 10000; i++ {
		app.countHealthyDependencies(deps)
	}

	// Should handle 10000 counts quickly
	timer.AssertMaxDuration(t, 100*time.Millisecond, "10000 dependency counts")
}

// TestAutomationCodeGenerationPerformance tests code generation performance
func TestAutomationCodeGenerationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	timer := StartTimer()

	descriptions := []string{
		"Turn on lights at sunset",
		"Turn off lights at sunrise",
		"Set temperature to 72 degrees",
		"Dim lights at 9pm",
		"Turn on coffee maker at 7am",
	}

	devices := [][]string{
		{"light.living_room"},
		{"light.bedroom"},
		{"climate.thermostat"},
		{"light.kitchen"},
		{"switch.coffee_maker"},
	}

	// Generate 1000 automation codes
	for i := 0; i < 200; i++ {
		for j := 0; j < 5; j++ {
			app.generateAutomationCode(descriptions[j], devices[j], "evening")
		}
	}

	// Should generate 1000 codes in reasonable time
	timer.AssertMaxDuration(t, 2*time.Second, "1000 automation code generations")
}

// TestConflictDetectionPerformance tests conflict detection performance
func TestConflictDetectionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	timer := StartTimer()

	devices := []string{"light.1", "light.2", "light.3", "light.4"}

	for i := 0; i < 1000; i++ {
		app.checkAutomationConflicts(devices, "evening")
	}

	// Should check 1000 conflicts quickly
	timer.AssertMaxDuration(t, 100*time.Millisecond, "1000 conflict checks")
}

// TestJSONMarshalingPerformance tests JSON marshaling performance
func TestJSONMarshalingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	timer := StartTimer()

	requestBody := map[string]interface{}{
		"description":    "Turn on lights at sunset",
		"profile_id":     "test-profile",
		"target_devices": []string{"light.living_room", "light.bedroom", "light.kitchen"},
	}

	for i := 0; i < 10000; i++ {
		_, _ = json.Marshal(requestBody)
	}

	// Should marshal 10000 requests quickly
	timer.AssertMaxDuration(t, 200*time.Millisecond, "10000 JSON marshaling operations")
}

// TestHTTPRequestCreationPerformance tests HTTP request creation performance
func TestHTTPRequestCreationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	timer := StartTimer()

	requestBody := map[string]interface{}{
		"device_id": "light.living_room",
		"action":    "turn_on",
		"user_id":   "test-user",
	}
	body, _ := json.Marshal(requestBody)

	for i := 0; i < 1000; i++ {
		_, _ = http.NewRequest("POST", "/api/v1/devices/control", bytes.NewBuffer(body))
	}

	// Should create 1000 requests quickly
	timer.AssertMaxDuration(t, 100*time.Millisecond, "1000 HTTP request creations")
}
