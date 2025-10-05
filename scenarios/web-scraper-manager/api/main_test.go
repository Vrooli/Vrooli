// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func TestMain(m *testing.M) {
	// Setup test environment
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "8080")

	// Initialize database connection for tests
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL != "" {
		var err error
		db, err = sql.Open("postgres", postgresURL)
		if err != nil {
			log.Printf("Warning: Failed to connect to test database: %v", err)
			log.Println("Database-dependent tests will be skipped")
		} else {
			// Test connection
			if err := db.Ping(); err != nil {
				log.Printf("Warning: Database ping failed: %v", err)
				log.Println("Database-dependent tests will be skipped")
				db.Close()
				db = nil
			} else {
				log.Println("✅ Test database connection established")
			}
		}
	} else {
		log.Println("Warning: POSTGRES_URL not set, database-dependent tests will be skipped")
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if db != nil {
		db.Close()
	}

	os.Exit(code)
}

// Test configuration loading
func TestLoadConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("LoadConfigWithIndividualComponents", func(t *testing.T) {
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "test")
		os.Setenv("POSTGRES_PASSWORD", "test")
		os.Setenv("POSTGRES_DB", "testdb")
		os.Setenv("API_PORT", "8080")

		cfg := loadConfig()

		if cfg.Port != "8080" {
			t.Errorf("Expected port 8080, got %s", cfg.Port)
		}

		if cfg.PostgresURL == "" {
			t.Error("Expected PostgresURL to be set")
		}
	})

	t.Run("LoadConfigWithPostgresURL", func(t *testing.T) {
		os.Setenv("POSTGRES_URL", "postgres://user:pass@localhost:5432/db")
		os.Setenv("API_PORT", "9000")

		cfg := loadConfig()

		if cfg.Port != "9000" {
			t.Errorf("Expected port 9000, got %s", cfg.Port)
		}

		if cfg.PostgresURL != "postgres://user:pass@localhost:5432/db" {
			t.Errorf("Unexpected PostgresURL: %s", cfg.PostgresURL)
		}
	})

	t.Run("LoadConfigWithDefaults", func(t *testing.T) {
		os.Setenv("POSTGRES_URL", "postgres://localhost/test")
		os.Setenv("API_PORT", "8080")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("MINIO_URL")

		cfg := loadConfig()

		if cfg.RedisURL != "redis://localhost:6379" {
			t.Errorf("Expected default Redis URL, got %s", cfg.RedisURL)
		}

		if cfg.MinioURL != "http://localhost:9000" {
			t.Errorf("Expected default Minio URL, got %s", cfg.MinioURL)
		}
	})
}

// Test health endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("HealthSuccess", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		healthHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success=true")
		}

		if response.Data == nil {
			t.Error("Expected data in response")
		}
	})
}

// Test CORS middleware
func TestCorsMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CORSHeaders", func(t *testing.T) {
		handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS origin header")
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected CORS methods header")
		}
	})

	t.Run("OptionsRequest", func(t *testing.T) {
		handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})
}

