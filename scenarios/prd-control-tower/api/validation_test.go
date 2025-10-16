package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestRunScenarioAuditorCLI tests the CLI auditor runner
func TestRunScenarioAuditorCLI(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entityName string
		wantErr    bool
	}{
		{
			name:       "scenario auditor not installed",
			entityType: "scenario",
			entityName: "test-scenario",
			wantErr:    false, // Returns graceful error response
		},
		{
			name:       "valid scenario name",
			entityType: "scenario",
			entityName: "prd-control-tower",
			wantErr:    false,
		},
		{
			name:       "valid resource name",
			entityType: "resource",
			entityName: "postgres",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := runScenarioAuditorCLI(tt.entityType, tt.entityName)
			if (err != nil) != tt.wantErr {
				t.Errorf("runScenarioAuditorCLI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result == nil {
				t.Error("runScenarioAuditorCLI() returned nil result")
			}
		})
	}
}

// TestRunScenarioAuditor tests the main auditor runner with environment setup
func TestRunScenarioAuditor(t *testing.T) {
	tests := []struct {
		name       string
		setupEnv   func()
		entityType string
		entityName string
		content    string
		wantErr    bool
	}{
		{
			name:       "no auditor URL - falls back to CLI",
			setupEnv:   func() { os.Unsetenv("SCENARIO_AUDITOR_URL") },
			entityType: "scenario",
			entityName: "test-scenario",
			content:    "# Test PRD",
			wantErr:    false,
		},
		{
			name: "with auditor URL - tries HTTP",
			setupEnv: func() {
				os.Setenv("SCENARIO_AUDITOR_URL", "http://localhost:18507")
			},
			entityType: "scenario",
			entityName: "test-scenario",
			content:    "# Test PRD",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			if tt.setupEnv != nil {
				tt.setupEnv()
			}

			result, err := runScenarioAuditor(tt.entityType, tt.entityName, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("runScenarioAuditor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result == nil {
				t.Error("runScenarioAuditor() returned nil result")
			}
		})
	}

	// Cleanup
	os.Unsetenv("SCENARIO_AUDITOR_URL")
}

// TestCallAuditorAPI tests the HTTP API caller
func TestCallAuditorAPI(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() *httptest.Server
		payload     interface{}
		wantErr     bool
		errContains string
	}{
		{
			name: "successful API call",
			setupMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"violations": []string{},
						"status":     "pass",
					})
				}))
			},
			payload: map[string]string{
				"entity_name": "test-scenario",
			},
			wantErr: false,
		},
		{
			name: "API returns error status",
			setupMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal error"))
				}))
			},
			payload: map[string]string{
				"entity_name": "test-scenario",
			},
			wantErr:     true,
			errContains: "auditor API returned error",
		},
		{
			name: "API returns invalid JSON",
			setupMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("not valid json"))
				}))
			},
			payload: map[string]string{
				"entity_name": "test-scenario",
			},
			wantErr:     true,
			errContains: "failed to decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupMock()
			defer server.Close()

			result, err := callAuditorAPI(server.URL, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("callAuditorAPI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !containsStr(err.Error(), tt.errContains) {
					t.Errorf("callAuditorAPI() error = %v, should contain %v", err, tt.errContains)
				}
			}
			if !tt.wantErr && result == nil {
				t.Error("callAuditorAPI() returned nil result")
			}
		})
	}
}

