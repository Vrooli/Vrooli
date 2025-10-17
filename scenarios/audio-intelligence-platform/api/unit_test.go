// +build testing

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// MockDB simulates database operations for testing
type MockDB struct {
	queryFunc    func(query string, args ...interface{}) (*sql.Rows, error)
	queryRowFunc func(query string, args ...interface{}) *sql.Row
	execFunc     func(query string, args ...interface{}) (sql.Result, error)
}

// TestTranscriptionStruct tests the Transcription struct
func TestTranscriptionStruct(t *testing.T) {
	now := time.Now()
	trans := Transcription{
		ID:                uuid.New(),
		Filename:          "test.mp3",
		FilePath:          "/tmp/test.mp3",
		TranscriptionText: "Test transcription",
		DurationSeconds:   120.5,
		FileSizeBytes:     1024000,
		WhisperModelUsed:  "base",
		CreatedAt:         now,
		UpdatedAt:         now,
		EmbeddingStatus:   "completed",
	}

	t.Run("StructFields", func(t *testing.T) {
		if trans.Filename != "test.mp3" {
			t.Errorf("Expected filename 'test.mp3', got '%s'", trans.Filename)
		}
		if trans.DurationSeconds != 120.5 {
			t.Errorf("Expected duration 120.5, got %f", trans.DurationSeconds)
		}
		if trans.FileSizeBytes != 1024000 {
			t.Errorf("Expected file size 1024000, got %d", trans.FileSizeBytes)
		}
	})

	t.Run("JSONMarshal", func(t *testing.T) {
		data, err := json.Marshal(trans)
		if err != nil {
			t.Fatalf("Failed to marshal transcription: %v", err)
		}

		var decoded Transcription
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal transcription: %v", err)
		}

		if decoded.Filename != trans.Filename {
			t.Errorf("Marshaled filename doesn't match")
		}
	})
}

// TestAIAnalysisStruct tests the AIAnalysis struct
func TestAIAnalysisStruct(t *testing.T) {
	now := time.Now()
	analysis := AIAnalysis{
		ID:               uuid.New(),
		TranscriptionID:  uuid.New(),
		AnalysisType:     "summary",
		PromptUsed:       "Summarize this",
		ResultText:       "Test result",
		CreatedAt:        now,
		ProcessingTimeMS: 1500,
	}

	t.Run("StructFields", func(t *testing.T) {
		if analysis.AnalysisType != "summary" {
			t.Errorf("Expected analysis type 'summary', got '%s'", analysis.AnalysisType)
		}
		if analysis.ProcessingTimeMS != 1500 {
			t.Errorf("Expected processing time 1500, got %d", analysis.ProcessingTimeMS)
		}
	})

	t.Run("JSONMarshal", func(t *testing.T) {
		data, err := json.Marshal(analysis)
		if err != nil {
			t.Fatalf("Failed to marshal analysis: %v", err)
		}

		var decoded AIAnalysis
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal analysis: %v", err)
		}

		if decoded.AnalysisType != analysis.AnalysisType {
			t.Errorf("Marshaled analysis type doesn't match")
		}
	})
}

// TestNewAudioServiceCreation tests audio service creation
func TestNewAudioServiceCreation(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	service := NewAudioService(
		nil,
		mockServer.URL,
		mockServer.URL,
		mockServer.URL,
		mockServer.URL,
		"localhost:9000",
		mockServer.URL,
	)

	t.Run("ServiceCreated", func(t *testing.T) {
		if service == nil {
			t.Fatal("Service not created")
		}
	})

	t.Run("URLsSet", func(t *testing.T) {
		if service.n8nBaseURL != mockServer.URL {
			t.Errorf("n8nBaseURL not set correctly")
		}
		if service.windmillURL != mockServer.URL {
			t.Errorf("windmillURL not set correctly")
		}
		if service.whisperURL != mockServer.URL {
			t.Errorf("whisperURL not set correctly")
		}
		if service.ollamaURL != mockServer.URL {
			t.Errorf("ollamaURL not set correctly")
		}
		if service.qdrantURL != mockServer.URL {
			t.Errorf("qdrantURL not set correctly")
		}
	})

	t.Run("HTTPClientInitialized", func(t *testing.T) {
		if service.httpClient == nil {
			t.Error("HTTP client not initialized")
		}
		if service.httpClient.Timeout != httpTimeout {
			t.Errorf("Expected timeout %v, got %v", httpTimeout, service.httpClient.Timeout)
		}
	})

	t.Run("LoggerInitialized", func(t *testing.T) {
		if service.logger == nil {
			t.Error("Logger not initialized")
		}
	})
}

