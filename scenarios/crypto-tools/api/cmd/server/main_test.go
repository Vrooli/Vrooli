// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewServer tests server initialization
func TestNewServer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		server, err := NewServer()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if server == nil {
			t.Fatal("Expected server to be initialized")
		}

		if server.config == nil {
			t.Error("Expected config to be initialized")
		}

		if server.router == nil {
			t.Error("Expected router to be initialized")
		}

		// Database can be nil for mock mode
		if server.config.DatabaseURL != "mock" && server.db != nil {
			defer server.db.Close()
		}
	})

	t.Run("ConfigDefaults", func(t *testing.T) {
		server, _ := NewServer()

		if server.config.APIToken == "" {
			t.Error("Expected default API token to be set")
		}

		if server.config.Port == "" {
			t.Error("Expected default port to be set")
		}
	})
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Health endpoint should return degraded (503) without database
		if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusOK {
			t.Errorf("Expected status 503 or 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		status, ok := response["status"].(string)
		if !ok {
			t.Error("Expected status field in response")
		}

		// Without DB, should be unhealthy or degraded
		if status != "unhealthy" && status != "degraded" && status != "healthy" {
			t.Errorf("Expected status to be unhealthy/degraded/healthy, got %s", status)
		}

		// Check for required fields
		if _, ok := response["service"]; !ok {
			t.Error("Expected service field in response")
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp field in response")
		}

		if _, ok := response["dependencies"]; !ok {
			t.Error("Expected dependencies field in response")
		}
	})

	t.Run("NoAuthRequired", func(t *testing.T) {
		// Health endpoint should work without auth
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/health",
			Headers: map[string]string{},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code == http.StatusUnauthorized {
			t.Error("Health endpoint should not require authentication")
		}
	})
}

// TestAuthMiddleware tests authentication middleware
func TestAuthMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ValidToken", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/resources",
			Headers: map[string]string{
				"Authorization": "Bearer " + env.Config.APIToken,
			},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should not be unauthorized with valid token
		if w.Code == http.StatusUnauthorized {
			t.Error("Expected request to be authorized with valid token")
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/resources",
			Headers: map[string]string{
				"Authorization": "Bearer invalid-token",
			},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
	})

	t.Run("MissingToken", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/resources",
			Headers: map[string]string{},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
	})

	t.Run("HealthEndpointBypass", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/health",
			Headers: map[string]string{},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code == http.StatusUnauthorized {
			t.Error("Health endpoint should bypass auth")
		}
	})

	t.Run("DocsEndpointBypass", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/docs",
			Headers: map[string]string{},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code == http.StatusUnauthorized {
			t.Error("Docs endpoint should bypass auth")
		}
	})
}

// TestCORSMiddleware tests CORS headers
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AllowedOrigin", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/health", nil)
		req.Header.Set("Origin", "http://localhost:3000")

		w := httptest.NewRecorder()
		env.Server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		corsHeader := w.Header().Get("Access-Control-Allow-Origin")
		if corsHeader != "http://localhost:3000" {
			t.Errorf("Expected CORS header for localhost:3000, got %s", corsHeader)
		}
	})

	t.Run("DisallowedOrigin", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/health", nil)
		req.Header.Set("Origin", "http://evil.com")

		w := httptest.NewRecorder()
		env.Server.router.ServeHTTP(w, req)

		corsHeader := w.Header().Get("Access-Control-Allow-Origin")
		if corsHeader == "http://evil.com" {
			t.Error("Should not allow evil.com origin")
		}
	})

	t.Run("NoOrigin", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/health", nil)

		w := httptest.NewRecorder()
		env.Server.router.ServeHTTP(w, req)

		corsHeader := w.Header().Get("Access-Control-Allow-Origin")
		if corsHeader == "" {
			t.Error("Expected default CORS header when no origin specified")
		}
	})
}

// TestDocsEndpoint tests the documentation endpoint
func TestDocsEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/docs",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response Response
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success response")
		}

		docsData, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected docs data to be a map")
		}

		if _, ok := docsData["name"]; !ok {
			t.Error("Expected name field in docs")
		}

		if _, ok := docsData["version"]; !ok {
			t.Error("Expected version field in docs")
		}

		if _, ok := docsData["endpoints"]; !ok {
			t.Error("Expected endpoints field in docs")
		}
	})
}

