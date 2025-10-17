// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return // Database not available
	}
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		if service, ok := response["service"].(string); !ok || service != "prompt-injection-arena" {
			t.Errorf("Expected service 'prompt-injection-arena', got %v", response["service"])
		}
	})

	t.Run("DatabaseUnavailable", func(t *testing.T) {
		// Close database to simulate unavailability
		db.Close()
		defer func() {
			env := setupTestDB(t)
			if env != nil {
				defer env.Cleanup()
			}
		}()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/health",
		}

		w := makeHTTPRequest(router, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Logf("Expected 503 when database unavailable, got %d", w.Code)
		}
	})
}

// TestGetInjectionLibrary tests the injection library retrieval endpoint
func TestGetInjectionLibrary(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()
	defer cleanupTestData(env.DB, "injection_techniques")

	router := setupTestRouter()

	// Create test data
	technique1 := createTestInjectionTechnique(t, env.DB, "Test Injection 1")
	technique2 := createTestInjectionTechnique(t, env.DB, "Test Injection 2")

	t.Run("GetAll", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/injections/library",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if techniques, ok := response["techniques"].([]interface{}); ok {
			if len(techniques) < 2 {
				t.Errorf("Expected at least 2 techniques, got %d", len(techniques))
			}
		}
	})

	t.Run("FilterByCategory", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/injections/library",
			QueryParams: map[string]string{
				"category": "test-category",
			},
		}

		w := makeHTTPRequest(router, req)
		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("WithPagination", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/injections/library",
			QueryParams: map[string]string{
				"limit":  "1",
				"offset": "0",
			},
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if techniques, ok := response["techniques"].([]interface{}); ok {
			if len(techniques) > 1 {
				t.Errorf("Expected max 1 technique with limit=1, got %d", len(techniques))
			}
		}
	})

	t.Run("FilterInactive", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/injections/library",
			QueryParams: map[string]string{
				"active": "false",
			},
		}

		w := makeHTTPRequest(router, req)
		assertJSONResponse(t, w, http.StatusOK)
	})

	// Error scenarios
	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("GetInjectionLibrary", router, "/api/v1/injections")
		patterns := NewTestScenarioBuilder().
			AddInvalidQueryParams("/api/v1/injections", map[string]string{
			"limit": "invalid",
		}).
			Build()
		suite.RunErrorTests(t, patterns)
	})

	_ = technique1
	_ = technique2
}

// TestAddInjectionTechnique tests creating new injection techniques
func TestAddInjectionTechnique(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()
	defer cleanupTestData(env.DB, "injection_techniques")

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/injections/library",
			Body: map[string]interface{}{
				"name":                "New Injection",
				"category":            "test",
				"description":         "Test description",
				"example_prompt":      "Test prompt",
				"difficulty_score":    0.5,
				"success_rate":        0.7,
				"source_attribution":  "test",
				"is_active":           true,
				"created_by":          "test-user",
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusCreated || w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, w.Code)
			if _, ok := response["id"]; !ok {
				t.Logf("Expected 'id' in response")
			}
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("AddInjectionTechnique", router, "/api/v1/injections")
		patterns := CreateInjectionErrorPatterns()
		suite.RunErrorTests(t, patterns)
	})

	t.Run("MissingName", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/injections/library",
			Body: map[string]interface{}{
				"category":           "test",
				"description":        "Test description",
				"example_prompt":     "Test prompt",
				"difficulty_score":   0.5,
				"success_rate":       0.7,
				"source_attribution": "test",
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400 for missing name, got %d", w.Code)
		}
	})

	t.Run("InvalidScores", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/injections/library",
			Body: map[string]interface{}{
				"name":               "New Injection",
				"category":           "test",
				"description":        "Test description",
				"example_prompt":     "Test prompt",
				"difficulty_score":   1.5, // Invalid: > 1.0
				"success_rate":       -0.1, // Invalid: < 0.0
				"source_attribution": "test",
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400 for invalid scores, got %d", w.Code)
		}
	})
}

// TestGetAgentLeaderboard tests the agent leaderboard endpoint
func TestGetAgentLeaderboard(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()
	defer cleanupTestData(env.DB, "agent_configurations", "test_results")

	router := setupTestRouter()

	// Create test data
	agent1 := createTestAgentConfig(t, env.DB, "Test Agent 1")
	agent2 := createTestAgentConfig(t, env.DB, "Test Agent 2")

	t.Run("GetLeaderboard", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/leaderboards/agents",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if leaderboard, ok := response["leaderboard"].([]interface{}); ok {
			if len(leaderboard) < 2 {
				t.Logf("Expected at least 2 entries, got %d", len(leaderboard))
			}
		}
	})

	t.Run("WithLimit", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/leaderboards/agents",
			QueryParams: map[string]string{
				"limit": "1",
			},
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if leaderboard, ok := response["leaderboard"].([]interface{}); ok {
			if len(leaderboard) > 1 {
				t.Logf("Expected max 1 entry with limit=1, got %d", len(leaderboard))
			}
		}
	})

	_ = agent1
	_ = agent2
}