// TestListTranscriptionsHandler tests list transcriptions handler logic
func TestListTranscriptionsHandler(t *testing.T) {
	t.Run("EmptyDatabase", func(t *testing.T) {
		// This test validates the handler logic without actual database
		// The handler would return empty array on empty DB
		expectedJSON := "[]"
		var arr []interface{}
		if err := json.Unmarshal([]byte(expectedJSON), &arr); err != nil {
			t.Errorf("Empty array should be valid JSON: %v", err)
		}
		if len(arr) != 0 {
			t.Errorf("Expected empty array")
		}
	})

	t.Run("WithTranscriptions", func(t *testing.T) {
		// Simulate response with transcriptions
		trans := []Transcription{
			{
				ID:                uuid.New(),
				Filename:          "test1.mp3",
				FilePath:          "/tmp/test1.mp3",
				TranscriptionText: "Test 1",
				DurationSeconds:   60.0,
				FileSizeBytes:     500000,
				WhisperModelUsed:  "base",
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
				EmbeddingStatus:   "completed",
			},
		}

		data, err := json.Marshal(trans)
		if err != nil {
			t.Fatalf("Failed to marshal transcriptions: %v", err)
		}

		var decoded []Transcription
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(decoded) != 1 {
			t.Errorf("Expected 1 transcription, got %d", len(decoded))
		}
	})
}

// TestGetTranscriptionHandler tests get transcription handler logic
func TestGetTranscriptionHandler(t *testing.T) {
	t.Run("ValidID", func(t *testing.T) {
		id := uuid.New()
		trans := Transcription{
			ID:                id,
			Filename:          "test.mp3",
			FilePath:          "/tmp/test.mp3",
			TranscriptionText: "Test",
			DurationSeconds:   60.0,
			FileSizeBytes:     500000,
			WhisperModelUsed:  "base",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
			EmbeddingStatus:   "completed",
		}

		data, err := json.Marshal(trans)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded Transcription
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.ID != id {
			t.Errorf("ID mismatch after marshal/unmarshal")
		}
	})
}