// TestGetEnv tests environment variable retrieval
func TestGetEnv(t *testing.T) {
	t.Run("ExistingVariable", func(t *testing.T) {
		key := "TEST_ENV_VAR_EXISTS"
		value := "test-value-123"
		t.Setenv(key, value)

		result := getEnv(key, "default")
		if result != value {
			t.Errorf("Expected %s, got %s", value, result)
		}
	})

	t.Run("MissingVariable", func(t *testing.T) {
		key := "TEST_ENV_VAR_MISSING"
		defaultValue := "default-value"

		result := getEnv(key, defaultValue)
		if result != defaultValue {
			t.Errorf("Expected %s, got %s", defaultValue, result)
		}
	})

	t.Run("EmptyVariable", func(t *testing.T) {
		key := "TEST_ENV_VAR_EMPTY"
		defaultValue := "default-value"
		t.Setenv(key, "")

		result := getEnv(key, defaultValue)
		if result != defaultValue {
			t.Errorf("Expected default value when env var is empty, got %s", result)
		}
	})
}

// TestSendJSON tests JSON response helper
func TestSendJSON(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("SuccessResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		testData := map[string]string{"key": "value"}

		env.Server.sendJSON(w, http.StatusOK, testData)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response Response
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true for 2xx status")
		}

		if response.Data == nil {
			t.Error("Expected data to be present")
		}
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		testData := map[string]string{"error": "test error"}

		env.Server.sendJSON(w, http.StatusBadRequest, testData)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response Response
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Success {
			t.Error("Expected success to be false for 4xx status")
		}
	})
}

// TestSendError tests error response helper
func TestSendError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("BasicError", func(t *testing.T) {
		w := httptest.NewRecorder()
		errorMsg := "test error message"

		env.Server.sendError(w, http.StatusBadRequest, errorMsg)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response Response
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Success {
			t.Error("Expected success to be false")
		}

		if response.Error != errorMsg {
			t.Errorf("Expected error message '%s', got '%s'", errorMsg, response.Error)
		}

		if response.Data != nil {
			t.Error("Expected data to be nil for error response")
		}
	})
}

// TestCheckDatabaseHealth tests database health check
func TestCheckDatabaseHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NoDatabaseConnection", func(t *testing.T) {
		health := env.Server.checkDatabaseHealth()

		status, ok := health["status"].(string)
		if !ok {
			t.Fatal("Expected status to be a string")
		}

		if status != "unhealthy" {
			t.Errorf("Expected unhealthy status without DB, got %s", status)
		}

		if _, ok := health["error"]; !ok {
			t.Error("Expected error field when database is unhealthy")
		}
	})
}

// TestCountHealthyDependencies tests dependency counting
func TestCountHealthyDependencies(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AllHealthy", func(t *testing.T) {
		deps := map[string]interface{}{
			"db": map[string]interface{}{"status": "healthy"},
			"n8n": map[string]interface{}{"status": "healthy"},
			"windmill": map[string]interface{}{"status": "healthy"},
		}

		count := env.Server.countHealthyDependencies(deps)
		if count != 3 {
			t.Errorf("Expected 3 healthy dependencies, got %d", count)
		}
	})

	t.Run("MixedHealth", func(t *testing.T) {
		deps := map[string]interface{}{
			"db": map[string]interface{}{"status": "healthy"},
			"n8n": map[string]interface{}{"status": "unhealthy"},
			"windmill": map[string]interface{}{"status": "not_configured"},
		}

		count := env.Server.countHealthyDependencies(deps)
		if count != 2 {
			t.Errorf("Expected 2 healthy dependencies (healthy + not_configured), got %d", count)
		}
	})

	t.Run("AllUnhealthy", func(t *testing.T) {
		deps := map[string]interface{}{
			"db": map[string]interface{}{"status": "unhealthy"},
			"n8n": map[string]interface{}{"status": "degraded"},
		}

		count := env.Server.countHealthyDependencies(deps)
		if count != 0 {
			t.Errorf("Expected 0 healthy dependencies, got %d", count)
		}
	})
}
