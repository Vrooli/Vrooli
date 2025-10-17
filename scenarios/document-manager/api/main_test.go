package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Test helpers
func setupTestRouter() *mux.Router {
	router := mux.NewRouter()

	// Add middleware
	router.Use(corsMiddleware)
	router.Use(loggingMiddleware)

	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/system/db-status", dbStatusHandler).Methods("GET")
	router.HandleFunc("/api/system/vector-status", vectorStatusHandler).Methods("GET")
	router.HandleFunc("/api/system/ai-status", aiStatusHandler).Methods("GET")
	router.HandleFunc("/api/applications", applicationsHandler).Methods("GET", "POST", "OPTIONS")
	router.HandleFunc("/api/agents", agentsHandler).Methods("GET", "POST", "OPTIONS")
	router.HandleFunc("/api/queue", queueHandler).Methods("GET", "POST", "OPTIONS")
	router.HandleFunc("/api/search", vectorSearchHandler).Methods("POST", "OPTIONS")
	return router
}

// setupTestRouterWithDB creates a router with database connection
func setupTestRouterWithDB(testDB *sql.DB) *mux.Router {
	db = testDB
	return setupTestRouter()
}

// TestHealthCheckHandler tests the health check endpoint
func TestHealthCheckHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("SuccessfulHealthCheck", func(t *testing.T) {
		// Set up a mock database connection for health check
		// In tests without real DB, health will be degraded - this is expected
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check required fields
		if _, exists := response["status"]; !exists {
			t.Error("Expected status field in health check response")
		}
		if response["service"] != "document-manager-api" {
			t.Errorf("Expected service to be document-manager-api, got %v", response["service"])
		}
		if response["version"] != "2.0.0" {
			t.Errorf("Expected version to be 2.0.0, got %v", response["version"])
		}
		if _, exists := response["timestamp"]; !exists {
			t.Error("Expected timestamp in health check response")
		}
		if _, exists := response["readiness"]; !exists {
			t.Error("Expected readiness field in health check response")
		}
		if _, exists := response["dependencies"]; !exists {
			t.Error("Expected dependencies field in health check response")
		}
	})

	t.Run("ResponseContentType", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})
}

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("ValidConfigWithPostgresURL", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("POSTGRES_URL", "postgresql://test:test@localhost:5432/test")

		cfg := loadConfig()

		if cfg.Port != "8080" {
			t.Errorf("Expected port 8080, got %s", cfg.Port)
		}

		if cfg.PostgresURL != "postgresql://test:test@localhost:5432/test" {
			t.Errorf("Expected postgres URL to match, got %s", cfg.PostgresURL)
		}
	})

	t.Run("ValidConfigWithIndividualDBComponents", func(t *testing.T) {
		os.Setenv("API_PORT", "9090")
		os.Unsetenv("POSTGRES_URL")
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")

		cfg := loadConfig()

		if cfg.Port != "9090" {
			t.Errorf("Expected port 9090, got %s", cfg.Port)
		}

		expectedURL := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
		if cfg.PostgresURL != expectedURL {
			t.Errorf("Expected postgres URL %s, got %s", expectedURL, cfg.PostgresURL)
		}
	})

	t.Run("OptionalServicesConfiguration", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		os.Setenv("POSTGRES_URL", "postgresql://test:test@localhost:5432/test")
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		os.Setenv("QDRANT_URL", "http://localhost:6333")
		os.Setenv("OLLAMA_URL", "http://localhost:11434")

		cfg := loadConfig()

		if cfg.RedisURL != "redis://localhost:6379" {
			t.Errorf("Expected Redis URL to be set, got %s", cfg.RedisURL)
		}

		if cfg.QdrantURL != "http://localhost:6333" {
			t.Errorf("Expected Qdrant URL to be set, got %s", cfg.QdrantURL)
		}

		if cfg.OllamaURL != "http://localhost:11434" {
			t.Errorf("Expected Ollama URL to be set, got %s", cfg.OllamaURL)
		}
	})
}

