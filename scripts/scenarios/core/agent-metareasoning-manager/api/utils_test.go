package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	originalEnv := map[string]string{
		"PORT":               os.Getenv("PORT"),
		"DATABASE_URL":       os.Getenv("DATABASE_URL"),
		"N8N_BASE_URL":       os.Getenv("N8N_BASE_URL"),
		"WINDMILL_BASE_URL":  os.Getenv("WINDMILL_BASE_URL"),
		"WINDMILL_WORKSPACE": os.Getenv("WINDMILL_WORKSPACE"),
	}

	// Restore env vars after test
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected Config
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			expected: Config{
				Port:              "8093",
				DatabaseURL:       "postgres://postgres:postgres@localhost:5432/metareasoning?sslmode=disable",
				N8nBaseURL:        "http://localhost:5678",
				WindmillBaseURL:   "http://localhost:8000",
				WindmillWorkspace: "demo",
			},
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"PORT":               "9000",
				"DATABASE_URL":       "postgres://custom:custom@db:5432/test",
				"N8N_BASE_URL":       "http://n8n:5678",
				"WINDMILL_BASE_URL":  "http://windmill:8000",
				"WINDMILL_WORKSPACE": "production",
			},
			expected: Config{
				Port:              "9000",
				DatabaseURL:       "postgres://custom:custom@db:5432/test",
				N8nBaseURL:        "http://n8n:5678",
				WindmillBaseURL:   "http://windmill:8000",
				WindmillWorkspace: "production",
			},
		},
		{
			name: "partial custom values",
			envVars: map[string]string{
				"PORT":         "8080",
				"DATABASE_URL": "postgres://prod:prod@localhost:5432/prod",
			},
			expected: Config{
				Port:              "8080",
				DatabaseURL:       "postgres://prod:prod@localhost:5432/prod",
				N8nBaseURL:        "http://localhost:5678",
				WindmillBaseURL:   "http://localhost:8000",
				WindmillWorkspace: "demo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			os.Unsetenv("PORT")
			os.Unsetenv("DATABASE_URL")
			os.Unsetenv("N8N_BASE_URL")
			os.Unsetenv("WINDMILL_BASE_URL")
			os.Unsetenv("WINDMILL_WORKSPACE")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Load config
			config := LoadConfig()

			// Check values
			if config.Port != tt.expected.Port {
				t.Errorf("Port: got %s, want %s", config.Port, tt.expected.Port)
			}
			if config.DatabaseURL != tt.expected.DatabaseURL {
				t.Errorf("DatabaseURL: got %s, want %s", config.DatabaseURL, tt.expected.DatabaseURL)
			}
			if config.N8nBaseURL != tt.expected.N8nBaseURL {
				t.Errorf("N8nBaseURL: got %s, want %s", config.N8nBaseURL, tt.expected.N8nBaseURL)
			}
			if config.WindmillBaseURL != tt.expected.WindmillBaseURL {
				t.Errorf("WindmillBaseURL: got %s, want %s", config.WindmillBaseURL, tt.expected.WindmillBaseURL)
			}
			if config.WindmillWorkspace != tt.expected.WindmillWorkspace {
				t.Errorf("WindmillWorkspace: got %s, want %s", config.WindmillWorkspace, tt.expected.WindmillWorkspace)
			}
		})
	}
}

func TestCheckService(t *testing.T) {
	// Create test servers
	availableServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer availableServer.Close()

	unavailableServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer unavailableServer.Close()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "available service",
			url:      availableServer.URL,
			expected: true,
		},
		{
			name:     "unavailable service (500)",
			url:      unavailableServer.URL,
			expected: false,
		},
		{
			name:     "invalid URL",
			url:      "http://nonexistent.local:99999",
			expected: false,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: false,
		},
		{
			name:     "malformed URL",
			url:      "not-a-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkService(tt.url)
			if result != tt.expected {
				t.Errorf("checkService(%s): got %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		data        interface{}
		wantStatus  int
		wantHeaders map[string]string
	}{
		{
			name:   "write simple object",
			status: http.StatusOK,
			data: map[string]string{
				"message": "success",
				"id":      "123",
			},
			wantStatus: http.StatusOK,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:   "write array",
			status: http.StatusOK,
			data:   []string{"item1", "item2", "item3"},
			wantStatus: http.StatusOK,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:   "write with error status",
			status: http.StatusBadRequest,
			data: map[string]string{
				"error": "validation failed",
			},
			wantStatus: http.StatusBadRequest,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:       "write nil data",
			status:     http.StatusNoContent,
			data:       nil,
			wantStatus: http.StatusNoContent,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:   "write complex nested structure",
			status: http.StatusOK,
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"id":   123,
					"name": "Test User",
					"tags": []string{"admin", "user"},
				},
				"metadata": map[string]interface{}{
					"created": "2024-01-01",
					"version": 1.5,
				},
			},
			wantStatus: http.StatusOK,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tt.status, tt.data)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status code: got %d, want %d", w.Code, tt.wantStatus)
			}

			// Check headers
			for header, value := range tt.wantHeaders {
				got := w.Header().Get(header)
				if got != value {
					t.Errorf("Header %s: got %s, want %s", header, got, value)
				}
			}

			// Verify JSON is valid
			if tt.data != nil {
				var decoded interface{}
				err := json.NewDecoder(w.Body).Decode(&decoded)
				if err != nil {
					t.Errorf("Failed to decode JSON response: %v", err)
				}

				// Re-encode and compare (simple way to check structure)
				expectedJSON, _ := json.Marshal(tt.data)
				actualJSON, _ := json.Marshal(decoded)
				
				if string(expectedJSON) != string(actualJSON) {
					t.Errorf("JSON mismatch:\nExpected: %s\nGot: %s", 
						string(expectedJSON), string(actualJSON))
				}
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
	}{
		{
			name:       "bad request error",
			status:     http.StatusBadRequest,
			message:    "Invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized error",
			status:     http.StatusUnauthorized,
			message:    "Authentication required",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "not found error",
			status:     http.StatusNotFound,
			message:    "Resource not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal server error",
			status:     http.StatusInternalServerError,
			message:    "Something went wrong",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "empty message",
			status:     http.StatusBadRequest,
			message:    "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeError(w, tt.status, tt.message)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status code: got %d, want %d", w.Code, tt.wantStatus)
			}

			// Check Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type: got %s, want application/json", contentType)
			}

			// Check error message in response
			var response map[string]string
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("Error message: got %s, want %s", response["error"], tt.message)
			}
		})
	}
}

