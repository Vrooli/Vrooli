package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEndToEndScenarioFlow tests the complete scenario workflow
func TestEndToEndScenarioFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Set up test scenarios
	kidScenarios = []Scenario{
		{
			ID:          "retro-game-launcher",
			Name:        "retro-game-launcher",
			Title:       "Retro Games",
			Description: "Play classic arcade games!",
			Icon:        "🕹️",
			Color:       "bg-gradient-to-br from-purple-500 to-pink-500",
			Category:    "games",
			Port:        3301,
			AgeRange:    "5-12",
		},
		{
			ID:          "study-buddy",
			Name:        "study-buddy",
			Title:       "Study Buddy",
			Description: "A friendly helper for homework!",
			Icon:        "📚",
			Color:       "bg-gradient-to-br from-teal-400 to-cyan-500",
			Category:    "learn",
			Port:        3304,
			AgeRange:    "9-12",
		},
	}

	t.Run("CompleteUserJourney", func(t *testing.T) {
		// Step 1: Check health
		healthReq := httptest.NewRequest("GET", "/health", nil)
		healthW := httptest.NewRecorder()
		healthHandler(healthW, healthReq)

		if healthW.Code != http.StatusOK {
			t.Errorf("Health check failed with status %d", healthW.Code)
		}

		// Step 2: List all scenarios
		listReq := httptest.NewRequest("GET", "/api/v1/kids/scenarios", nil)
		listW := httptest.NewRecorder()
		scenariosHandler(listW, listReq)

		if listW.Code != http.StatusOK {
			t.Errorf("List scenarios failed with status %d", listW.Code)
		}

		var listResponse map[string]interface{}
		json.NewDecoder(listW.Body).Decode(&listResponse)

		scenarios := listResponse["scenarios"].([]interface{})
		if len(scenarios) != 2 {
			t.Errorf("Expected 2 scenarios, got %d", len(scenarios))
		}

		// Step 3: Filter by category
		filterReq := httptest.NewRequest("GET", "/api/v1/kids/scenarios?category=games", nil)
		filterW := httptest.NewRecorder()
		scenariosHandler(filterW, filterReq)

		var filterResponse map[string]interface{}
		json.NewDecoder(filterW.Body).Decode(&filterResponse)

		filteredScenarios := filterResponse["scenarios"].([]interface{})
		if len(filteredScenarios) != 1 {
			t.Errorf("Expected 1 game scenario, got %d", len(filteredScenarios))
		}

		// Step 4: Launch a scenario
		launchBody := strings.NewReader(`{"scenarioId":"retro-game-launcher"}`)
		launchReq := httptest.NewRequest("POST", "/api/v1/kids/launch", launchBody)
		launchReq.Header.Set("Content-Type", "application/json")
		launchW := httptest.NewRecorder()
		launchHandler(launchW, launchReq)

		if launchW.Code != http.StatusOK {
			t.Errorf("Launch failed with status %d", launchW.Code)
		}

		var launchResponse map[string]interface{}
		json.NewDecoder(launchW.Body).Decode(&launchResponse)

		if launchResponse["url"] != "http://localhost:3301" {
			t.Errorf("Expected URL http://localhost:3301, got %v", launchResponse["url"])
		}
	})
}

