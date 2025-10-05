package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPerformanceLargeScenarioList tests performance with many scenarios
func TestPerformanceLargeScenarioList(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create 100 test scenarios
	scenarios := make([]Scenario, 100)
	for i := 0; i < 100; i++ {
		scenarios[i] = Scenario{
			ID:          strings.Repeat("a", i%10+1),
			Name:        strings.Repeat("name", i%5+1),
			Title:       "Test Scenario",
			Description: "Performance test scenario",
			Icon:        "🎮",
			Color:       "bg-blue-500",
			Category:    "games",
			Port:        3000 + i,
			AgeRange:    "5-12",
		}
	}

	kidScenarios = scenarios

	t.Run("ListAllScenarios", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/kids/scenarios", nil)
		w := httptest.NewRecorder()

		scenariosHandler(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.NewDecoder(w.Body).Decode(&response)

		if int(response["count"].(float64)) != 100 {
			t.Errorf("Expected count 100, got %v", response["count"])
		}
	})

	t.Run("FilterPerformance", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/kids/scenarios?category=games", nil)
		w := httptest.NewRecorder()

		scenariosHandler(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestConcurrentRequests tests thread safety
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	kidScenarios = []Scenario{
		{
			ID:       "test-scenario",
			Name:     "test-scenario",
			Category: "games",
			Port:     3000,
			AgeRange: "5-12",
		},
	}

	done := make(chan bool)

	// Launch 10 concurrent requests
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/v1/kids/scenarios", nil)
			w := httptest.NewRecorder()
			scenariosHandler(w, req)
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// BenchmarkScenariosHandler benchmarks the scenarios handler
func BenchmarkScenariosHandler(b *testing.B) {
	kidScenarios = []Scenario{
		{ID: "test1", Category: "games", AgeRange: "5-12"},
		{ID: "test2", Category: "learn", AgeRange: "9-12"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/kids/scenarios", nil)
		w := httptest.NewRecorder()
		scenariosHandler(w, req)
	}
}

// BenchmarkLaunchHandler benchmarks the launch handler
func BenchmarkLaunchHandler(b *testing.B) {
	kidScenarios = []Scenario{
		{ID: "test-scenario", Port: 3000},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := strings.NewReader(`{"scenarioId":"test-scenario"}`)
		req := httptest.NewRequest("POST", "/api/v1/kids/launch", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		launchHandler(w, req)
	}
}

// BenchmarkFilterScenarios benchmarks the filter function
func BenchmarkFilterScenarios(b *testing.B) {
	scenarios := make([]Scenario, 100)
	for i := 0; i < 100; i++ {
		scenarios[i] = Scenario{
			ID:       "test",
			Category: "games",
			AgeRange: "5-12",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterScenarios(scenarios, "5-12", "games")
	}
}