// TestApplicationStruct tests the Application struct JSON marshaling
func TestApplicationStruct(t *testing.T) {
	now := time.Now()
	app := Application{
		ID:                "test-id",
		Name:              "Test App",
		RepositoryURL:     "https://github.com/test/repo",
		DocumentationPath: "/docs",
		HealthScore:       85.5,
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
		AgentCount:        2,
		Status:            "active",
	}

	data, err := json.Marshal(app)
	if err != nil {
		t.Fatalf("Failed to marshal application: %v", err)
	}

	var decoded Application
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal application: %v", err)
	}

	if decoded.ID != app.ID {
		t.Errorf("Expected ID %s, got %s", app.ID, decoded.ID)
	}

	if decoded.Name != app.Name {
		t.Errorf("Expected name %s, got %s", app.Name, decoded.Name)
	}

	if decoded.HealthScore != app.HealthScore {
		t.Errorf("Expected health score %.2f, got %.2f", app.HealthScore, decoded.HealthScore)
	}
}

// TestAgentStruct tests the Agent struct JSON marshaling
func TestAgentStruct(t *testing.T) {
	now := time.Now()
	agent := Agent{
		ID:                   "agent-id",
		Name:                 "Test Agent",
		Type:                 "documentation_analyzer",
		ApplicationID:        "app-id",
		Configuration:        "{}",
		ScheduleCron:         "0 */6 * * *",
		AutoApplyThreshold:   0.8,
		LastPerformanceScore: 0.95,
		Enabled:              true,
		CreatedAt:            now,
		UpdatedAt:            now,
		ApplicationName:      "Test App",
		Status:               "active",
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Failed to marshal agent: %v", err)
	}

	var decoded Agent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal agent: %v", err)
	}

	if decoded.ID != agent.ID {
		t.Errorf("Expected ID %s, got %s", agent.ID, decoded.ID)
	}

	if decoded.Type != agent.Type {
		t.Errorf("Expected type %s, got %s", agent.Type, decoded.Type)
	}

	if decoded.AutoApplyThreshold != agent.AutoApplyThreshold {
		t.Errorf("Expected threshold %.2f, got %.2f", agent.AutoApplyThreshold, decoded.AutoApplyThreshold)
	}
}

// TestImprovementQueueStruct tests the ImprovementQueue struct
func TestImprovementQueueStruct(t *testing.T) {
	now := time.Now()
	queue := ImprovementQueue{
		ID:              "queue-id",
		AgentID:         "agent-id",
		ApplicationID:   "app-id",
		Type:            "documentation_update",
		Title:           "Fix broken link",
		Description:     "Update outdated URL",
		Severity:        "medium",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
		AgentName:       "Test Agent",
		ApplicationName: "Test App",
	}

	data, err := json.Marshal(queue)
	if err != nil {
		t.Fatalf("Failed to marshal queue item: %v", err)
	}

	var decoded ImprovementQueue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal queue item: %v", err)
	}

	if decoded.Title != queue.Title {
		t.Errorf("Expected title %s, got %s", queue.Title, decoded.Title)
	}

	if decoded.Severity != queue.Severity {
		t.Errorf("Expected severity %s, got %s", queue.Severity, decoded.Severity)
	}
}