// TestServiceConfigParsing tests parsing of service.json files
func TestServiceConfigParsing(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	testCases := []struct {
		name           string
		config         ServiceConfig
		expectedResult bool
	}{
		{
			name: "ValidKidFriendlyConfig",
			config: ServiceConfig{
				Name:        "test-game",
				Description: "A fun game",
				Category:    []string{"games", "kid-friendly"},
				Deployment: struct {
					Port int `json:"port"`
				}{Port: 3500},
			},
			expectedResult: true,
		},
		{
			name: "ValidWithMetadataTags",
			config: ServiceConfig{
				Name:        "test-app",
				Description: "A test app",
				Category:    []string{"utilities"},
				Metadata: struct {
					Tags           []string `json:"tags"`
					TargetAudience struct {
						AgeRange string `json:"ageRange"`
					} `json:"targetAudience"`
				}{
					Tags: []string{"kid-friendly"},
				},
				Deployment: struct {
					Port int `json:"port"`
				}{Port: 3600},
			},
			expectedResult: true,
		},
		{
			name: "InvalidBlacklistedCategory",
			config: ServiceConfig{
				Name:        "admin-panel",
				Description: "Admin tools",
				Category:    []string{"admin"},
				Deployment: struct {
					Port int `json:"port"`
				}{Port: 3700},
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create config file
			configDir := filepath.Join(env.TempDir, "test-scenario", ".vrooli")
			os.MkdirAll(configDir, 0755)

			configPath := filepath.Join(configDir, "service.json")
			configData, _ := json.MarshalIndent(tc.config, "", "  ")
			ioutil.WriteFile(configPath, configData, 0644)

			// Read and parse
			data, err := ioutil.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read config: %v", err)
			}

			var parsedConfig ServiceConfig
			if err := json.Unmarshal(data, &parsedConfig); err != nil {
				t.Fatalf("Failed to parse config: %v", err)
			}

			result := isKidFriendly(parsedConfig)
			if result != tc.expectedResult {
				t.Errorf("Expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("LaunchWithMissingContentType", func(t *testing.T) {
		kidScenarios = []Scenario{{ID: "test", Port: 3000}}

		body := strings.NewReader(`{"scenarioId":"test"}`)
		req := httptest.NewRequest("POST", "/api/v1/kids/launch", body)
		// Intentionally not setting Content-Type
		w := httptest.NewRecorder()

		launchHandler(w, req)

		// Should still work or return appropriate error
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status %d", w.Code)
		}
	})

	t.Run("LaunchWithEmptyBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/kids/launch", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		launchHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("ScenariosWithNoScenarios", func(t *testing.T) {
		kidScenarios = []Scenario{}

		req := httptest.NewRequest("GET", "/api/v1/kids/scenarios", nil)
		w := httptest.NewRecorder()

		scenariosHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		json.NewDecoder(w.Body).Decode(&response)

		if int(response["count"].(float64)) != 0 {
			t.Errorf("Expected count 0, got %v", response["count"])
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ScenarioWithSpecialCharacters", func(t *testing.T) {
		kidScenarios = []Scenario{
			{
				ID:          "test-scenario!@#",
				Name:        "Test Scenario!@#",
				Title:       "Special <>&\" Characters",
				Description: "Contains special chars: <>&\"'",
				Category:    "games",
				Port:        3000,
				AgeRange:    "5-12",
			},
		}

		req := httptest.NewRequest("GET", "/api/v1/kids/scenarios", nil)
		w := httptest.NewRecorder()

		scenariosHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("ScenarioWithVeryLongStrings", func(t *testing.T) {
		longString := strings.Repeat("a", 10000)
		kidScenarios = []Scenario{
			{
				ID:          longString,
				Name:        longString,
				Title:       longString,
				Description: longString,
				Category:    "games",
				Port:        3000,
				AgeRange:    "5-12",
			},
		}

		req := httptest.NewRequest("GET", "/api/v1/kids/scenarios", nil)
		w := httptest.NewRecorder()

		scenariosHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("FilterWithEmptyStrings", func(t *testing.T) {
		scenarios := []Scenario{
			{ID: "test", Category: "", AgeRange: ""},
		}

		result := filterScenarios(scenarios, "", "")
		if len(result) != 1 {
			t.Errorf("Expected 1 scenario, got %d", len(result))
		}
	})

	t.Run("MultipleConsecutiveLaunches", func(t *testing.T) {
		kidScenarios = []Scenario{{ID: "test", Port: 3000}}

		for i := 0; i < 5; i++ {
			body := strings.NewReader(`{"scenarioId":"test"}`)
			req := httptest.NewRequest("POST", "/api/v1/kids/launch", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			launchHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Launch %d failed with status %d", i, w.Code)
			}
		}
	})
}
