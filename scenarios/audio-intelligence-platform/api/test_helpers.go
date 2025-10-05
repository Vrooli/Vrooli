// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes a test logger
func setupTestLogger() func() {
	// Return a cleanup function that does nothing
	// Audio service creates its own logger
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Service    *AudioService
	MockServer *httptest.Server
	Cleanup    func()
}

// setupTestDB creates an in-memory test database
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// For now, we'll use a mock DB approach
	// In production, this would connect to a test PostgreSQL instance
	db, err := sql.Open("postgres", "postgres://test:test@localhost:5433/test_audio?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping test - database not available: %v", err)
	}

	// Initialize schema
	schema := `
		CREATE TABLE IF NOT EXISTS transcriptions (
			id UUID PRIMARY KEY,
			filename TEXT NOT NULL,
			file_path TEXT NOT NULL,
			transcription_text TEXT,
			duration_seconds FLOAT,
			file_size_bytes BIGINT,
			whisper_model_used TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			embedding_status TEXT DEFAULT 'pending'
		);

		CREATE TABLE IF NOT EXISTS ai_analyses (
			id UUID PRIMARY KEY,
			transcription_id UUID REFERENCES transcriptions(id),
			analysis_type TEXT NOT NULL,
			prompt_used TEXT,
			result_text TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			processing_time_ms INTEGER
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Skipf("Skipping test - failed to create schema: %v", err)
	}

	cleanup := func() {
		db.Exec("DROP TABLE IF EXISTS ai_analyses")
		db.Exec("DROP TABLE IF EXISTS transcriptions")
		db.Close()
	}

	return db, cleanup
}

// setupTestEnvironment creates a complete test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Create mock external services
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock responses for n8n webhooks
		if strings.Contains(r.URL.Path, "/webhook/transcription-upload") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "uploaded"})
			return
		}

		if strings.Contains(r.URL.Path, "/webhook/ai-analysis") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"analysis": "This is a test analysis result",
			})
			return
		}

		if strings.Contains(r.URL.Path, "/webhook/semantic-search") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []map[string]interface{}{
					{"id": uuid.New().String(), "score": 0.95},
				},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	// Setup database
	db, dbCleanup := setupTestDB(t)

	// Create audio service with mock URLs
	service := NewAudioService(
		db,
		mockServer.URL, // n8nURL
		mockServer.URL, // windmillURL
		mockServer.URL, // whisperURL
		mockServer.URL, // ollamaURL
		"localhost:9000", // minioEndpoint
		mockServer.URL, // qdrantURL
	)

	return &TestEnvironment{
		DB:         db,
		Service:    service,
		MockServer: mockServer,
		Cleanup: func() {
			dbCleanup()
			mockServer.Close()
		},
	}
}

// TestTranscription provides a pre-configured transcription for testing
type TestTranscription struct {
	Transcription *Transcription
	Cleanup       func()
}

// setupTestTranscription creates a test transcription
func setupTestTranscription(t *testing.T, db *sql.DB, filename string) *TestTranscription {
	transcription := &Transcription{
		ID:                uuid.New(),
		Filename:          filename,
		FilePath:          "/tmp/test/" + filename,
		TranscriptionText: "This is a test transcription",
		DurationSeconds:   120.5,
		FileSizeBytes:     1024000,
		WhisperModelUsed:  "base",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		EmbeddingStatus:   "completed",
	}

	_, err := db.Exec(`
		INSERT INTO transcriptions
		(id, filename, file_path, transcription_text, duration_seconds,
		 file_size_bytes, whisper_model_used, created_at, updated_at, embedding_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		transcription.ID, transcription.Filename, transcription.FilePath,
		transcription.TranscriptionText, transcription.DurationSeconds,
		transcription.FileSizeBytes, transcription.WhisperModelUsed,
		transcription.CreatedAt, transcription.UpdatedAt, transcription.EmbeddingStatus)

	if err != nil {
		t.Fatalf("Failed to create test transcription: %v", err)
	}

	return &TestTranscription{
		Transcription: transcription,
		Cleanup: func() {
			db.Exec("DELETE FROM transcriptions WHERE id = $1", transcription.ID)
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLVars     map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		var bodyBytes []byte
		var err error

		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default content type for POST/PUT requests with body
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set URL variables (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	return w, httpReq, nil
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
	for key, expectedValue := range expectedFields {
		actualValue, exists := response[key]
		if !exists {
			t.Errorf("Expected field '%s' not found in response", key)
			continue
		}

		if expectedValue != nil && actualValue != expectedValue {
			t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
		}
	}

	return response
}

// assertJSONArray validates that response contains an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON array response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
		return
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Error("Expected error field in response")
		return
	}

	if expectedErrorMessage != "" && !strings.Contains(errorMsg.(string), expectedErrorMessage) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, errorMsg)
	}
}

// createMultipartRequest creates a multipart form request for file upload testing
func createMultipartRequest(t *testing.T, path string, filename string, content []byte) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("audio", filename)
	if err != nil {
		return nil, err
	}

	if _, err := part.Write(content); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// AnalyzeRequest creates a test analysis request
func (g *TestDataGenerator) AnalyzeRequest(analysisType, customPrompt, model string) map[string]interface{} {
	req := map[string]interface{}{
		"analysis_type": analysisType,
	}
	if customPrompt != "" {
		req["custom_prompt"] = customPrompt
	}
	if model != "" {
		req["model"] = model
	}
	return req
}

// SearchRequest creates a test search request
func (g *TestDataGenerator) SearchRequest(query string, limit int) map[string]interface{} {
	req := map[string]interface{}{
		"query": query,
	}
	if limit > 0 {
		req["limit"] = limit
	}
	return req
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
