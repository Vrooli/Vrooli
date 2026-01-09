// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealth tests the health endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		Health(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response != nil {
			if service, ok := response["service"].(string); !ok || service != serviceName {
				t.Errorf("Expected service name %s, got %v", serviceName, service)
			}
			if version, ok := response["version"].(string); !ok || version != apiVersion {
				t.Errorf("Expected version %s, got %v", apiVersion, version)
			}
		}
	})
}

// TestListTranscriptions tests the list transcriptions endpoint
func TestListTranscriptions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyList", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/transcriptions", nil)
		w := httptest.NewRecorder()

		env.Service.ListTranscriptions(w, req)

		// Should return empty array
		response := assertJSONArray(t, w, http.StatusOK)
		if response == nil {
			t.Fatal("Expected array response")
		}
	})

	t.Run("Success_WithTranscriptions", func(t *testing.T) {
		// Create test transcriptions
		trans1 := setupTestTranscription(t, env.DB, "test1.mp3")
		defer trans1.Cleanup()
		trans2 := setupTestTranscription(t, env.DB, "test2.wav")
		defer trans2.Cleanup()

		req := httptest.NewRequest("GET", "/api/transcriptions", nil)
		w := httptest.NewRecorder()

		env.Service.ListTranscriptions(w, req)

		response := assertJSONArray(t, w, http.StatusOK)
		if response != nil && len(response) < 2 {
			t.Errorf("Expected at least 2 transcriptions, got %d", len(response))
		}
	})
}

// TestGetTranscription tests the get transcription endpoint
func TestGetTranscription(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		trans := setupTestTranscription(t, env.DB, "test.mp3")
		defer trans.Cleanup()

		reqData := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/transcriptions/" + trans.Transcription.ID.String(),
			URLVars: map[string]string{"id": trans.Transcription.ID.String()},
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.GetTranscription(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"filename": trans.Transcription.Filename,
		})

		if response != nil {
			if id, ok := response["id"].(string); !ok || id != trans.Transcription.ID.String() {
				t.Errorf("Expected ID %s, got %v", trans.Transcription.ID.String(), id)
			}
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "GetTranscription",
			Handler:     env.Service.GetTranscription,
			BaseURL:     "/api/transcriptions/{id}",
		}

		patterns := NewTestScenarioBuilder().
			AddNonExistentTranscription("GET", "/api/transcriptions/{id}").
			AddInvalidUUID("GET", "/api/transcriptions/invalid-uuid").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestUploadAudio tests the upload audio endpoint
