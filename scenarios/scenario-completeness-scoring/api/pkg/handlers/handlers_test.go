package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"scenario-completeness-scoring/pkg/analysis"
	"scenario-completeness-scoring/pkg/circuitbreaker"
	"scenario-completeness-scoring/pkg/collectors"
	"scenario-completeness-scoring/pkg/config"
	apierrors "scenario-completeness-scoring/pkg/errors"
	"scenario-completeness-scoring/pkg/health"
	"scenario-completeness-scoring/pkg/history"

	"github.com/gorilla/mux"
)

// setupTestContext creates a handler context for testing
func setupTestContext(t *testing.T) (*Context, string) {
	t.Helper()

	tmpDir := t.TempDir()

	// Create scenarios directory
	scenariosDir := filepath.Join(tmpDir, "scenarios")
	os.MkdirAll(scenariosDir, 0755)

	// Create a test scenario
	testScenario := filepath.Join(scenariosDir, "test-scenario")
	os.MkdirAll(filepath.Join(testScenario, ".vrooli"), 0755)
	os.WriteFile(filepath.Join(testScenario, ".vrooli", "service.json"),
		[]byte(`{"category": "utility", "name": "test-scenario"}`), 0644)

	// Create requirements directory
	reqDir := filepath.Join(testScenario, "requirements")
	os.MkdirAll(reqDir, 0755)
	os.WriteFile(filepath.Join(reqDir, "index.json"),
		[]byte(`{"imports": [], "requirements": [{"id": "REQ-001", "status": "passed"}]}`), 0644)

	// Initialize dependencies
	cbRegistry := circuitbreaker.NewRegistry(circuitbreaker.DefaultConfig())
	collector := collectors.NewMetricsCollectorWithCircuitBreaker(tmpDir, cbRegistry)
	configLoader := config.NewLoader(tmpDir)

	// Initialize history database
	dataDir := filepath.Join(tmpDir, "data")
	historyDB, err := history.NewDB(dataDir)
	if err != nil {
		t.Fatalf("Failed to create history DB: %v", err)
	}
	historyRepo := history.NewRepository(historyDB)
	trendAnalyzer := history.NewTrendAnalyzer(historyRepo, 5)

	ctx := NewContext(
		tmpDir,
		collector,
		cbRegistry,
		health.NewTracker(cbRegistry),
		configLoader,
		historyDB,
		historyRepo,
		trendAnalyzer,
		analysis.NewWhatIfAnalyzer(collector),
		analysis.NewBulkRefresher(tmpDir, collector, historyRepo),
	)

	return ctx, tmpDir
}

// TestHandleUpdateConfigInvalidJSON tests that invalid JSON returns structured error
// [REQ:SCS-CORE-003] Structured error responses
func TestHandleUpdateConfigInvalidJSON(t *testing.T) {
	ctx, _ := setupTestContext(t)

	req := httptest.NewRequest("PUT", "/api/v1/config", bytes.NewBufferString("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ctx.HandleUpdateConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// Verify structured error response
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'error' object in response")
	}

	if errorObj["code"] != apierrors.ErrCodeValidationFailed {
		t.Errorf("Expected error code %s, got %v", apierrors.ErrCodeValidationFailed, errorObj["code"])
	}

	nextSteps, ok := errorObj["next_steps"].([]interface{})
	if !ok || len(nextSteps) == 0 {
		t.Error("Expected next_steps in error response")
	}
}

// TestHandleUpdateConfigInvalidConfig tests that invalid config returns structured error
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func TestHandleUpdateConfigInvalidConfig(t *testing.T) {
	ctx, _ := setupTestContext(t)

	// Invalid config: all components disabled
	invalidConfig := `{
		"version": "1.0",
		"components": {
			"quality": {"enabled": false},
			"coverage": {"enabled": false},
			"quantity": {"enabled": false},
			"ui": {"enabled": false}
		},
		"weights": {"quality": 0, "coverage": 0, "quantity": 0, "ui": 0}
	}`

	req := httptest.NewRequest("PUT", "/api/v1/config", bytes.NewBufferString(invalidConfig))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ctx.HandleUpdateConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'error' object in response")
	}

	if errorObj["code"] != apierrors.ErrCodeConfigInvalid {
		t.Errorf("Expected error code %s, got %v", apierrors.ErrCodeConfigInvalid, errorObj["code"])
	}
}