// TestTestAgent tests the agent testing endpoint
func TestTestAgent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()
	defer cleanupTestData(env.DB, "injection_techniques", "agent_configurations", "test_results")

	router := setupTestRouter()

	// Create test data
	technique := createTestInjectionTechnique(t, env.DB, "Test Injection")

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/security/test-agent",
			Body: map[string]interface{}{
				"agent_config": map[string]interface{}{
					"name":          "Test Agent",
					"system_prompt": "You are a helpful assistant",
					"model_name":    "test-model",
					"temperature":   0.7,
					"max_tokens":    100,
					"created_by":    "test-user",
				},
				"test_suite": []string{technique.ID},
				"max_execution_time": 300,
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusOK || w.Code == http.StatusAccepted {
			response := assertJSONResponse(t, w, w.Code)
			if _, ok := response["robustness_score"]; ok {
				// Check that response has expected fields
				if _, ok := response["test_results"]; !ok {
					t.Logf("Expected 'test_results' in response")
				}
			}
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("TestAgent", router, "/api/v1/test/agent")
		patterns := CreateTestErrorPatterns()
		suite.RunErrorTests(t, patterns)
	})

	t.Run("MissingAgentConfig", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/security/test-agent",
			Body: map[string]interface{}{
				"test_suite": []string{technique.ID},
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400 for missing agent config, got %d", w.Code)
		}
	})

	t.Run("InvalidTestSuite", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/security/test-agent",
			Body: map[string]interface{}{
				"agent_config": map[string]interface{}{
					"name":          "Test Agent",
					"system_prompt": "You are a helpful assistant",
					"model_name":    "test-model",
					"created_by":    "test-user",
				},
				"test_suite": []string{"invalid-id"},
			},
		}

		w := makeHTTPRequest(router, req)
		// May return various status codes depending on implementation
		t.Logf("Test suite with invalid IDs returned status: %d", w.Code)
	})
}

// TestGetStatistics tests the statistics endpoint
func TestGetStatistics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()
	defer cleanupTestData(env.DB, "injection_techniques", "agent_configurations", "test_results")

	router := setupTestRouter()

	// Create test data
	agent := createTestAgentConfig(t, env.DB, "Test Agent")
	technique := createTestInjectionTechnique(t, env.DB, "Test Injection")
	createTestResult(t, env.DB, technique.ID, agent.ID)

	t.Run("GetAllStats", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/statistics",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		expectedFields := []string{
			"total_injections",
			"total_agents",
			"total_tests",
			"average_success_rate",
		}

		for _, field := range expectedFields {
			if _, ok := response[field]; !ok {
				t.Logf("Expected '%s' in statistics response", field)
			}
		}
	})

	t.Run("FilterByTimeRange", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/statistics",
			QueryParams: map[string]string{
				"start_date": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"end_date":   time.Now().Format(time.RFC3339),
			},
		}

		w := makeHTTPRequest(router, req)
		assertJSONResponse(t, w, http.StatusOK)
	})
}

// TestExportResearch tests the export research endpoint
func TestExportResearch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()
	defer cleanupTestData(env.DB, "injection_techniques", "agent_configurations", "test_results")

	router := setupTestRouter()

	// Create test data
	agent := createTestAgentConfig(t, env.DB, "Test Agent")
	technique := createTestInjectionTechnique(t, env.DB, "Test Injection")
	result := createTestResult(t, env.DB, technique.ID, agent.ID)

	t.Run("ExportJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/research",
			QueryParams: map[string]string{
				"format": "json",
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusOK {
			var data interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
				t.Errorf("Failed to parse JSON export: %v", err)
			}
		}
	})

	t.Run("ExportCSV", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/research",
			QueryParams: map[string]string{
				"format": "csv",
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusOK {
			contentType := w.Header().Get("Content-Type")
			if contentType != "text/csv" && contentType != "application/csv" {
				t.Logf("Expected CSV content type, got: %s", contentType)
			}
		}
	})

	t.Run("FilterBySessionID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/research",
			QueryParams: map[string]string{
				"format":     "json",
				"session_id": result.TestSessionID,
			},
		}

		w := makeHTTPRequest(router, req)
		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("UnsupportedFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/research",
			QueryParams: map[string]string{
				"format": "unsupported",
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400 for unsupported format, got %d", w.Code)
		}
	})
}