func TestUploadAudio(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create a test audio file content
		audioContent := []byte("fake audio data for testing")

		req, err := createMultipartRequest(t, "/api/upload", "test.mp3", audioContent)
		if err != nil {
			t.Fatalf("Failed to create multipart request: %v", err)
		}

		w := httptest.NewRecorder()
		env.Service.UploadAudio(w, req)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"status":   "uploaded",
			"filename": "test.mp3",
		})

		if response != nil {
			if _, ok := response["transcription_id"].(string); !ok {
				t.Error("Expected transcription_id in response")
			}
		}
	})

	t.Run("UnsupportedFileType", func(t *testing.T) {
		audioContent := []byte("fake audio data")

		req, err := createMultipartRequest(t, "/api/upload", "test.txt", audioContent)
		if err != nil {
			t.Fatalf("Failed to create multipart request: %v", err)
		}

		w := httptest.NewRecorder()
		env.Service.UploadAudio(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Unsupported file type")
	})

	t.Run("MissingFile", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/upload", bytes.NewReader([]byte{}))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test")

		w := httptest.NewRecorder()
		env.Service.UploadAudio(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestAnalyzeTranscription tests the analyze transcription endpoint
func TestAnalyzeTranscription(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_SummaryAnalysis", func(t *testing.T) {
		trans := setupTestTranscription(t, env.DB, "test.mp3")
		defer trans.Cleanup()

		reqData := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/transcriptions/" + trans.Transcription.ID.String() + "/analyze",
			URLVars: map[string]string{"id": trans.Transcription.ID.String()},
			Body: TestData.AnalyzeRequest("summary", "", ""),
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.AnalyzeTranscription(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"analysis_type": "summary",
		})

		if response != nil {
			if _, ok := response["analysis_id"].(string); !ok {
				t.Error("Expected analysis_id in response")
			}
			if _, ok := response["result"].(string); !ok {
				t.Error("Expected result in response")
			}
		}
	})

	t.Run("Success_CustomPrompt", func(t *testing.T) {
		trans := setupTestTranscription(t, env.DB, "test.mp3")
		defer trans.Cleanup()

		reqData := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/transcriptions/" + trans.Transcription.ID.String() + "/analyze",
			URLVars: map[string]string{"id": trans.Transcription.ID.String()},
			Body: TestData.AnalyzeRequest("custom", "Analyze sentiment", "llama3.1:8b"),
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.AnalyzeTranscription(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "AnalyzeTranscription",
			Handler:     env.Service.AnalyzeTranscription,
			BaseURL:     "/api/transcriptions/{id}/analyze",
		}

		patterns := NewTestScenarioBuilder().
			AddNonExistentTranscription("POST", "/api/transcriptions/{id}/analyze").
			AddInvalidJSON("POST", "/api/transcriptions/{id}/analyze").
			AddMissingRequiredField("POST", "/api/transcriptions/{id}/analyze", "analysis_type").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestSearchTranscriptions tests the search transcriptions endpoint
func TestSearchTranscriptions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		reqData := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   TestData.SearchRequest("test query", 10),
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.SearchTranscriptions(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}
	})

	t.Run("Success_DefaultLimit", func(t *testing.T) {
		reqData := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   TestData.SearchRequest("test query", 0),
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.SearchTranscriptions(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "SearchTranscriptions",
			Handler:     env.Service.SearchTranscriptions,
			BaseURL:     "/api/search",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/search").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestGetAnalyses tests the get analyses endpoint
func TestGetAnalyses(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyList", func(t *testing.T) {
		trans := setupTestTranscription(t, env.DB, "test.mp3")
		defer trans.Cleanup()

		reqData := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/transcriptions/" + trans.Transcription.ID.String() + "/analyses",
			URLVars: map[string]string{"id": trans.Transcription.ID.String()},
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.GetAnalyses(w, httpReq)

		response := assertJSONArray(t, w, http.StatusOK)
		if response == nil {
			t.Fatal("Expected array response")
		}
	})

	t.Run("Success_WithAnalyses", func(t *testing.T) {
		trans := setupTestTranscription(t, env.DB, "test.mp3")
		defer trans.Cleanup()

		// Create a test analysis
		analysisID := uuid.New()
		_, err := env.DB.Exec(`
			INSERT INTO ai_analyses (id, transcription_id, analysis_type, prompt_used, result_text, processing_time_ms)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			analysisID, trans.Transcription.ID, "summary", "Summarize this", "Test result", 1000)
		if err != nil {
			t.Fatalf("Failed to create test analysis: %v", err)
		}
		defer env.DB.Exec("DELETE FROM ai_analyses WHERE id = $1", analysisID)

		reqData := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/transcriptions/" + trans.Transcription.ID.String() + "/analyses",
			URLVars: map[string]string{"id": trans.Transcription.ID.String()},
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.GetAnalyses(w, httpReq)

		response := assertJSONArray(t, w, http.StatusOK)
		if response != nil && len(response) == 0 {
			t.Error("Expected at least one analysis in response")
		}
	})
}

// TestNewAudioService tests the audio service constructor
func TestNewAudioService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	service := NewAudioService(
		db,
		"http://n8n:5678",
		"http://whisper:8090",
		"http://ollama:11434",
		"minio:9000",
		"http://qdrant:6333",
	)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.db != db {
		t.Error("Expected db to be set")
	}

	if service.n8nBaseURL != "http://n8n:5678" {
		t.Errorf("Expected n8nBaseURL to be http://n8n:5678, got %s", service.n8nBaseURL)
	}

	if service.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}

	if service.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

// TestNewLogger tests the logger constructor
func TestNewLogger(t *testing.T) {
	logger := NewLogger()

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	if logger.Logger == nil {
		t.Error("Expected internal logger to be initialized")
	}
}

// TestLoggerMethods tests logger methods
func TestLoggerMethods(t *testing.T) {
	logger := NewLogger()

	// Test that methods don't panic
	t.Run("Info", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Info method panicked: %v", r)
			}
		}()
		logger.Info("test info message")
	})

	t.Run("Error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error method panicked: %v", r)
			}
		}()
		logger.Error("test error message", nil)
	})

	t.Run("Warn", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warn method panicked: %v", r)
			}
		}()
		logger.Warn("test warn message", nil)
	})
}

// TestHTTPError tests the HTTP error helper
func TestHTTPError(t *testing.T) {
	t.Run("StandardError", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTTPError(w, "test error", http.StatusBadRequest, nil)

		assertErrorResponse(t, w, http.StatusBadRequest, "test error")
	})

	t.Run("InternalServerError", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTTPError(w, "internal error", http.StatusInternalServerError, nil)

		assertErrorResponse(t, w, http.StatusInternalServerError, "internal error")
	})

	t.Run("ResponseStructure", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTTPError(w, "test", http.StatusBadRequest, nil)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if _, ok := response["error"]; !ok {
			t.Error("Expected error field in response")
		}

		if _, ok := response["status"]; !ok {
			t.Error("Expected status field in response")
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp field in response")
		}
	})
}

// TestConstants tests that constants are defined correctly
func TestConstants(t *testing.T) {
	if apiVersion == "" {
		t.Error("apiVersion should not be empty")
	}

	if serviceName == "" {
		t.Error("serviceName should not be empty")
	}

	if defaultPort == "" {
		t.Error("defaultPort should not be empty")
	}

	if httpTimeout == 0 {
		t.Error("httpTimeout should not be zero")
	}

	if maxFileSize == 0 {
		t.Error("maxFileSize should not be zero")
	}
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyQuerySearch", func(t *testing.T) {
		reqData := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   map[string]interface{}{"query": ""},
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.SearchTranscriptions(w, httpReq)

		// Should still return 200 even with empty query
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for empty query, got %d", w.Code)
		}
	})

	t.Run("VeryLargeLimit", func(t *testing.T) {
		reqData := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   TestData.SearchRequest("test", 10000),
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.SearchTranscriptions(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestConcurrentRequests tests concurrent access to handlers
func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	trans := setupTestTranscription(t, env.DB, "test.mp3")
	defer trans.Cleanup()

	t.Run("ConcurrentReads", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				defer func() { done <- true }()

				reqData := HTTPTestRequest{
					Method:  "GET",
					Path:    "/api/transcriptions/" + trans.Transcription.ID.String(),
					URLVars: map[string]string{"id": trans.Transcription.ID.String()},
				}

				w, httpReq, err := makeHTTPRequest(reqData)
				if err != nil {
					t.Errorf("Failed to create request: %v", err)
					return
				}

				env.Service.GetTranscription(w, httpReq)

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", w.Code)
				}
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < concurrency; i++ {
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent requests")
			}
		}
	})
}
