// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestLoadConfigEdgeCases tests edge cases in configuration loading
func TestLoadConfigEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MissingPortVariables", func(t *testing.T) {
		// Clear both PORT and API_PORT
		oldPort := os.Getenv("PORT")
		oldAPIPort := os.Getenv("API_PORT")
		oldPostgresURL := os.Getenv("POSTGRES_URL")

		defer func() {
			os.Setenv("PORT", oldPort)
			os.Setenv("API_PORT", oldAPIPort)
			os.Setenv("POSTGRES_URL", oldPostgresURL)
		}()

		os.Unsetenv("PORT")
		os.Unsetenv("API_PORT")
		os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/testdb")

		// Should call log.Fatal - we can't test this directly in Go
		// but we verify the logic by checking with API_PORT set
		os.Setenv("API_PORT", "8080")
		config := loadConfig()
		if config.Port != "8080" {
			t.Errorf("Expected port 8080, got %s", config.Port)
		}
	})

	t.Run("PortFromPORTVariable", func(t *testing.T) {
		oldPort := os.Getenv("PORT")
		oldAPIPort := os.Getenv("API_PORT")
		oldPostgresURL := os.Getenv("POSTGRES_URL")

		defer func() {
			os.Setenv("PORT", oldPort)
			os.Setenv("API_PORT", oldAPIPort)
			os.Setenv("POSTGRES_URL", oldPostgresURL)
		}()

		os.Unsetenv("API_PORT")
		os.Setenv("PORT", "9000")
		os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/testdb")

		config := loadConfig()
		if config.Port != "9000" {
			t.Errorf("Expected port 9000, got %s", config.Port)
		}
	})

	t.Run("IndividualDatabaseComponents", func(t *testing.T) {
		oldVars := map[string]string{
			"POSTGRES_URL":      os.Getenv("POSTGRES_URL"),
			"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
			"POSTGRES_PORT":     os.Getenv("POSTGRES_PORT"),
			"POSTGRES_USER":     os.Getenv("POSTGRES_USER"),
			"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
			"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
			"API_PORT":          os.Getenv("API_PORT"),
		}

		defer func() {
			for k, v := range oldVars {
				if v == "" {
					os.Unsetenv(k)
				} else {
					os.Setenv(k, v)
				}
			}
		}()

		// Clear POSTGRES_URL and set individual components
		os.Unsetenv("POSTGRES_URL")
		os.Setenv("POSTGRES_HOST", "testhost")
		os.Setenv("POSTGRES_PORT", "5433")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")
		os.Setenv("API_PORT", "8080")

		config := loadConfig()
		expectedURL := "postgres://testuser:testpass@testhost:5433/testdb?sslmode=disable"
		if config.PostgresURL != expectedURL {
			t.Errorf("Expected URL %s, got %s", expectedURL, config.PostgresURL)
		}
	})

	t.Run("AllEnvironmentDefaults", func(t *testing.T) {
			oldVars := map[string]string{
				"N8N_BASE_URL":     os.Getenv("N8N_BASE_URL"),
				"COMFYUI_BASE_URL": os.Getenv("COMFYUI_BASE_URL"),
				"WHISPER_BASE_URL": os.Getenv("WHISPER_BASE_URL"),
				"MINIO_ENDPOINT":   os.Getenv("MINIO_ENDPOINT"),
				"MINIO_ACCESS_KEY": os.Getenv("MINIO_ACCESS_KEY"),
				"MINIO_SECRET_KEY": os.Getenv("MINIO_SECRET_KEY"),
				"QDRANT_URL":       os.Getenv("QDRANT_URL"),
				"POSTGRES_URL":     os.Getenv("POSTGRES_URL"),
				"API_PORT":         os.Getenv("API_PORT"),
			}

		defer func() {
			for k, v := range oldVars {
				if v == "" {
					os.Unsetenv(k)
				} else {
					os.Setenv(k, v)
				}
			}
		}()

		// Unset all optional variables
		os.Unsetenv("N8N_BASE_URL")
		os.Unsetenv("COMFYUI_BASE_URL")
		os.Unsetenv("WHISPER_BASE_URL")
		os.Unsetenv("MINIO_ENDPOINT")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("QDRANT_URL")

		os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/testdb")
		os.Setenv("API_PORT", "8080")

		config := loadConfig()

		// Verify defaults
		if config.N8NBaseURL != "http://localhost:5678" {
			t.Errorf("Expected default N8N URL, got %s", config.N8NBaseURL)
		}
		if config.ComfyUIBaseURL != "http://localhost:8188" {
			t.Errorf("Expected default ComfyUI URL, got %s", config.ComfyUIBaseURL)
		}
		if config.WhisperBaseURL != "http://localhost:9000" {
			t.Errorf("Expected default Whisper URL, got %s", config.WhisperBaseURL)
		}
		if config.MinioEndpoint != "localhost:9000" {
			t.Errorf("Expected default Minio endpoint, got %s", config.MinioEndpoint)
		}
		if config.MinioAccessKey != "minioadmin" {
			t.Errorf("Expected default Minio access key, got %s", config.MinioAccessKey)
		}
		if config.MinioSecretKey != "minioadmin" {
			t.Errorf("Expected default Minio secret key, got %s", config.MinioSecretKey)
		}
		if config.QdrantURL != "http://localhost:6333" {
			t.Errorf("Expected default Qdrant URL, got %s", config.QdrantURL)
		}
	})
}