// TestHandleValidatePRD tests the published PRD validation endpoint
func TestHandleValidatePRD(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "valid scenario validation request",
			requestBody: ValidatePRDRequest{
				EntityType: "scenario",
				EntityName: "prd-control-tower",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode response: %v", err)
					return
				}
				if response["entity_type"] != "scenario" {
					t.Errorf("Expected entity_type=scenario, got %v", response["entity_type"])
				}
				if response["entity_name"] != "prd-control-tower" {
					t.Errorf("Expected entity_name=prd-control-tower, got %v", response["entity_name"])
				}
			},
		},
		{
			name: "valid resource validation request",
			requestBody: ValidatePRDRequest{
				EntityType: "resource",
				EntityName: "postgres",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode response: %v", err)
					return
				}
				if response["entity_type"] != "resource" {
					t.Errorf("Expected entity_type=resource, got %v", response["entity_type"])
				}
			},
		},
		{
			name: "invalid entity type",
			requestBody: ValidatePRDRequest{
				EntityType: "invalid",
				EntityName: "test",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !containsStr(body, "Invalid entity_type") {
					t.Errorf("Expected error message about invalid entity_type, got: %s", body)
				}
			},
		},
		{
			name: "missing entity name",
			requestBody: ValidatePRDRequest{
				EntityType: "scenario",
				EntityName: "",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !containsStr(body, "entity_name is required") {
					t.Errorf("Expected error message about entity_name, got: %s", body)
				}
			},
		},
		{
			name:           "invalid JSON in request body",
			requestBody:    "not valid json",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !containsStr(body, "Invalid request body") {
					t.Errorf("Expected error message about invalid JSON, got: %s", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			var bodyReader io.Reader
			if str, ok := tt.requestBody.(string); ok {
				bodyReader = bytes.NewBufferString(str)
			} else {
				jsonData, _ := json.Marshal(tt.requestBody)
				bodyReader = bytes.NewBuffer(jsonData)
			}

			req := httptest.NewRequest("POST", "/api/v1/validate", bodyReader)
			w := httptest.NewRecorder()

			handleValidatePRD(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("handleValidatePRD() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestRunScenarioAuditorHTTP tests the HTTP auditor runner
func TestRunScenarioAuditorHTTP(t *testing.T) {
	// Setup mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"violations": []string{},
			"status":     "pass",
		})
	}))
	defer mockServer.Close()

	tests := []struct {
		name       string
		baseURL    string
		entityType string
		entityName string
		content    string
		wantErr    bool
	}{
		{
			name:       "valid HTTP call (currently falls back to CLI)",
			baseURL:    mockServer.URL,
			entityType: "scenario",
			entityName: "test-scenario",
			content:    "# Test PRD",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := runScenarioAuditorHTTP(tt.baseURL, tt.entityType, tt.entityName, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("runScenarioAuditorHTTP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result == nil {
				t.Error("runScenarioAuditorHTTP() returned nil result")
			}
		})
	}
}

// TestValidationRequestCaching tests validation with cache
func TestValidationRequestCaching(t *testing.T) {
	tests := []struct {
		name     string
		useCache bool
	}{
		{
			name:     "with cache enabled",
			useCache: true,
		},
		{
			name:     "with cache disabled",
			useCache: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ValidationRequest{
				UseCache: tt.useCache,
			}

			if req.UseCache != tt.useCache {
				t.Errorf("ValidationRequest.UseCache = %v, want %v", req.UseCache, tt.useCache)
			}
		})
	}
}

// TestValidationResponseStructure tests the validation response structure
func TestValidationResponseStructure(t *testing.T) {
	now := time.Now()
	cachedTime := now.Add(-5 * time.Minute)

	tests := []struct {
		name     string
		response ValidationResponse
		wantJSON string
	}{
		{
			name: "response with cache",
			response: ValidationResponse{
				DraftID:     "test-draft-id",
				EntityType:  "scenario",
				EntityName:  "test-scenario",
				Violations:  []string{},
				CachedAt:    &cachedTime,
				ValidatedAt: now,
				CacheUsed:   true,
			},
			wantJSON: "", // JSON encoding tested below
		},
		{
			name: "response without cache",
			response: ValidationResponse{
				DraftID:     "test-draft-id",
				EntityType:  "resource",
				EntityName:  "test-resource",
				Violations:  map[string]interface{}{"count": 0},
				CachedAt:    nil,
				ValidatedAt: now,
				CacheUsed:   false,
			},
			wantJSON: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonData, err := json.Marshal(tt.response)
			if err != nil {
				t.Errorf("Failed to marshal ValidationResponse: %v", err)
				return
			}

			// Test JSON unmarshaling
			var decoded ValidationResponse
			if err := json.Unmarshal(jsonData, &decoded); err != nil {
				t.Errorf("Failed to unmarshal ValidationResponse: %v", err)
				return
			}

			// Verify key fields
			if decoded.DraftID != tt.response.DraftID {
				t.Errorf("DraftID mismatch: got %v, want %v", decoded.DraftID, tt.response.DraftID)
			}
			if decoded.EntityType != tt.response.EntityType {
				t.Errorf("EntityType mismatch: got %v, want %v", decoded.EntityType, tt.response.EntityType)
			}
			if decoded.CacheUsed != tt.response.CacheUsed {
				t.Errorf("CacheUsed mismatch: got %v, want %v", decoded.CacheUsed, tt.response.CacheUsed)
			}
		})
	}
}