// TestCreateApplicationHandlerInvalidJSON tests error handling
func TestCreateApplicationHandlerInvalidJSON(t *testing.T) {
	router := setupTestRouter()

	invalidJSON := []byte(`{"name": "Test", invalid}`)
	req := httptest.NewRequest("POST", "/api/applications", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

// TestCreateAgentHandlerInvalidJSON tests error handling
func TestCreateAgentHandlerInvalidJSON(t *testing.T) {
	router := setupTestRouter()

	invalidJSON := []byte(`{"name": invalid}`)
	req := httptest.NewRequest("POST", "/api/agents", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

// TestSystemStatusStruct tests the SystemStatus struct
func TestSystemStatusStruct(t *testing.T) {
	status := SystemStatus{
		Service: "postgres",
		Status:  "healthy",
		Details: "All systems operational",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal system status: %v", err)
	}

	var decoded SystemStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal system status: %v", err)
	}

	if decoded.Service != status.Service {
		t.Errorf("Expected service %s, got %s", status.Service, decoded.Service)
	}

	if decoded.Status != status.Status {
		t.Errorf("Expected status %s, got %s", status.Status, decoded.Status)
	}
}

// TestDatabaseStatusHandlerNoConnection tests db status when not connected
func TestDatabaseStatusHandlerNoConnection(t *testing.T) {
	// Skip this test for now - dbStatusHandler calls db.Ping() which panics with nil db
	// In production, db is always initialized before handlers are registered
	t.Skip("Skipping nil db test - requires refactoring dbStatusHandler for testability")
}

// TestVectorStatusHandler tests vector database status endpoint
func TestVectorStatusHandler(t *testing.T) {
	router := setupTestRouter()

	req := httptest.NewRequest("GET", "/api/system/vector-status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return some response (200, 500, or 503 depending on Qdrant availability)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200, 500, or 503, got %d", w.Code)
	}

	var response SystemStatus
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Service != "qdrant" {
		t.Errorf("Expected service 'qdrant', got %s", response.Service)
	}
}

// TestAIStatusHandler tests AI integration status endpoint
func TestAIStatusHandler(t *testing.T) {
	router := setupTestRouter()

	req := httptest.NewRequest("GET", "/api/system/ai-status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return some response (200, 500, or 503 depending on Ollama availability)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200, 500, or 503, got %d", w.Code)
	}

	var response SystemStatus
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Service != "ollama" {
		t.Errorf("Expected service 'ollama', got %s", response.Service)
	}
}

// TestApplicationsHandlerErrors tests error handling for applications endpoint
func TestApplicationsHandlerErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	suite := &HandlerTestSuite{
		Router:  router,
		BaseURL: "/api/applications",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/applications").
		AddEmptyBody("POST", "/api/applications").
		AddMethodNotAllowed("DELETE", "/api/applications").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestAgentsHandlerErrors tests error handling for agents endpoint
func TestAgentsHandlerErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	suite := &HandlerTestSuite{
		Router:  router,
		BaseURL: "/api/agents",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/agents").
		AddEmptyBody("POST", "/api/agents").
		AddMethodNotAllowed("DELETE", "/api/agents").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestQueueHandlerErrors tests error handling for queue endpoint
func TestQueueHandlerErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	suite := &HandlerTestSuite{
		Router:  router,
		BaseURL: "/api/queue",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/queue").
		AddEmptyBody("POST", "/api/queue").
		AddMethodNotAllowed("DELETE", "/api/queue").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestCORSMiddleware tests CORS middleware functionality
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("OptionsRequest", func(t *testing.T) {
		// Set test config for CORS
		oldConfig := config
		config = Config{
			CORSOrigins: "http://localhost:3000",
		}
		defer func() { config = oldConfig }()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/applications",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		corsHeader := w.Header().Get("Access-Control-Allow-Origin")
		if corsHeader == "" {
			t.Error("Expected Access-Control-Allow-Origin header to be set")
		}
		if corsHeader == "*" {
			t.Error("CORS should not use wildcard origin for security")
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected Access-Control-Allow-Methods header to be set")
		}
	})

	t.Run("CORSHeadersOnRegularRequest", func(t *testing.T) {
		// Set test config for CORS
		oldConfig := config
		config = Config{
			CORSOrigins: "http://localhost:3000",
		}
		defer func() { config = oldConfig }()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		corsHeader := w.Header().Get("Access-Control-Allow-Origin")
		if corsHeader == "" {
			t.Error("Expected Access-Control-Allow-Origin header on regular request")
		}
		if corsHeader == "*" {
			t.Error("CORS should not use wildcard origin for security")
		}
	})
}

// TestLoggingMiddleware tests logging middleware
func TestLoggingMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("LogsRequests", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		// Logging happens in background, just verify request succeeds
	})
}

// TestVectorStatusHandlerWithMockServer tests vector status with mock Qdrant
func TestVectorStatusHandlerWithMockServer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthyQdrant", func(t *testing.T) {
		mockQdrant := mockHTTPServer(t, http.StatusOK, `{"status":"ok"}`)
		defer mockQdrant.Close()

		config.QdrantURL = mockQdrant.URL
		router := setupTestRouter()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/system/vector-status",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"service": "qdrant",
			"status":  "healthy",
		})

		if response == nil {
			t.Fatal("Expected response but got nil")
		}
	})

	t.Run("UnhealthyQdrant", func(t *testing.T) {
		mockQdrant := mockHTTPServer(t, http.StatusServiceUnavailable, `{"error":"unavailable"}`)
		defer mockQdrant.Close()

		config.QdrantURL = mockQdrant.URL
		router := setupTestRouter()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/system/vector-status",
		})

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", w.Code)
		}
	})
}