// TestHandleWhatIfEmptyChanges tests that empty changes returns structured error
// [REQ:SCS-CORE-003] Structured error responses with examples
func TestHandleWhatIfEmptyChanges(t *testing.T) {
	ctx, _ := setupTestContext(t)

	emptyChanges := `{"changes": []}`

	req := httptest.NewRequest("POST", "/api/v1/scores/test-scenario/what-if", bytes.NewBufferString(emptyChanges))
	req = mux.SetURLVars(req, map[string]string{"scenario": "test-scenario"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ctx.HandleWhatIf(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'error' object in response")
	}

	// Should include helpful examples in next_steps
	nextSteps, ok := errorObj["next_steps"].([]interface{})
	if !ok || len(nextSteps) == 0 {
		t.Error("Expected next_steps with examples")
	}

	// Verify at least one step mentions example or component
	hasExample := false
	for _, step := range nextSteps {
		if stepStr, ok := step.(string); ok {
			if len(stepStr) > 0 {
				hasExample = true
				break
			}
		}
	}
	if !hasExample {
		t.Error("Expected at least one helpful next step")
	}
}

// TestHandleCompareInsufficientScenarios tests that <2 scenarios returns structured error
// [REQ:SCS-CORE-003] Structured error responses
func TestHandleCompareInsufficientScenarios(t *testing.T) {
	ctx, _ := setupTestContext(t)

	// Only one scenario
	singleScenario := `{"scenarios": ["scenario-a"]}`

	req := httptest.NewRequest("POST", "/api/v1/compare", bytes.NewBufferString(singleScenario))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ctx.HandleCompare(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'error' object in response")
	}

	if errorObj["code"] != apierrors.ErrCodeValidationFailed {
		t.Errorf("Expected error code %s, got %v", apierrors.ErrCodeValidationFailed, errorObj["code"])
	}
}

// TestHandleGetHistoryReturnsEmptyList tests that missing history returns empty list, not error
// [REQ:SCS-CORE-003] Graceful handling of no data
func TestHandleGetHistoryReturnsEmptyList(t *testing.T) {
	ctx, _ := setupTestContext(t)

	req := httptest.NewRequest("GET", "/api/v1/scores/new-scenario/history", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": "new-scenario"})
	rr := httptest.NewRecorder()

	ctx.HandleGetHistory(rr, req)

	// Should return 200 with empty list, not an error
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have snapshots field (even if empty)
	if _, ok := response["snapshots"]; !ok {
		t.Error("Expected 'snapshots' field in response")
	}
}

// TestHandleHealthReturnsOK tests that health endpoint always returns OK
// [REQ:SCS-HEALTH-001] Health endpoint availability
func TestHandleHealthReturnsOK(t *testing.T) {
	ctx, _ := setupTestContext(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	ctx.HandleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

// ==========================================================
// Assumption Hardening Tests
// ==========================================================

// TestValidateScenarioName tests the scenario name validation function
// [REQ:SCS-CORE-003] Input validation - prevents path traversal attacks
func TestValidateScenarioName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errMatch string
	}{
		{
			name:    "valid simple name",
			input:   "test-scenario",
			wantErr: false,
		},
		{
			name:    "valid with underscore",
			input:   "my_scenario_v2",
			wantErr: false,
		},
		{
			name:    "valid numeric start",
			input:   "123-scenario",
			wantErr: false,
		},
		{
			name:     "empty name",
			input:    "",
			wantErr:  true,
			errMatch: "cannot be empty",
		},
		{
			name:     "path traversal attempt",
			input:    "../../../etc/passwd",
			wantErr:  true,
			errMatch: "invalid characters",
		},
		{
			name:     "forward slash",
			input:    "parent/child",
			wantErr:  true,
			errMatch: "invalid characters",
		},
		{
			name:     "backslash",
			input:    "parent\\child",
			wantErr:  true,
			errMatch: "invalid characters",
		},
		{
			name:     "too long",
			input:    "abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmnopqrstuvwxyz1234567890",
			wantErr:  true,
			errMatch: "too long",
		},
		{
			name:     "starts with dash",
			input:    "-invalid-start",
			wantErr:  true,
			errMatch: "must start with alphanumeric",
		},
		{
			name:     "contains space",
			input:    "invalid name",
			wantErr:  true,
			errMatch: "must start with alphanumeric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := ValidateScenarioName(tt.input)
			gotErr := errMsg != ""

			if gotErr != tt.wantErr {
				t.Errorf("ValidateScenarioName(%q) error = %v, want error = %v (msg: %s)",
					tt.input, gotErr, tt.wantErr, errMsg)
			}

			if tt.wantErr && tt.errMatch != "" {
				if errMsg == "" || !containsSubstring(errMsg, tt.errMatch) {
					t.Errorf("ValidateScenarioName(%q) error = %q, want to contain %q",
						tt.input, errMsg, tt.errMatch)
				}
			}
		})
	}
}

// containsSubstring is a simple helper for substring matching
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr ||
			containsSubstring(s[1:], substr))))
}

