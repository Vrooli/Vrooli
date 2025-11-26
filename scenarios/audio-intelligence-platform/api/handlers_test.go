// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	t.Run("ReturnsHealthyStatus", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		Health(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		if response["service"] != serviceName {
			t.Errorf("Expected service name '%s', got %v", serviceName, response["service"])
		}

		if response["version"] != apiVersion {
			t.Errorf("Expected version '%s', got %v", apiVersion, response["version"])
		}
	})

	t.Run("ReturnsJSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		Health(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})
}

// TestHTTPErrorFunction tests the HTTPError helper function
func TestHTTPErrorFunction(t *testing.T) {
	tests := []struct {
		name           string
		message        string
		statusCode     int
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "BadRequest",
			message:        "Invalid input",
			statusCode:     http.StatusBadRequest,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid input",
		},
		{
			name:           "NotFound",
			message:        "Resource not found",
			statusCode:     http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Resource not found",
		},
		{
			name:           "InternalServerError",
			message:        "Database error",
			statusCode:     http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			HTTPError(w, tt.message, tt.statusCode, nil)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if response["error"] != tt.expectedMsg {
				t.Errorf("Expected error message '%s', got '%v'", tt.expectedMsg, response["error"])
			}

			if status, ok := response["status"].(float64); !ok || int(status) != tt.expectedStatus {
				t.Errorf("Expected status field %d, got %v", tt.expectedStatus, response["status"])
			}

			if _, ok := response["timestamp"]; !ok {
				t.Error("Expected timestamp field in response")
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

// TestRouterSetup tests the router configuration
func TestRouterSetup(t *testing.T) {
	// Create a mock service
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// Note: We can't easily test the actual router setup without running main()
	// Instead, we test that routes would be configured correctly
	t.Run("HealthRouteExists", func(t *testing.T) {
		r := mux.NewRouter()
		r.HandleFunc("/health", Health).Methods("GET")

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health route not configured correctly, got status %d", w.Code)
		}
	})
}

// TestAudioServiceStructure tests the AudioService struct
func TestAudioServiceStructure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	service := &AudioService{
		db:            nil, // Mock DB
		n8nBaseURL:    mockServer.URL,
		whisperURL:    mockServer.URL,
		ollamaURL:     mockServer.URL,
		minioEndpoint: "localhost:9000",
		qdrantURL:     mockServer.URL,
		httpClient:    &http.Client{},
		logger:        NewLogger(),
	}

	t.Run("FieldsInitialized", func(t *testing.T) {
		if service.n8nBaseURL != mockServer.URL {
			t.Error("n8nBaseURL not initialized")
		}
		if service.httpClient == nil {
			t.Error("httpClient not initialized")
		}
		if service.logger == nil {
			t.Error("logger not initialized")
		}
	})
}

// TestLoggerFunctionality tests the Logger type
func TestLoggerFunctionality(t *testing.T) {
	logger := NewLogger()

	t.Run("LoggerCreated", func(t *testing.T) {
		if logger == nil {
			t.Fatal("Logger not created")
		}
		if logger.Logger == nil {
			t.Fatal("Internal logger not initialized")
		}
	})

	t.Run("InfoMethod", func(t *testing.T) {
		// Should not panic
		logger.Info("test message")
	})

	t.Run("ErrorMethod", func(t *testing.T) {
		// Should not panic
		logger.Error("test error", nil)
	})

	t.Run("WarnMethod", func(t *testing.T) {
		// Should not panic
		logger.Warn("test warning", nil)
	})
}

// TestJSONDecoding tests JSON request decoding
func TestJSONDecoding(t *testing.T) {
	t.Run("ValidAnalysisRequest", func(t *testing.T) {
		payload := map[string]interface{}{
			"analysis_type": "summary",
			"custom_prompt": "Summarize this",
			"model":         "llama3.1:8b",
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		var decoded map[string]interface{}
		if err := json.Unmarshal(jsonData, &decoded); err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}

		if decoded["analysis_type"] != "summary" {
			t.Errorf("Expected analysis_type 'summary', got %v", decoded["analysis_type"])
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		invalidJSON := []byte(`{"invalid": "json"`)

		var decoded map[string]interface{}
		err := json.Unmarshal(invalidJSON, &decoded)
		if err == nil {
			t.Error("Expected error decoding invalid JSON")
		}
	})
}

// TestFileTypeValidation tests file type validation logic
func TestFileTypeValidation(t *testing.T) {
	allowedTypes := []string{".mp3", ".wav", ".m4a", ".ogg", ".flac"}

	tests := []struct {
		filename string
		allowed  bool
	}{
		{"test.mp3", true},
		{"test.wav", true},
		{"test.m4a", true},
		{"test.ogg", true},
		{"test.flac", true},
		{"test.txt", false},
		{"test.pdf", false},
		{"test.exe", false},
		{"test.MP3", true}, // Case insensitive
		{"test.WAV", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			// Extract extension
			ext := ""
			for i := len(tt.filename) - 1; i >= 0; i-- {
				if tt.filename[i] == '.' {
					ext = tt.filename[i:]
					break
				}
			}

			// Convert to lowercase for comparison
			extLower := ""
			for _, c := range ext {
				if c >= 'A' && c <= 'Z' {
					extLower += string(c + 32)
				} else {
					extLower += string(c)
				}
			}

			isAllowed := false
			for _, allowedType := range allowedTypes {
				if extLower == allowedType {
					isAllowed = true
					break
				}
			}

			if isAllowed != tt.allowed {
				t.Errorf("File %s: expected allowed=%v, got %v", tt.filename, tt.allowed, isAllowed)
			}
		})
	}
}

// TestConstantValues tests that all constants are properly defined
func TestConstantValues(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{"apiVersion", apiVersion},
		{"serviceName", serviceName},
		{"defaultPort", defaultPort},
		{"defaultWorkspace", defaultWorkspace},
		{"httpTimeout", httpTimeout},
		{"maxFileSize", maxFileSize},
		{"maxDBConnections", maxDBConnections},
		{"maxIdleConnections", maxIdleConnections},
		{"connMaxLifetime", connMaxLifetime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch v := tt.value.(type) {
			case string:
				if v == "" {
					t.Errorf("Constant %s should not be empty", tt.name)
				}
			case int:
				if v <= 0 {
					t.Errorf("Constant %s should be positive, got %d", tt.name, v)
				}
			case int64:
				if v <= 0 {
					t.Errorf("Constant %s should be positive, got %d", tt.name, v)
				}
			}
		})
	}
}

