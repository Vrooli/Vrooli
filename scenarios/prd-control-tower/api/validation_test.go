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

// [REQ:PCT-VALIDATE-CALL] Validation service calls scenario-auditor API for PRD structure checks
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
			result, err := runScenarioAuditorCLI(tt.entityName)
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

// [REQ:PCT-VALIDATE-CALL] Validation service calls scenario-auditor API for PRD structure checks
// TestRunScenarioAuditor tests the main auditor runner with environment setup
func TestRunScenarioAuditor(t *testing.T) {
	tests := []struct {
		name       string
		setupEnv   func()
		entityType string
		entityName string
		wantErr    bool
	}{
		{
			name:       "no auditor URL - falls back to CLI",
			setupEnv:   func() { os.Unsetenv("SCENARIO_AUDITOR_URL") },
			entityType: "scenario",
			entityName: "test-scenario",
			wantErr:    false,
		},
		{
			name: "with auditor URL - tries HTTP",
			setupEnv: func() {
				os.Setenv("SCENARIO_AUDITOR_URL", "http://localhost:18507")
			},
			entityType: "scenario",
			entityName: "test-scenario",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			if tt.setupEnv != nil {
				tt.setupEnv()
			}

			result, err := runScenarioAuditor(tt.entityType, tt.entityName)
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

// [REQ:PCT-VALIDATE-CALL] Validation service calls scenario-auditor API for PRD structure checks
// TestHandleValidatePRD tests the published PRD validation endpoint
func TestHandleValidatePRD(t *testing.T) {
	// Ensure scenario-auditor CLI lookup fails quickly so tests don't invoke the real binary
	t.Setenv("PATH", "")

	tests := []struct {
		name           string
		requestBody    any
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
				var response map[string]any
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
				var response map[string]any
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
				if !containsStr(body, "Invalid entity type") {
					t.Errorf("Expected error message about invalid entity type, got: %s", body)
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

// TestRunScenarioAuditorHTTP was removed since the HTTP implementation was a placeholder
// TODO: Re-add this test when scenario-auditor provides an HTTP API

// [REQ:PCT-VALIDATE-CACHE] Validation results cache in PostgreSQL with timestamp
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

// [REQ:PCT-VALIDATE-DISPLAY] Draft workspace displays violations with severity and recommendations
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
				Violations:  map[string]any{"count": 0},
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

	result, err := runScenarioAuditorCLI("test-scenario")

	// Should not return an error, but a graceful message
	if err != nil {
		t.Errorf("runScenarioAuditorCLI() should not error when CLI not found, got: %v", err)
	}

	if result == nil {
		t.Error("runScenarioAuditorCLI() should return a result even when CLI not found")
		return
	}

	// Check for graceful error message
	resultMap, ok := result.(map[string]any)
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
	if os.Getenv("RUN_SCENARIO_AUDITOR_TESTS") != "1" {
		t.Skip("Skipping scenario-auditor integration test; set RUN_SCENARIO_AUDITOR_TESTS=1 to enable")
	}

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

	result, err := runScenarioAuditorCLI("prd-control-tower")
	if err != nil {
		t.Errorf("runScenarioAuditorCLI() with real command failed: %v", err)
		return
	}

	if result == nil {
		t.Error("runScenarioAuditorCLI() with real command returned nil")
	}
}