// Test agent handlers
func TestGetAgentsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("GetAllAgents", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/agents", nil)
		w := httptest.NewRecorder()

		getAgentsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got error: %s", response.Error)
		}
	})

	t.Run("GetAgentsWithPlatformFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/agents?platform=huginn", nil)
		w := httptest.NewRecorder()

		getAgentsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("GetAgentsWithEnabledFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/agents?enabled=true", nil)
		w := httptest.NewRecorder()

		getAgentsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestCreateAgentHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("CreateAgentSuccess", func(t *testing.T) {
		agent := ScrapingAgent{
			Name:          "Test Agent",
			Platform:      "huginn",
			AgentType:     "WebsiteAgent",
			Configuration: map[string]interface{}{"url": "https://example.com"},
			Enabled:       true,
			Tags:          []string{"test"},
		}

		body, _ := json.Marshal(agent)
		req := httptest.NewRequest("POST", "/api/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createAgentHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got error: %s", response.Error)
		}
	})

	t.Run("CreateAgentInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/agents", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createAgentHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("CreateAgentMissingName", func(t *testing.T) {
		agent := ScrapingAgent{
			Platform: "huginn",
		}

		body, _ := json.Marshal(agent)
		req := httptest.NewRequest("POST", "/api/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createAgentHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestGetAgentHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("GetAgentNotFound", func(t *testing.T) {
		agentID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/agents/"+agentID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": agentID})
		w := httptest.NewRecorder()

		getAgentHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestUpdateAgentHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("UpdateAgentNotFound", func(t *testing.T) {
		agentID := uuid.New().String()
		agent := ScrapingAgent{
			Name:     "Updated Agent",
			Platform: "huginn",
		}

		body, _ := json.Marshal(agent)
		req := httptest.NewRequest("PUT", "/api/agents/"+agentID, bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"id": agentID})
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		updateAgentHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("UpdateAgentInvalidJSON", func(t *testing.T) {
		agentID := uuid.New().String()
		req := httptest.NewRequest("PUT", "/api/agents/"+agentID, bytes.NewBufferString("invalid"))
		req = mux.SetURLVars(req, map[string]string{"id": agentID})
		w := httptest.NewRecorder()

		updateAgentHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestDeleteAgentHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("DeleteAgentNotFound", func(t *testing.T) {
		agentID := uuid.New().String()
		req := httptest.NewRequest("DELETE", "/api/agents/"+agentID, nil)
		req = mux.SetURLVars(req, map[string]string{"id": agentID})
		w := httptest.NewRecorder()

		deleteAgentHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// Test target handlers
func TestCreateTargetHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("CreateTargetInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/targets", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()

		createTargetHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("CreateTargetMissingFields", func(t *testing.T) {
		target := ScrapingTarget{
			URL: "https://example.com",
		}

		body, _ := json.Marshal(target)
		req := httptest.NewRequest("POST", "/api/targets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createTargetHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// Test result handlers
func TestGetResultsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("GetAllResults", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/results", nil)
		w := httptest.NewRecorder()

		getResultsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("GetResultsWithStatusFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/results?status=success", nil)
		w := httptest.NewRecorder()

		getResultsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("GetResultsWithLimit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/results?limit=10", nil)
		w := httptest.NewRecorder()

		getResultsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// Test platform capabilities
func TestGetPlatformsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GetPlatforms", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/platforms", nil)
		w := httptest.NewRecorder()

		getPlatformsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success=true")
		}

		platforms, ok := response.Data.([]interface{})
		if !ok {
			t.Fatal("Expected platforms array")
		}

		if len(platforms) == 0 {
			t.Error("Expected at least one platform")
		}
	})
}

// Test execution handlers
func TestExecuteAgentHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ExecuteAgent", func(t *testing.T) {
		agentID := uuid.New().String()
		req := httptest.NewRequest("POST", "/api/agents/"+agentID+"/execute", nil)
		req = mux.SetURLVars(req, map[string]string{"id": agentID})
		w := httptest.NewRecorder()

		executeAgentHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success=true")
		}
	})
}

func TestExecuteWorkflowHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ExecuteWorkflow", func(t *testing.T) {
		workflowID := uuid.New().String()
		req := httptest.NewRequest("POST", "/api/workflows/"+workflowID+"/execute", nil)
		req = mux.SetURLVars(req, map[string]string{"id": workflowID})
		w := httptest.NewRecorder()

		executeWorkflowHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// Test export handler
func TestExportDataHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ExportDataSuccess", func(t *testing.T) {
		exportReq := map[string]interface{}{
			"agent_ids": []string{uuid.New().String()},
			"format":    "json",
			"filters":   map[string]interface{}{},
		}

		body, _ := json.Marshal(exportReq)
		req := httptest.NewRequest("POST", "/api/export", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		exportDataHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ExportDataInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/export", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()

		exportDataHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// Test metrics handler
func TestGetMetricsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("GetMetrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/metrics", nil)
		w := httptest.NewRecorder()

		getMetricsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success=true")
		}
	})
}

// Test status handler
func TestGetStatusHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("GetStatus", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/status", nil)
		w := httptest.NewRecorder()

		getStatusHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success=true")
		}

		// Data is interface{}, need to handle properly
		if response.Data == nil {
			t.Error("Expected status data")
		}
	})
}

// Test respondJSON helper
func TestRespondJSON(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RespondJSONSuccess", func(t *testing.T) {
		w := httptest.NewRecorder()
		response := APIResponse{
			Success: true,
			Data:    map[string]string{"test": "value"},
			Message: "Test message",
		}

		respondJSON(w, http.StatusOK, response)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}

		var decoded APIResponse
		if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !decoded.Success {
			t.Error("Expected success=true")
		}
	})

	t.Run("RespondJSONError", func(t *testing.T) {
		w := httptest.NewRecorder()
		response := APIResponse{
			Success: false,
			Error:   "Test error",
		}

		respondJSON(w, http.StatusBadRequest, response)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var decoded APIResponse
		if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if decoded.Success {
			t.Error("Expected success=false")
		}

		if decoded.Error != "Test error" {
			t.Errorf("Expected error 'Test error', got '%s'", decoded.Error)
		}
	})
}