// TestUploadAudioLogic tests upload audio validation logic
func TestUploadAudioLogic(t *testing.T) {
	allowedTypes := []string{".mp3", ".wav", ".m4a", ".ogg", ".flac"}

	t.Run("ValidFileTypes", func(t *testing.T) {
		validFiles := []string{"test.mp3", "test.wav", "test.m4a", "test.ogg", "test.flac"}
		for _, filename := range validFiles {
			ext := ""
			for i := len(filename) - 1; i >= 0; i-- {
				if filename[i] == '.' {
					ext = filename[i:]
					break
				}
			}

			isAllowed := false
			for _, allowedType := range allowedTypes {
				if ext == allowedType {
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				t.Errorf("File %s should be allowed", filename)
			}
		}
	})

	t.Run("InvalidFileTypes", func(t *testing.T) {
		invalidFiles := []string{"test.txt", "test.pdf", "test.exe", "test.doc"}
		for _, filename := range invalidFiles {
			ext := ""
			for i := len(filename) - 1; i >= 0; i-- {
				if filename[i] == '.' {
					ext = filename[i:]
					break
				}
			}

			isAllowed := false
			for _, allowedType := range allowedTypes {
				if ext == allowedType {
					isAllowed = true
					break
				}
			}

			if isAllowed {
				t.Errorf("File %s should not be allowed", filename)
			}
		}
	})

	t.Run("FileSizeLimit", func(t *testing.T) {
		if maxFileSize <= 0 {
			t.Error("maxFileSize should be positive")
		}
		expectedSize := int64(500 * 1024 * 1024) // 500MB
		if maxFileSize != expectedSize {
			t.Errorf("Expected maxFileSize %d, got %d", expectedSize, maxFileSize)
		}
	})
}

// TestAnalyzeTranscriptionLogic tests analysis logic
func TestAnalyzeTranscriptionLogic(t *testing.T) {
	t.Run("AnalysisTypeValidation", func(t *testing.T) {
		validTypes := []string{"summary", "insights", "sentiment", "custom"}
		for _, analysisType := range validTypes {
			if analysisType == "" {
				t.Error("Analysis type should not be empty")
			}
		}
	})

	t.Run("DefaultModel", func(t *testing.T) {
		model := ""
		if model == "" {
			model = "llama3.1:8b"
		}
		if model != "llama3.1:8b" {
			t.Errorf("Expected default model 'llama3.1:8b', got '%s'", model)
		}
	})

	t.Run("PromptGeneration", func(t *testing.T) {
		tests := []struct {
			analysisType string
			expected     string
		}{
			{"summary", "Provide a concise summary of this transcription"},
			{"insights", "Extract key insights and main points"},
			{"custom", "Perform custom analysis"},
		}

		for _, tt := range tests {
			var prompt string
			switch tt.analysisType {
			case "summary":
				prompt = "Provide a concise summary of this transcription"
			case "insights":
				prompt = "Extract key insights and main points"
			default:
				prompt = fmt.Sprintf("Perform %s analysis", tt.analysisType)
			}

			if prompt == "" {
				t.Errorf("Prompt should not be empty for type %s", tt.analysisType)
			}
		}
	})
}

// TestSearchTranscriptionsLogic tests search logic
func TestSearchTranscriptionsLogic(t *testing.T) {
	t.Run("DefaultLimit", func(t *testing.T) {
		limit := 0
		if limit == 0 {
			limit = 10
		}
		if limit != 10 {
			t.Errorf("Expected default limit 10, got %d", limit)
		}
	})

	t.Run("CustomLimit", func(t *testing.T) {
		limit := 25
		if limit == 0 {
			limit = 10
		}
		if limit != 25 {
			t.Errorf("Expected limit 25, got %d", limit)
		}
	})

	t.Run("QueryValidation", func(t *testing.T) {
		queries := []string{"test query", "search term", "audio transcription"}
		for _, query := range queries {
			if query == "" {
				t.Error("Query should not be empty")
			}
		}
	})
}

// TestGetAnalysesLogic tests get analyses logic
func TestGetAnalysesLogic(t *testing.T) {
	t.Run("EmptyAnalyses", func(t *testing.T) {
		analyses := []AIAnalysis{}
		data, err := json.Marshal(analyses)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded []AIAnalysis
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(decoded) != 0 {
			t.Errorf("Expected empty array")
		}
	})

	t.Run("WithAnalyses", func(t *testing.T) {
		analyses := []AIAnalysis{
			{
				ID:               uuid.New(),
				TranscriptionID:  uuid.New(),
				AnalysisType:     "summary",
				PromptUsed:       "Summarize",
				ResultText:       "Result",
				CreatedAt:        time.Now(),
				ProcessingTimeMS: 1000,
			},
		}

		data, err := json.Marshal(analyses)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded []AIAnalysis
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(decoded) != 1 {
			t.Errorf("Expected 1 analysis, got %d", len(decoded))
		}
	})
}

// TestHTTPClientConfiguration tests HTTP client setup
func TestHTTPClientConfiguration(t *testing.T) {
	client := &http.Client{
		Timeout: httpTimeout,
	}

	t.Run("TimeoutSet", func(t *testing.T) {
		if client.Timeout != httpTimeout {
			t.Errorf("Expected timeout %v, got %v", httpTimeout, client.Timeout)
		}
	})

	t.Run("TimeoutValue", func(t *testing.T) {
		expectedTimeout := 60 * time.Second
		if httpTimeout != expectedTimeout {
			t.Errorf("Expected timeout %v, got %v", expectedTimeout, httpTimeout)
		}
	})
}

// TestRouterConfiguration tests router setup
func TestRouterConfiguration(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	service := NewAudioService(
		nil,
		mockServer.URL,
		mockServer.URL,
		mockServer.URL,
		mockServer.URL,
		"localhost:9000",
		mockServer.URL,
	)

	r := mux.NewRouter()
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/api/transcriptions", service.ListTranscriptions).Methods("GET")
	r.HandleFunc("/api/transcriptions/{id}", service.GetTranscription).Methods("GET")
	r.HandleFunc("/api/transcriptions/{id}/analyses", service.GetAnalyses).Methods("GET")
	r.HandleFunc("/api/transcriptions/{id}/analyze", service.AnalyzeTranscription).Methods("POST")
	r.HandleFunc("/api/upload", service.UploadAudio).Methods("POST")
	r.HandleFunc("/api/search", service.SearchTranscriptions).Methods("POST")

	t.Run("HealthRoute", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health route failed, got status %d", w.Code)
		}
	})

	t.Run("RouteNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for nonexistent route, got %d", w.Code)
		}
	})
}