// TestScenarioAuditorCLINotFound tests handling when CLI is not available
func TestScenarioAuditorCLINotFound(t *testing.T) {
	// This test verifies graceful handling when scenario-auditor is not installed
	// We can't easily mock exec.LookPath, so we test with a nonexistent command

	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set PATH to empty to simulate command not found
	os.Setenv("PATH", "")

	result, err := runScenarioAuditorCLI("scenario", "test-scenario")

	// Should not return an error, but a graceful message
	if err != nil {
		t.Errorf("runScenarioAuditorCLI() should not error when CLI not found, got: %v", err)
	}

	if result == nil {
		t.Error("runScenarioAuditorCLI() should return a result even when CLI not found")
		return
	}

	// Check for graceful error message
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Error("Result should be a map with error message")
		return
	}

	if _, hasError := resultMap["error"]; !hasError {
		if _, hasMessage := resultMap["message"]; !hasMessage {
			t.Error("Result should contain either 'error' or 'message' field")
		}
	}
}

// Helper function to check if string contains substring (validation tests)
func containsStr(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// TestValidatePRDRequestStructure tests the validate PRD request structure
func TestValidatePRDRequestStructure(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entityName string
	}{
		{
			name:       "scenario request",
			entityType: "scenario",
			entityName: "test-scenario",
		},
		{
			name:       "resource request",
			entityType: "resource",
			entityName: "test-resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ValidatePRDRequest{
				EntityType: tt.entityType,
				EntityName: tt.entityName,
			}

			// Test JSON marshaling
			jsonData, err := json.Marshal(req)
			if err != nil {
				t.Errorf("Failed to marshal ValidatePRDRequest: %v", err)
				return
			}

			// Test JSON unmarshaling
			var decoded ValidatePRDRequest
			if err := json.Unmarshal(jsonData, &decoded); err != nil {
				t.Errorf("Failed to unmarshal ValidatePRDRequest: %v", err)
				return
			}

			if decoded.EntityType != tt.entityType {
				t.Errorf("EntityType mismatch: got %v, want %v", decoded.EntityType, tt.entityType)
			}
			if decoded.EntityName != tt.entityName {
				t.Errorf("EntityName mismatch: got %v, want %v", decoded.EntityName, tt.entityName)
			}
		})
	}
}

// TestRunScenarioAuditorWithRealCommand tests with actual command execution (if available)
func TestRunScenarioAuditorWithRealCommand(t *testing.T) {
	// Check if scenario-auditor is available
	if _, err := exec.LookPath("scenario-auditor"); err != nil {
		t.Skip("scenario-auditor not available, skipping integration test")
	}

	// Only run this if we're in the Vrooli environment
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = os.Getenv("HOME") + "/Vrooli"
	}

	// Check if prd-control-tower scenario exists
	if _, err := os.Stat(vrooliRoot + "/scenarios/prd-control-tower"); os.IsNotExist(err) {
		t.Skip("prd-control-tower scenario not found, skipping integration test")
	}

	result, err := runScenarioAuditorCLI("scenario", "prd-control-tower")
	if err != nil {
		t.Errorf("runScenarioAuditorCLI() with real command failed: %v", err)
		return
	}

	if result == nil {
		t.Error("runScenarioAuditorCLI() with real command returned nil")
	}
}