// Test getEnv helper
func TestGetEnv(t *testing.T) {
	t.Run("GetEnvWithValue", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "test_value")
		defer os.Unsetenv("TEST_ENV_VAR")

		value := getEnv("TEST_ENV_VAR", "default")
		if value != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", value)
		}
	})

	t.Run("GetEnvWithDefault", func(t *testing.T) {
		os.Unsetenv("TEST_ENV_VAR")

		value := getEnv("TEST_ENV_VAR", "default_value")
		if value != "default_value" {
			t.Errorf("Expected 'default_value', got '%s'", value)
		}
	})
}

// Test getTargetsHandler
func TestGetTargetsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("GetAllTargets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/targets", nil)
		w := httptest.NewRecorder()

		getTargetsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got error: %s", response.Error)
		}
	})
}

// Test getAgentTargetsHandler
func TestGetAgentTargetsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("GetTargetsForAgent", func(t *testing.T) {
		agentID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/agents/"+agentID+"/targets", nil)
		req = mux.SetURLVars(req, map[string]string{"agentId": agentID})
		w := httptest.NewRecorder()

		getAgentTargetsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got error: %s", response.Error)
		}
	})
}

// Test getAgentResultsHandler
func TestGetAgentResultsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("GetResultsForAgent", func(t *testing.T) {
		agentID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/agents/"+agentID+"/results", nil)
		req = mux.SetURLVars(req, map[string]string{"agentId": agentID})
		w := httptest.NewRecorder()

		getAgentResultsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got error: %s", response.Error)
		}
	})

	t.Run("GetResultsWithStatusFilter", func(t *testing.T) {
		agentID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/agents/"+agentID+"/results?status=success", nil)
		req = mux.SetURLVars(req, map[string]string{"agentId": agentID})
		w := httptest.NewRecorder()

		getAgentResultsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("GetResultsWithDateRange", func(t *testing.T) {
		agentID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/agents/"+agentID+"/results?start_date=2024-01-01&end_date=2024-12-31", nil)
		req = mux.SetURLVars(req, map[string]string{"agentId": agentID})
		w := httptest.NewRecorder()

		getAgentResultsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// Test createTargetHandler success path
func TestCreateTargetHandlerSuccess(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("CreateTargetSuccess", func(t *testing.T) {
		// First create an agent to reference
		testAgent := createTestAgent(t, "Test Agent", "huginn")
		defer testAgent.Cleanup()

		target := ScrapingTarget{
			AgentID: testAgent.Agent.ID,
			URL:     "https://example.com",
			SelectorConfig: map[string]interface{}{
				"title": "h1",
			},
			RateLimitMs: 1000,
			MaxRetries:  3,
		}

		body, _ := json.Marshal(target)
		req := httptest.NewRequest("POST", "/api/targets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createTargetHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got error: %s", response.Error)
		}
	})
}

// Test loadConfig edge cases
func TestLoadConfigEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("LoadConfigWithEmptyPostgresHost", func(t *testing.T) {
		os.Setenv("POSTGRES_HOST", "")
		os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/test")
		os.Setenv("API_PORT", "8080")

		cfg := loadConfig()

		// Should use POSTGRES_URL
		if cfg.PostgresURL != "postgres://test:test@localhost:5432/test" {
			t.Errorf("Expected to use POSTGRES_URL, got %s", cfg.PostgresURL)
		}
	})

	t.Run("LoadConfigWithAllComponentsSet", func(t *testing.T) {
		os.Setenv("POSTGRES_HOST", "dbhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")
		os.Setenv("API_PORT", "9090")
		os.Setenv("REDIS_URL", "redis://custom:6380")
		os.Setenv("MINIO_URL", "http://custom:9001")

		cfg := loadConfig()

		if cfg.Port != "9090" {
			t.Errorf("Expected port 9090, got %s", cfg.Port)
		}

		if cfg.RedisURL != "redis://custom:6380" {
			t.Errorf("Expected custom Redis URL, got %s", cfg.RedisURL)
		}

		if cfg.MinioURL != "http://custom:9001" {
			t.Errorf("Expected custom Minio URL, got %s", cfg.MinioURL)
		}
	})
}

// Test healthHandler database error
func TestHealthHandlerDatabaseError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("HealthWithDatabase", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		healthHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success=true")
		}

		// Verify database status is included
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		if data["database"] != "connected" {
			t.Errorf("Expected database status 'connected', got %v", data["database"])
		}
	})
}

// Test getAgentsHandler with various query parameters
func TestGetAgentsHandlerFilters(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("GetAgentsWithMultipleFilters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/agents?platform=huginn&enabled=true&limit=5", nil)
		w := httptest.NewRecorder()

		getAgentsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("GetAgentsWithInvalidLimit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/agents?limit=invalid", nil)
		w := httptest.NewRecorder()

		getAgentsHandler(w, req)

		// Should still return 200 with default limit
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("GetAgentsWithSearchQuery", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/agents?search=test", nil)
		w := httptest.NewRecorder()

		getAgentsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}