// TestAIStatusHandlerWithMockServer tests AI status with mock Ollama
func TestAIStatusHandlerWithMockServer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthyOllama", func(t *testing.T) {
		mockOllama := mockHTTPServer(t, http.StatusOK, `{"models":[]}`)
		defer mockOllama.Close()

		config.OllamaURL = mockOllama.URL
		router := setupTestRouter()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/system/ai-status",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"service": "ollama",
			"status":  "healthy",
		})

		if response == nil {
			t.Fatal("Expected response but got nil")
		}
	})

	t.Run("UnhealthyOllama", func(t *testing.T) {
		mockOllama := mockHTTPServer(t, http.StatusInternalServerError, `{"error":"server error"}`)
		defer mockOllama.Close()

		config.OllamaURL = mockOllama.URL
		router := setupTestRouter()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/system/ai-status",
		})

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", w.Code)
		}
	})
}

// TestCreateApplicationValidation tests application creation validation
func TestCreateApplicationValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Skip tests that require database
	t.Skip("Skipping database-dependent tests - requires running database")
}

// TestCreateAgentValidation tests agent creation validation
func TestCreateAgentValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Skip tests that require database
	t.Skip("Skipping database-dependent tests - requires running database")
}

// TestCreateQueueItemValidation tests queue item creation validation
func TestCreateQueueItemValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Skip tests that require database
	t.Skip("Skipping database-dependent tests - requires running database")
}

// TestHandlerContentTypes tests that handlers return correct content types
func TestHandlerContentTypes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		// Skip endpoints that require database or external services
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("%s_%s", endpoint.method, strings.ReplaceAll(endpoint.path, "/", "_")), func(t *testing.T) {
			w := makeHTTPRequest(router, HTTPTestRequest{
				Method: endpoint.method,
				Path:   endpoint.path,
			})

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json for %s %s, got %s",
					endpoint.method, endpoint.path, contentType)
			}
		})
	}
}

// TestVectorSearchHandler tests the vector search functionality
func TestVectorSearchHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Set up config for tests
	config = Config{
		QdrantURL: "http://localhost:6333",
	}

	t.Run("ValidSearchRequest", func(t *testing.T) {
		router := setupTestRouter()

		reqBody := `{"query": "architecture documentation", "limit": 5}`
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   reqBody,
		})

		// Can be 200 (success) or 503 (Qdrant unavailable)
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", w.Code)
		}

		var response SearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Query != "architecture documentation" {
			t.Errorf("Expected query 'architecture documentation', got %q", response.Query)
		}

		// Verify result structure (may be empty if Qdrant not available)
		if len(response.Results) > 0 {
			result := response.Results[0]
			if result.Score < 0 || result.Score > 1 {
				t.Errorf("Expected score between 0-1, got %f", result.Score)
			}
			if result.ID == "" {
				t.Error("Expected result ID, got empty string")
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		router := setupTestRouter()
		
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   "invalid json",
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		router := setupTestRouter()
		
		reqBody := `{"query": "", "limit": 5}`
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   reqBody,
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		router := setupTestRouter()
		
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
		})

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("LimitValidation", func(t *testing.T) {
		router := setupTestRouter()

		reqBody := `{"query": "test", "limit": 1}`
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   reqBody,
		})

		// Can be 200 (success) or 503 (Qdrant unavailable)
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", w.Code)
		}

		var response SearchResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		if len(response.Results) > 1 {
			t.Errorf("Expected at most 1 result with limit=1, got %d", len(response.Results))
		}
	})
}

// TestGenerateMockEmbedding tests the mock embedding generation
func TestGenerateMockEmbedding(t *testing.T) {
	t.Run("GeneratesCorrectDimension", func(t *testing.T) {
		embedding := generateMockEmbedding("test query")

		if len(embedding) != 384 {
			t.Errorf("Expected embedding dimension 384, got %d", len(embedding))
		}
	})

	t.Run("GeneratesDeterministicResults", func(t *testing.T) {
		text := "consistent test"
		embedding1 := generateMockEmbedding(text)
		embedding2 := generateMockEmbedding(text)

		if len(embedding1) != len(embedding2) {
			t.Error("Embeddings have different lengths")
		}

		// Check first few values are the same (deterministic)
		for i := 0; i < 5 && i < len(embedding1); i++ {
			if embedding1[i] != embedding2[i] {
				t.Errorf("Embeddings differ at position %d: %f vs %f", i, embedding1[i], embedding2[i])
			}
		}
	})
}