func TestParseQueryInt(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		param      string
		defaultVal int
		expected   int
	}{
		{
			name:       "valid integer",
			queryStr:   "page=5&size=10",
			param:      "page",
			defaultVal: 1,
			expected:   5,
		},
		{
			name:       "missing parameter",
			queryStr:   "size=10",
			param:      "page",
			defaultVal: 1,
			expected:   1,
		},
		{
			name:       "invalid integer",
			queryStr:   "page=abc",
			param:      "page",
			defaultVal: 1,
			expected:   1,
		},
		{
			name:       "negative integer",
			queryStr:   "page=-5",
			param:      "page",
			defaultVal: 1,
			expected:   -5,
		},
		{
			name:       "zero value",
			queryStr:   "page=0",
			param:      "page",
			defaultVal: 1,
			expected:   0,
		},
		{
			name:       "large integer",
			queryStr:   "limit=1000000",
			param:      "limit",
			defaultVal: 100,
			expected:   1000000,
		},
		{
			name:       "empty query string",
			queryStr:   "",
			param:      "page",
			defaultVal: 10,
			expected:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.queryStr, nil)
			result := parseQueryInt(req, tt.param, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("parseQueryInt: got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestParseQueryBool(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		param      string
		defaultVal bool
		expected   bool
	}{
		{
			name:       "true string",
			queryStr:   "active=true",
			param:      "active",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "false string",
			queryStr:   "active=false",
			param:      "active",
			defaultVal: true,
			expected:   false,
		},
		{
			name:       "1 as true",
			queryStr:   "active=1",
			param:      "active",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "0 as false",
			queryStr:   "active=0",
			param:      "active",
			defaultVal: true,
			expected:   false,
		},
		{
			name:       "yes as true",
			queryStr:   "enabled=yes",
			param:      "enabled",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "no as false",
			queryStr:   "enabled=no",
			param:      "enabled",
			defaultVal: true,
			expected:   false,
		},
		{
			name:       "missing parameter",
			queryStr:   "other=value",
			param:      "active",
			defaultVal: true,
			expected:   true,
		},
		{
			name:       "invalid boolean",
			queryStr:   "active=maybe",
			param:      "active",
			defaultVal: false,
			expected:   false,
		},
		{
			name:       "uppercase TRUE",
			queryStr:   "active=TRUE",
			param:      "active",
			defaultVal: false,
			expected:   true,
		},
		{
			name:       "uppercase FALSE",
			queryStr:   "active=FALSE",
			param:      "active",
			defaultVal: true,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.queryStr, nil)
			result := parseQueryBool(req, tt.param, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("parseQueryBool: got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEnvOrDefault(t *testing.T) {
	// Save and restore env var
	testKey := "TEST_ENV_VAR"
	originalValue := os.Getenv(testKey)
	defer func() {
		if originalValue == "" {
			os.Unsetenv(testKey)
		} else {
			os.Setenv(testKey, originalValue)
		}
	}()

	tests := []struct {
		name       string
		envValue   string
		setEnv     bool
		defaultVal string
		expected   string
	}{
		{
			name:       "env var set",
			envValue:   "custom_value",
			setEnv:     true,
			defaultVal: "default_value",
			expected:   "custom_value",
		},
		{
			name:       "env var not set",
			envValue:   "",
			setEnv:     false,
			defaultVal: "default_value",
			expected:   "default_value",
		},
		{
			name:       "empty env var",
			envValue:   "",
			setEnv:     true,
			defaultVal: "default_value",
			expected:   "default_value",
		},
		{
			name:       "whitespace env var",
			envValue:   "  ",
			setEnv:     true,
			defaultVal: "default_value",
			expected:   "  ", // Should preserve whitespace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup env var
			if tt.setEnv {
				os.Setenv(testKey, tt.envValue)
			} else {
				os.Unsetenv(testKey)
			}

			// Test envOrDefault function
			result := envOrDefault(testKey, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("envOrDefault: got %s, want %s", result, tt.expected)
			}
		})
	}
}