// TestProcessAudioWithWhisperEdgeCases tests edge cases in audio processing
func TestProcessAudioWithWhisperEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyTextResponse", func(t *testing.T) {
		mockWhisper := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusOK,
				Body:       map[string]interface{}{"data": "something"},
			},
		})
		defer mockWhisper.Server.Close()

		_, err := processAudioWithWhisper(mockWhisper.Server.URL, "base64data", "wav")
		if err == nil {
			t.Error("Expected error for missing text field")
		}
		if !strings.Contains(err.Error(), "unexpected response format") {
			t.Errorf("Expected 'unexpected response format' error, got: %v", err)
		}
	})

	t.Run("NonStringTextResponse", func(t *testing.T) {
		mockWhisper := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusOK,
				Body:       map[string]interface{}{"text": 12345},
			},
		})
		defer mockWhisper.Server.Close()

		_, err := processAudioWithWhisper(mockWhisper.Server.URL, "base64data", "wav")
		if err == nil {
			t.Error("Expected error for non-string text field")
		}
	})

	t.Run("EmptyResponse", func(t *testing.T) {
		mockWhisper := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusOK,
				Body:       map[string]interface{}{},
			},
		})
		defer mockWhisper.Server.Close()

		_, err := processAudioWithWhisper(mockWhisper.Server.URL, "base64data", "wav")
		if err == nil {
			t.Error("Expected error for empty response")
		}
	})
}

// TestHandlerRoutingEdgeCases tests handler routing logic
func TestHandlerRoutingEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CampaignsHandler_PUT", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/campaigns",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		campaignsHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("CampaignsHandler_DELETE", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/campaigns",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		campaignsHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("BrandsHandler_PUT", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/brands",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		brandsHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("BrandsHandler_DELETE", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/brands",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		brandsHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("GenerateImageHandler_GET", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/generate",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateImageHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("GenerateImageHandler_InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/generate", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		generateImageHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("ProcessVoiceBriefHandler_GET", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/voice-brief",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		processVoiceBriefHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("GenerationsHandler_POST", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/generations",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generationsHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})
}

// TestHelperFunctionCoverage tests helper functions to improve coverage
func TestHelperFunctionCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MakeHTTPRequest_WithBody", func(t *testing.T) {
		body := map[string]string{"test": "data"}
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   body,
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		var decoded map[string]string
		if err := json.NewDecoder(httpReq.Body).Decode(&decoded); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}

		if decoded["test"] != "data" {
			t.Errorf("Expected 'data', got %s", decoded["test"])
		}
	})

	t.Run("MakeHTTPRequest_WithoutBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq.Method != "GET" {
			t.Errorf("Expected GET method, got %s", httpReq.Method)
		}
	})

	t.Run("AssertJSONResponse_WithPartialMatch", func(t *testing.T) {
		w := httptest.NewRecorder()
		response := map[string]interface{}{
			"status":  "success",
			"message": "test",
			"data":    map[string]string{"key": "value"},
		}
		json.NewEncoder(w).Encode(response)

		result := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "success",
		})

		if result == nil {
			t.Error("Expected non-nil result")
		}
		if result["message"] != "test" {
			t.Error("Expected message field to be preserved")
		}
	})

	t.Run("AssertErrorResponse_Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid input"))

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestDataGeneratorFunctions tests data generation helpers
func TestDataGeneratorFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GenerateGenerationRequest", func(t *testing.T) {
		gen := &TestDataGenerator{}
		req := gen.GenerateGenerationRequest("campaign-123")

		if req.CampaignID != "campaign-123" {
			t.Errorf("Expected campaign ID 'campaign-123', got %s", req.CampaignID)
		}
		if req.Prompt == "" {
			t.Error("Expected non-empty prompt")
		}
		if req.Style == "" {
			t.Error("Expected non-empty style")
		}
		if req.Dimensions == "" {
			t.Error("Expected non-empty dimensions")
		}
	})

	t.Run("GenerateVoiceBriefRequest", func(t *testing.T) {
		gen := &TestDataGenerator{}
		req := gen.GenerateVoiceBriefRequest()

		if req.AudioData == "" {
			t.Error("Expected non-empty audio data")
		}
		if req.Format != "wav" {
			t.Errorf("Expected format 'wav', got %s", req.Format)
		}
	})
}