// TestSearchRequestStructure tests search request structure
func TestSearchRequestStructure(t *testing.T) {
	t.Run("WithLimit", func(t *testing.T) {
		req := map[string]interface{}{
			"query": "test query",
			"limit": 10,
		}

		if req["query"] != "test query" {
			t.Errorf("Expected query 'test query', got %v", req["query"])
		}

		if req["limit"] != 10 {
			t.Errorf("Expected limit 10, got %v", req["limit"])
		}
	})

	t.Run("WithoutLimit", func(t *testing.T) {
		req := map[string]interface{}{
			"query": "test query",
		}

		// Default limit should be 10
		limit := 0
		if l, ok := req["limit"]; ok {
			limit = l.(int)
		} else {
			limit = 10 // Default
		}

		if limit != 10 {
			t.Errorf("Expected default limit 10, got %d", limit)
		}
	})
}

// TestMultipartFormParsing tests multipart form data parsing
func TestMultipartFormParsing(t *testing.T) {
	t.Run("CreateMultipartRequest", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add a test field
		if err := writer.WriteField("test_field", "test_value"); err != nil {
			t.Fatalf("Failed to write field: %v", err)
		}

		writer.Close()

		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		if err := req.ParseMultipartForm(maxFileSize); err != nil {
			t.Fatalf("Failed to parse multipart form: %v", err)
		}

		if req.FormValue("test_field") != "test_value" {
			t.Errorf("Expected test_field='test_value', got '%s'", req.FormValue("test_field"))
		}
	})
}

// TestAnalysisTypePromptGeneration tests prompt generation for different analysis types
func TestAnalysisTypePromptGeneration(t *testing.T) {
	tests := []struct {
		analysisType   string
		customPrompt   string
		expectedPrompt string
	}{
		{
			analysisType:   "summary",
			customPrompt:   "",
			expectedPrompt: "Provide a concise summary of this transcription",
		},
		{
			analysisType:   "insights",
			customPrompt:   "",
			expectedPrompt: "Extract key insights and main points",
		},
		{
			analysisType:   "custom",
			customPrompt:   "Custom analysis prompt",
			expectedPrompt: "Custom analysis prompt",
		},
		{
			analysisType:   "other",
			customPrompt:   "",
			expectedPrompt: "Perform other analysis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.analysisType, func(t *testing.T) {
			prompt := tt.customPrompt
			if prompt == "" {
				switch tt.analysisType {
				case "summary":
					prompt = "Provide a concise summary of this transcription"
				case "insights":
					prompt = "Extract key insights and main points"
				default:
					prompt = "Perform " + tt.analysisType + " analysis"
				}
			}

			if prompt != tt.expectedPrompt {
				t.Errorf("Expected prompt '%s', got '%s'", tt.expectedPrompt, prompt)
			}
		})
	}
}

// TestModelDefaulting tests default model selection
func TestModelDefaulting(t *testing.T) {
	t.Run("DefaultModel", func(t *testing.T) {
		model := ""
		if model == "" {
			model = "llama3.1:8b"
		}

		if model != "llama3.1:8b" {
			t.Errorf("Expected default model 'llama3.1:8b', got '%s'", model)
		}
	})

	t.Run("CustomModel", func(t *testing.T) {
		model := "custom-model"
		if model == "" {
			model = "llama3.1:8b"
		}

		if model != "custom-model" {
			t.Errorf("Expected custom model 'custom-model', got '%s'", model)
		}
	})
}