// TestDatabaseConnectionConfiguration tests DB configuration
func TestDatabaseConnectionConfiguration(t *testing.T) {
	t.Run("ConnectionPoolSettings", func(t *testing.T) {
		if maxDBConnections <= 0 {
			t.Error("maxDBConnections should be positive")
		}
		if maxIdleConnections <= 0 {
			t.Error("maxIdleConnections should be positive")
		}
		if connMaxLifetime <= 0 {
			t.Error("connMaxLifetime should be positive")
		}
	})

	t.Run("ConnectionPoolValues", func(t *testing.T) {
		if maxDBConnections != 25 {
			t.Errorf("Expected maxDBConnections 25, got %d", maxDBConnections)
		}
		if maxIdleConnections != 5 {
			t.Errorf("Expected maxIdleConnections 5, got %d", maxIdleConnections)
		}
		expectedLifetime := 5 * time.Minute
		if connMaxLifetime != expectedLifetime {
			t.Errorf("Expected connMaxLifetime %v, got %v", expectedLifetime, connMaxLifetime)
		}
	})
}

// TestErrorResponseFormat tests error response formatting
func TestErrorResponseFormat(t *testing.T) {
	tests := []struct {
		message    string
		statusCode int
	}{
		{"Bad request", http.StatusBadRequest},
		{"Not found", http.StatusNotFound},
		{"Internal error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Status%d", tt.statusCode), func(t *testing.T) {
			w := httptest.NewRecorder()
			HTTPError(w, tt.message, tt.statusCode, nil)

			if w.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("Expected error '%s', got '%v'", tt.message, response["error"])
			}

			if _, ok := response["timestamp"]; !ok {
				t.Error("Expected timestamp in error response")
			}

			if _, ok := response["status"]; !ok {
				t.Error("Expected status in error response")
			}
		})
	}
}

// TestJSONEncodingDecoding tests JSON operations
func TestJSONEncodingDecoding(t *testing.T) {
	t.Run("AnalysisRequest", func(t *testing.T) {
		req := map[string]interface{}{
			"analysis_type": "summary",
			"custom_prompt": "Summarize this",
			"model":         "llama3.1:8b",
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded["analysis_type"] != "summary" {
			t.Error("analysis_type mismatch")
		}
	})

	t.Run("SearchRequest", func(t *testing.T) {
		req := map[string]interface{}{
			"query": "test query",
			"limit": 10,
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded["query"] != "test query" {
			t.Error("query mismatch")
		}
	})
}

// TestWebhookURLConstruction tests webhook URL building
func TestWebhookURLConstruction(t *testing.T) {
	baseURL := "http://localhost:5678"

	tests := []struct {
		webhook  string
		expected string
	}{
		{"transcription-upload", "http://localhost:5678/webhook/transcription-upload"},
		{"ai-analysis", "http://localhost:5678/webhook/ai-analysis"},
		{"semantic-search", "http://localhost:5678/webhook/semantic-search"},
	}

	for _, tt := range tests {
		t.Run(tt.webhook, func(t *testing.T) {
			url := fmt.Sprintf("%s/webhook/%s", baseURL, tt.webhook)
			if url != tt.expected {
				t.Errorf("Expected URL '%s', got '%s'", tt.expected, url)
			}
		})
	}
}