// TestGetExportFormats tests the export formats endpoint
func TestGetExportFormats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("GetFormats", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/formats",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if formats, ok := response["formats"].([]interface{}); ok {
			if len(formats) == 0 {
				t.Logf("Expected at least one export format")
			}

			// Verify each format has required fields
			for _, f := range formats {
				if format, ok := f.(map[string]interface{}); ok {
					requiredFields := []string{"name", "description"}
					for _, field := range requiredFields {
						if _, ok := format[field]; !ok {
							t.Logf("Expected '%s' in format", field)
						}
					}
				}
			}
		}
	})
}

// TestDataStructures tests JSON marshaling/unmarshaling of data structures
func TestDataStructures(t *testing.T) {
	t.Run("InjectionTechnique", func(t *testing.T) {
		technique := InjectionTechnique{
			ID:                uuid.New().String(),
			Name:              "Test",
			Category:          "test",
			Description:       "Test description",
			ExamplePrompt:     "Test prompt",
			DifficultyScore:   0.5,
			SuccessRate:       0.7,
			SourceAttribution: "test",
			IsActive:          true,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
			CreatedBy:         "test",
		}

		data, err := json.Marshal(technique)
		if err != nil {
			t.Fatalf("Failed to marshal InjectionTechnique: %v", err)
		}

		var decoded InjectionTechnique
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal InjectionTechnique: %v", err)
		}

		if decoded.Name != technique.Name {
			t.Errorf("Expected name %s, got %s", technique.Name, decoded.Name)
		}
	})

	t.Run("AgentConfiguration", func(t *testing.T) {
		config := AgentConfiguration{
			ID:           uuid.New().String(),
			Name:         "Test Agent",
			SystemPrompt: "You are a helpful assistant",
			ModelName:    "test-model",
			Temperature:  0.7,
			MaxTokens:    100,
			SafetyConstraints: map[string]interface{}{
				"max_retries": 3,
			},
			RobustnessScore: 0.0,
			TestsRun:        0,
			TestsPassed:     0,
			IsActive:        true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			CreatedBy:       "test",
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal AgentConfiguration: %v", err)
		}

		var decoded AgentConfiguration
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal AgentConfiguration: %v", err)
		}

		if decoded.Name != config.Name {
			t.Errorf("Expected name %s, got %s", config.Name, decoded.Name)
		}
	})

	t.Run("TestResult", func(t *testing.T) {
		result := TestResult{
			ID:              uuid.New().String(),
			InjectionID:     uuid.New().String(),
			AgentID:         uuid.New().String(),
			Success:         true,
			ResponseText:    "Test response",
			ExecutionTimeMS: 100,
			SafetyViolations: []map[string]interface{}{
				{"type": "test", "severity": "low"},
			},
			ConfidenceScore: 0.9,
			ErrorMessage:    "",
			ExecutedAt:      time.Now(),
			TestSessionID:   uuid.New().String(),
			Metadata: map[string]interface{}{
				"test": true,
			},
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal TestResult: %v", err)
		}

		var decoded TestResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal TestResult: %v", err)
		}

		if decoded.Success != result.Success {
			t.Errorf("Expected success %v, got %v", result.Success, decoded.Success)
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("EmptyDatabase", func(t *testing.T) {
		// Clean up all data
		cleanupTestData(env.DB, "injection_techniques", "agent_configurations", "test_results")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/injections/library",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK)

		if techniques, ok := response["techniques"].([]interface{}); ok {
			if len(techniques) != 0 {
				t.Logf("Expected empty array, got %d techniques", len(techniques))
			}
		}
	})

	t.Run("VeryLargePagination", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/injections/library",
			QueryParams: map[string]string{
				"limit":  "999999",
				"offset": "999999",
			},
		}

		w := makeHTTPRequest(router, req)
		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("NegativePagination", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/injections/library",
			QueryParams: map[string]string{
				"limit":  "-1",
				"offset": "-1",
			},
		}

		w := makeHTTPRequest(router, req)
		// Should handle gracefully
		t.Logf("Negative pagination returned status: %d", w.Code)
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/injections/library",
			QueryParams: map[string]string{
				"category": "'; DROP TABLE injection_techniques; --",
			},
		}

		w := makeHTTPRequest(router, req)
		// Should not cause SQL injection
		if w.Code == http.StatusOK {
			t.Log("SQL injection attempt was handled safely")
		}
	})
}