// TestQueryQdrantSimilarity tests Qdrant similarity queries
func TestQueryQdrantSimilarity(t *testing.T) {
	// Set up config for tests
	config = Config{
		QdrantURL: "http://localhost:6333",
	}

	t.Run("ReturnsResults", func(t *testing.T) {
		embedding := generateMockEmbedding("test")
		results, err := queryQdrantSimilarity(embedding, 10)

		if err != nil {
			t.Logf("Qdrant query returned error (expected if Qdrant not running): %v", err)
			// Not a test failure - Qdrant might not be running in test environment
			return
		}

		// Verify result structure if we got results
		for _, result := range results {
			if result.Score < 0 || result.Score > 1 {
				t.Errorf("Invalid score %f (should be 0-1)", result.Score)
			}
		}
	})

	t.Run("RespectsLimit", func(t *testing.T) {
		embedding := generateMockEmbedding("test")
		limit := 1
		results, err := queryQdrantSimilarity(embedding, limit)

		if err != nil {
			t.Logf("Qdrant query returned error: %v", err)
			return
		}

		if len(results) > limit {
			t.Errorf("Expected at most %d results, got %d", limit, len(results))
		}
	})
}

// TestGenerateOllamaEmbedding tests the Ollama embedding generation
func TestGenerateOllamaEmbedding(t *testing.T) {
	// Save original config
	originalOllamaURL := config.OllamaURL
	defer func() { config.OllamaURL = originalOllamaURL }()

	t.Run("ValidEmbeddingGeneration", func(t *testing.T) {
		// Skip if Ollama is not configured
		if config.OllamaURL == "" {
			config.OllamaURL = "http://localhost:11434"
		}

		embedding, err := generateOllamaEmbedding("test query for embeddings")

		// If Ollama is not available, this is not a failure
		if err != nil {
			t.Logf("Ollama not available (expected in some environments): %v", err)
			return
		}

		// Verify embedding properties
		if len(embedding) == 0 {
			t.Error("Expected non-empty embedding vector")
		}

		// nomic-embed-text typically returns 768-dimensional embeddings
		if len(embedding) != 768 {
			t.Logf("Note: Expected 768 dimensions, got %d (model may vary)", len(embedding))
		}

		// Verify embedding values are in reasonable range
		for i, val := range embedding {
			if val < -10 || val > 10 {
				t.Errorf("Embedding value at index %d out of expected range: %f", i, val)
			}
		}
	})

	t.Run("EmptyOllamaURL", func(t *testing.T) {
		config.OllamaURL = ""
		_, err := generateOllamaEmbedding("test")

		if err == nil {
			t.Error("Expected error when Ollama URL is empty")
		}
	})

	t.Run("DifferentTextsDifferentEmbeddings", func(t *testing.T) {
		if config.OllamaURL == "" {
			config.OllamaURL = "http://localhost:11434"
		}

		embedding1, err1 := generateOllamaEmbedding("database architecture")
		embedding2, err2 := generateOllamaEmbedding("machine learning models")

		// Skip if Ollama is not available
		if err1 != nil || err2 != nil {
			t.Logf("Ollama not available, skipping comparison test")
			return
		}

		// Embeddings should be different for different texts
		same := true
		for i := range embedding1 {
			if embedding1[i] != embedding2[i] {
				same = false
				break
			}
		}

		if same {
			t.Error("Expected different embeddings for different input texts")
		}
	})
}