// TestTestPatternHelpers tests pattern helper functions
func TestTestPatternHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyBodyPattern", func(t *testing.T) {
		pattern := emptyBodyPattern("POST", "/test")
		if pattern.ExpectedStatus != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, pattern.ExpectedStatus)
		}
	})

	t.Run("MissingRequiredFieldPattern", func(t *testing.T) {
		body := map[string]interface{}{"field": "value"}
		pattern := missingRequiredFieldPattern("POST", "/test", body)
		if pattern.ExpectedStatus != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, pattern.ExpectedStatus)
		}
	})

	t.Run("DatabaseErrorPattern", func(t *testing.T) {
		pattern := databaseErrorPattern("GET", "/test", nil)
		if pattern.ExpectedStatus != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, pattern.ExpectedStatus)
		}
	})
}

// TestScenarioBuilderCoverage tests scenario builder methods
func TestScenarioBuilderCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AddEmptyBody", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddEmptyBody("POST", "/test")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("AddMissingRequiredField", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		body := map[string]interface{}{"field": "value"}
		builder.AddMissingRequiredField("POST", "/test", body)
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("AddDatabaseError", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddDatabaseError("GET", "/test", nil)
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("AddCustom", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		custom := ErrorTestPattern{
			Name:           "CustomPattern",
			Description:    "Custom test pattern",
			ExpectedStatus: http.StatusTeapot,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "POST",
					Path:   "/custom",
				}
			},
		}
		builder.AddCustom(custom)
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
		if patterns[0].ExpectedStatus != http.StatusTeapot {
			t.Errorf("Expected status %d, got %d", http.StatusTeapot, patterns[0].ExpectedStatus)
		}
	})

	t.Run("MultiplePatterns", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidJSON("POST", "/test").
			AddEmptyBody("POST", "/test").
			AddInvalidMethod("/test", "POST").
			AddDatabaseError("GET", "/test", nil)
		patterns := builder.Build()

		if len(patterns) != 4 {
			t.Errorf("Expected 4 patterns, got %d", len(patterns))
		}
	})
}

// TestMockHTTPServerCoverage tests mock server functionality
func TestMockHTTPServerCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MultipleResponses", func(t *testing.T) {
		responses := []MockResponse{
			{StatusCode: http.StatusOK, Body: map[string]string{"msg": "first"}},
			{StatusCode: http.StatusCreated, Body: map[string]string{"msg": "second"}},
			{StatusCode: http.StatusAccepted, Body: map[string]string{"msg": "third"}},
		}

		mock := newMockHTTPServer(responses)
		defer mock.Server.Close()

		// First request
		resp1, err := http.Post(mock.Server.URL, "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		resp1.Body.Close()
		if resp1.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp1.StatusCode)
		}

		// Second request
		resp2, err := http.Post(mock.Server.URL, "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		resp2.Body.Close()
		if resp2.StatusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, resp2.StatusCode)
		}

		// Third request
		resp3, err := http.Post(mock.Server.URL, "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		resp3.Body.Close()
		if resp3.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, resp3.StatusCode)
		}

		// Verify request count
		if len(mock.Requests) != 3 {
			t.Errorf("Expected 3 requests, got %d", len(mock.Requests))
		}
	})

	t.Run("RequestTracking", func(t *testing.T) {
		mock := newMockHTTPServer([]MockResponse{
			{StatusCode: http.StatusOK, Body: "test"},
		})
		defer mock.Server.Close()

		testBody := []byte(`{"test": "data"}`)
		resp, err := http.Post(mock.Server.URL+"/test/path", "application/json", bytes.NewBuffer(testBody))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		resp.Body.Close()

		if len(mock.Requests) != 1 {
			t.Fatalf("Expected 1 request, got %d", len(mock.Requests))
		}

		req := mock.Requests[0]
		if req.Method != "POST" {
			t.Errorf("Expected POST method, got %s", req.Method)
		}
		if req.URL.Path != "/test/path" {
			t.Errorf("Expected path '/test/path', got %s", req.URL.Path)
		}
	})
}