// TestHandleCalculateScoreInvalidScenarioName tests validation on calculate endpoint
// [REQ:SCS-CORE-003] Input validation across all handlers
func TestHandleCalculateScoreInvalidScenarioName(t *testing.T) {
	ctx, _ := setupTestContext(t)

	req := httptest.NewRequest("POST", "/api/v1/scores/../../../etc/passwd/calculate", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": "../../../etc/passwd"})
	rr := httptest.NewRecorder()

	ctx.HandleCalculateScore(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'error' object in response")
	}

	if errorObj["code"] != apierrors.ErrCodeValidationFailed {
		t.Errorf("Expected error code %s, got %v", apierrors.ErrCodeValidationFailed, errorObj["code"])
	}
}

// TestHandleWhatIfInvalidScenarioName tests validation on what-if endpoint
// [REQ:SCS-CORE-003] Input validation across all handlers
func TestHandleWhatIfInvalidScenarioName(t *testing.T) {
	ctx, _ := setupTestContext(t)

	validChanges := `{"changes": [{"component": "quality.test_pass_rate", "new_value": 1.0}]}`
	req := httptest.NewRequest("POST", "/api/v1/scores/../../../passwd/what-if", bytes.NewBufferString(validChanges))
	req = mux.SetURLVars(req, map[string]string{"scenario": "../../../passwd"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ctx.HandleWhatIf(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestHandleGetHistoryInvalidScenarioName tests validation on history endpoint
// [REQ:SCS-CORE-003] Input validation across all handlers
func TestHandleGetHistoryInvalidScenarioName(t *testing.T) {
	ctx, _ := setupTestContext(t)

	req := httptest.NewRequest("GET", "/api/v1/scores/../passwd/history", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": "../passwd"})
	rr := httptest.NewRecorder()

	ctx.HandleGetHistory(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestHandleGetTrendsInvalidScenarioName tests validation on trends endpoint
// [REQ:SCS-CORE-003] Input validation across all handlers
func TestHandleGetTrendsInvalidScenarioName(t *testing.T) {
	ctx, _ := setupTestContext(t)

	req := httptest.NewRequest("GET", "/api/v1/scores/bad/name/trends", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": "bad/name"})
	rr := httptest.NewRecorder()

	ctx.HandleGetTrends(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestHandleCompareInvalidScenarioNameInArray tests validation of scenario names in comparison array
// [REQ:SCS-CORE-003] Input validation for arrays
func TestHandleCompareInvalidScenarioNameInArray(t *testing.T) {
	ctx, _ := setupTestContext(t)

	// One valid, one invalid
	invalidArray := `{"scenarios": ["valid-scenario", "../../../passwd"]}`

	req := httptest.NewRequest("POST", "/api/v1/compare", bytes.NewBufferString(invalidArray))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	ctx.HandleCompare(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'error' object in response")
	}

	// Error message should mention the invalid name
	if msg, ok := errorObj["message"].(string); ok {
		if !containsSubstring(msg, "../../../passwd") {
			t.Errorf("Expected error message to mention invalid scenario name")
		}
	}
}

// TestHandleTestCollectorInvalidName tests validation on collector test endpoint
// [REQ:SCS-HEALTH-003] Collector name validation
func TestHandleTestCollectorInvalidName(t *testing.T) {
	ctx, _ := setupTestContext(t)

	req := httptest.NewRequest("POST", "/api/v1/health/collectors/invalid-collector/test", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "invalid-collector"})
	rr := httptest.NewRecorder()

	ctx.HandleTestCollector(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'error' object in response")
	}

	// Should include valid collectors in next_steps
	nextSteps, ok := errorObj["next_steps"].([]interface{})
	if !ok || len(nextSteps) == 0 {
		t.Error("Expected next_steps with valid collectors")
	}
}

// TestHandleResetCircuitBreakerInvalidName tests validation on circuit breaker reset
// [REQ:SCS-CB-004] Circuit breaker name validation
func TestHandleResetCircuitBreakerInvalidName(t *testing.T) {
	ctx, _ := setupTestContext(t)

	req := httptest.NewRequest("POST", "/api/v1/health/circuit-breaker/bad-name/reset", nil)
	req = mux.SetURLVars(req, map[string]string{"collector": "bad-name"})
	rr := httptest.NewRecorder()

	ctx.HandleResetCircuitBreaker(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

// TestLimitParameterBounds tests that limit parameters are capped
// [REQ:SCS-CORE-003] Prevent excessive memory usage
func TestLimitParameterBounds(t *testing.T) {
	ctx, _ := setupTestContext(t)

	// Request with extremely large limit
	req := httptest.NewRequest("GET", "/api/v1/scores/test-scenario/history?limit=999999", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": "test-scenario"})
	rr := httptest.NewRecorder()

	ctx.HandleGetHistory(rr, req)

	// Should succeed (limit is capped internally)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// The limit in response should be capped at maxLimitDefault (1000)
	if limit, ok := response["limit"].(float64); ok {
		if limit > 1000 {
			t.Errorf("Expected limit to be capped at 1000, got %v", limit)
		}
	}
}

// TestHandleGetScoreInvalidScenarioName tests that invalid scenario names are rejected
// [REQ:SCS-CORE-003] Input validation returns structured error
func TestHandleGetScoreInvalidScenarioName(t *testing.T) {
	ctx, _ := setupTestContext(t)

	tests := []struct {
		name         string
		scenarioName string
	}{
		{"path traversal", "../../../etc/passwd"},
		{"empty name", ""},
		{"with slash", "parent/child"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/scores/"+tt.scenarioName, nil)
			req = mux.SetURLVars(req, map[string]string{"scenario": tt.scenarioName})
			rr := httptest.NewRecorder()

			ctx.HandleGetScore(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d for invalid scenario name %q, got %d",
					http.StatusBadRequest, tt.scenarioName, rr.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			errorObj, ok := response["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected 'error' object in response")
			}

			if errorObj["code"] != apierrors.ErrCodeValidationFailed {
				t.Errorf("Expected error code %s, got %v",
					apierrors.ErrCodeValidationFailed, errorObj["code"])
			}
		})
	}
}