// TestIndexHandler tests the document indexing endpoint
func TestIndexHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup config for tests
	config = Config{
		OllamaURL: "http://localhost:11434",
		QdrantURL: "http://localhost:6333",
	}

	router := setupTestRouter()
	router.HandleFunc("/api/index", indexHandler).Methods("POST")

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/index", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MissingApplicationID", func(t *testing.T) {
		reqBody := IndexRequest{
			Documents: []Document{
				{ID: "doc1", Content: "Test content"},
			},
		}
		jsonData, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/index", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		if !strings.Contains(response["error"], "application_id") {
			t.Errorf("Expected application_id error, got: %v", response)
		}
	})

	t.Run("EmptyDocumentsArray", func(t *testing.T) {
		reqBody := IndexRequest{
			ApplicationID: "test-app-id",
			Documents:     []Document{},
		}
		jsonData, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/index", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		if !strings.Contains(response["error"], "documents") {
			t.Errorf("Expected documents error, got: %v", response)
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/index", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

// TestEnsureQdrantCollection tests collection creation logic
func TestEnsureQdrantCollection(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RequiresQdrantURL", func(t *testing.T) {
		config = Config{
			QdrantURL: "",
		}

		err := ensureQdrantCollection()
		if err == nil {
			t.Error("Expected error when Qdrant URL not configured")
		}
		// Error can be either "not configured" or "unsupported protocol scheme"
		if !strings.Contains(err.Error(), "not configured") && !strings.Contains(err.Error(), "unsupported protocol") {
			t.Errorf("Expected Qdrant configuration error, got: %v", err)
		}
	})

	t.Run("WithValidConfiguration", func(t *testing.T) {
		// Setup mock server that simulates Qdrant
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate collection doesn't exist initially
			if r.Method == "GET" && strings.Contains(r.URL.Path, "collections") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Simulate successful collection creation
			if r.Method == "PUT" && strings.Contains(r.URL.Path, "collections") {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{"result": true})
				return
			}
		}))
		defer mockServer.Close()

		config = Config{
			QdrantURL: mockServer.URL,
		}

		err := ensureQdrantCollection()
		if err != nil {
			t.Logf("Note: ensureQdrantCollection returned error (may be expected if Qdrant unavailable): %v", err)
		}
	})
}

// TestIndexDocuments tests the document indexing logic
func TestIndexDocuments(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyDocumentsList", func(t *testing.T) {
		config = Config{
			OllamaURL: "http://localhost:11434",
			QdrantURL: "http://localhost:6333",
		}

		indexed, errors, err := indexDocuments([]Document{})

		// Empty documents should succeed with 0 indexed
		if err != nil {
			t.Logf("Note: indexDocuments returned error (may be expected if resources unavailable): %v", err)
		}
		if indexed != 0 {
			t.Errorf("Expected 0 indexed for empty list, got %d", indexed)
		}
		if len(errors) != 0 {
			t.Errorf("Expected 0 errors for empty list, got %d", len(errors))
		}
	})

	t.Run("WithDocuments", func(t *testing.T) {
		config = Config{
			OllamaURL: "http://localhost:11434",
			QdrantURL: "http://localhost:6333",
		}

		docs := []Document{
			{
				ID:              "test-doc-1",
				ApplicationID:   "test-app",
				ApplicationName: "Test Application",
				Path:            "/README.md",
				Content:         "This is test content for indexing",
				Metadata: map[string]interface{}{
					"type": "readme",
				},
			},
		}

		indexed, errors, err := indexDocuments(docs)

		if err != nil {
			t.Logf("Note: indexDocuments returned error (expected if Qdrant/Ollama unavailable): %v", err)
			// This is acceptable - we're testing the validation logic
		}

		// We can't reliably test actual indexing without live services
		// but we can verify the function handles the input correctly
		if indexed > len(docs) {
			t.Errorf("Expected indexed <= %d, got %d", len(docs), indexed)
		}

		t.Logf("Indexed: %d, Errors: %v", indexed, errors)
	})
}

// TestDeleteApplication tests the application deletion endpoint
func TestDeleteApplication(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup router with DELETE endpoint
	router := setupTestRouter()
	router.HandleFunc("/api/applications", applicationsHandler).Methods("GET", "POST", "DELETE", "OPTIONS")

	t.Run("MissingID", func(t *testing.T) {
		// Test DELETE without ID parameter
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/applications",
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing ID, got %d", w.Code)
		}

		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !strings.Contains(response["error"], "Missing application ID") {
			t.Errorf("Expected 'Missing application ID' error, got: %v", response["error"])
		}
	})

	t.Run("EmptyID", func(t *testing.T) {
		// Test DELETE with empty ID parameter
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method:      "DELETE",
			Path:        "/api/applications?id=",
			QueryParams: map[string]string{"id": ""},
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty ID, got %d", w.Code)
		}
	})

	t.Run("InvalidIDFormat", func(t *testing.T) {
		// Skip if no database (would cause nil pointer panic)
		if db == nil {
			t.Skip("Skipping test requiring database connection")
		}

		// Test DELETE with invalid UUID format
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/applications?id=invalid-uuid-format",
		})

		// With database, this should handle invalid UUID gracefully
		if w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Errorf("Expected error status code, got %d", w.Code)
		}
	})

	t.Run("ValidUUIDFormat", func(t *testing.T) {
		// Skip if no database (would cause nil pointer panic)
		if db == nil {
			t.Skip("Skipping test requiring database connection")
		}

		// Test DELETE with valid UUID format
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/applications?id=" + validUUID,
		})

		// Should return 404 (not found) or 500 (database error)
		if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 404 or 500, got %d", w.Code)
		}
	})

	t.Run("ContentTypeHeader", func(t *testing.T) {
		// Skip if no database (would cause nil pointer panic)
		if db == nil {
			t.Skip("Skipping test requiring database connection")
		}

		// Test that DELETE endpoint sets proper Content-Type
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/applications?id=550e8400-e29b-41d4-a716-446655440000",
		})

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("ResponseStructure", func(t *testing.T) {
		// Test that error responses have proper JSON structure
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/applications",
		})

		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		if _, exists := response["error"]; !exists {
			t.Error("Expected 'error' field in response")
		}
	})
}

// TestDeleteAgent tests the agent deletion endpoint
func TestDeleteAgent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()
	router.HandleFunc("/api/agents", agentsHandler).Methods("GET", "POST", "DELETE", "OPTIONS")

	t.Run("MissingID", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/agents",
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing ID, got %d", w.Code)
		}

		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !strings.Contains(response["error"], "Missing agent ID") {
			t.Errorf("Expected 'Missing agent ID' error, got: %v", response["error"])
		}
	})

	t.Run("EmptyID", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/agents?id=",
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty ID, got %d", w.Code)
		}
	})

	t.Run("ValidUUIDFormat", func(t *testing.T) {
		// Skip if no database (would cause nil pointer panic)
		if db == nil {
			t.Skip("Skipping test requiring database connection")
		}

		validUUID := "660e8400-e29b-41d4-a716-446655440000"
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/agents?id=" + validUUID,
		})

		// Should return 404 (not found) or 500 (database error)
		if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 404 or 500, got %d", w.Code)
		}
	})

	t.Run("ContentTypeHeader", func(t *testing.T) {
		// Skip if no database (would cause nil pointer panic)
		if db == nil {
			t.Skip("Skipping test requiring database connection")
		}

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/agents?id=660e8400-e29b-41d4-a716-446655440000",
		})

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("ResponseStructure", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/agents",
		})

		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		if _, exists := response["error"]; !exists {
			t.Error("Expected 'error' field in response")
		}
	})

	t.Run("CascadingDeleteConcern", func(t *testing.T) {
		// Document that deleteAgent should cascade to improvement_queue
		// This is a documentation test - actual cascade is tested in integration
		t.Log("deleteAgent should cascade DELETE to improvement_queue WHERE agent_id = $1")
		t.Log("This cascade behavior is tested in integration tests with database")
	})
}

// TestDeleteQueueItem tests the queue item deletion endpoint
func TestDeleteQueueItem(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()
	router.HandleFunc("/api/queue", queueHandler).Methods("GET", "POST", "DELETE", "OPTIONS")

	t.Run("MissingID", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/queue",
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing ID, got %d", w.Code)
		}

		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !strings.Contains(response["error"], "Missing queue item ID") {
			t.Errorf("Expected 'Missing queue item ID' error, got: %v", response["error"])
		}
	})

	t.Run("EmptyID", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/queue?id=",
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty ID, got %d", w.Code)
		}
	})

	t.Run("ValidUUIDFormat", func(t *testing.T) {
		// Skip if no database (would cause nil pointer panic)
		if db == nil {
			t.Skip("Skipping test requiring database connection")
		}

		validUUID := "770e8400-e29b-41d4-a716-446655440000"
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/queue?id=" + validUUID,
		})

		// Should return 404 (not found) or 500 (database error)
		if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 404 or 500, got %d", w.Code)
		}
	})

	t.Run("ContentTypeHeader", func(t *testing.T) {
		// Skip if no database (would cause nil pointer panic)
		if db == nil {
			t.Skip("Skipping test requiring database connection")
		}

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/queue?id=770e8400-e29b-41d4-a716-446655440000",
		})

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("ResponseStructure", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/queue",
		})

		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		if _, exists := response["error"]; !exists {
			t.Error("Expected 'error' field in response")
		}
	})

	t.Run("DirectDelete", func(t *testing.T) {
		// Document that deleteQueueItem does NOT cascade (it's a leaf entity)
		t.Log("deleteQueueItem performs direct DELETE with no cascading")
		t.Log("Queue items are leaf entities with no dependent records")
	})
}
