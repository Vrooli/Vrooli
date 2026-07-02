package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/assessment"
)

// Test helpers for handler testing

// marshalBodyOrString marshals body to JSON bytes, or converts string directly
func marshalBodyOrString(t *testing.T, body interface{}) []byte {
	t.Helper()
	if str, ok := body.(string); ok {
		return []byte(str)
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}
	return bodyBytes
}

// testHandlerRequest executes a handler with a test request and returns the response recorder
func testHandlerRequest(t *testing.T, method, url string, body interface{}, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	bodyBytes := marshalBodyOrString(t, body)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// assertHandlerStatus checks if handler returned expected status code
func assertHandlerStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int, handlerName string) {
	t.Helper()
	if rr.Code != expected {
		t.Errorf("%s status = %v, want %v", handlerName, rr.Code, expected)
	}
}

// assertHandlerError decodes error response and checks message
func assertHandlerError(t *testing.T, rr *httptest.ResponseRecorder, wantError, handlerName string) {
	t.Helper()
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if response["error"] != wantError {
		t.Errorf("%s error = %v, want %v", handlerName, response["error"], wantError)
	}
}

// assertParseResponseFields validates parse endpoint response has required fields
func assertParseResponseFields(t *testing.T, rr *httptest.ResponseRecorder, handlerName string) {
	t.Helper()
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode success response: %v", err)
	}
	if _, ok := response["issues"]; !ok {
		t.Errorf("%s response missing 'issues' field", handlerName)
	}
	if _, ok := response["count"]; !ok {
		t.Errorf("%s response missing 'count' field", handlerName)
	}
}

// Test respondJSON helper function
func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "success response",
			status:     http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "created response",
			status:     http.StatusCreated,
			data:       map[string]int{"id": 123},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			respondJSON(rr, tt.status, tt.data)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("respondJSON() status = %v, want %v", status, tt.wantStatus)
			}

			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("respondJSON() Content-Type = %v, want application/json", ct)
			}
		})
	}
}

// Test respondError helper function
func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
	}{
		{
			name:       "bad request error",
			status:     http.StatusBadRequest,
			message:    "invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal server error",
			status:     http.StatusInternalServerError,
			message:    "something went wrong",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			respondError(rr, tt.status, tt.message)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("respondError() status = %v, want %v", status, tt.wantStatus)
			}

			var response map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("respondError() message = %v, want %v", response["error"], tt.message)
			}
		})
	}
}

func TestBuildTidinessScanReportsMaintainabilityOnly(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}

	var content strings.Builder
	content.WriteString("package main\n\n")
	content.WriteString("import \"fmt\"\n\n")
	content.WriteString("func main() {\n")
	for i := 0; i < 12; i++ {
		content.WriteString("\t// TODO: planned cleanup\n")
	}
	content.WriteString("\tfmt.Println(\"ok\")\n")
	content.WriteString("}\n")
	for i := 0; i < 520; i++ {
		content.WriteString("// filler line\n")
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(content.String()), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	result, err := buildTidinessScan(context.Background(), "demo", tmpDir, 0)
	if err != nil {
		t.Fatalf("buildTidinessScan failed: %v", err)
	}

	if result.Summary.LongFiles == 0 {
		t.Fatalf("expected long-file finding, got summary %+v", result.Summary)
	}
	if result.Summary.TechDebt == 0 {
		t.Fatalf("expected tech-debt finding, got summary %+v", result.Summary)
	}
	if err := assessment.ValidateAssessment(result.Assessment); err != nil {
		t.Fatalf("assessment invalid: %v", err)
	}
	if result.Assessment.GetProvider() != "tidiness-manager" || result.Assessment.GetPhase() != "tidiness" {
		t.Fatalf("assessment identity = %s/%s, want tidiness-manager/tidiness", result.Assessment.GetProvider(), result.Assessment.GetPhase())
	}
	for _, finding := range result.Findings {
		if finding.Category == "type_safety" || strings.Contains(finding.RuleID, "TS_CONFIG") {
			t.Fatalf("tidiness scan must not emit static-quality finding: %+v", finding)
		}
		if finding.WhyItMatters == "" || finding.RecommendedRemediation == "" {
			t.Fatalf("finding missing guidance fields: %+v", finding)
		}
	}
}

func TestTidinessMaturitySpecCoversEmittedRules(t *testing.T) {
	spec, err := loadTidinessMaturitySpec()
	if err != nil {
		t.Fatalf("loadTidinessMaturitySpec() error = %v", err)
	}
	if spec.Version != "2.0.0" {
		t.Fatalf("version = %q, want 2.0.0", spec.Version)
	}
	if len(spec.Capabilities) != 4 {
		t.Fatalf("capabilities = %d, want 4", len(spec.Capabilities))
	}
	for _, ruleID := range emittedTidinessRuleIDs() {
		mapping, ok := spec.Findings[ruleID]
		if !ok {
			t.Fatalf("maturity spec missing emitted rule %q", ruleID)
		}
		if mapping.CapabilityID == "" {
			t.Fatalf("maturity spec finding %q must declare capability_id", ruleID)
		}
	}
	if spec.Fallback.CapabilityID != "scan_contract" {
		t.Fatalf("fallback capability = %q, want scan_contract", spec.Fallback.CapabilityID)
	}

	got, err := buildTidinessMaturityAssessment("demo", []TidinessFinding{
		newTidinessFinding("demo", "long-file", "length", "medium", "api/server.go", "", 1, "large file", "too large", nil, "why", "split it", "file-size"),
	}, spec)
	if err != nil {
		t.Fatalf("buildTidinessMaturityAssessment() error = %v", err)
	}
	if len(got.GetCapabilities()) != 4 {
		t.Fatalf("capabilities = %d, want 4", len(got.GetCapabilities()))
	}
	if got.GetFindings()[0].GetMaturity().GetCapabilityId() != "local_debt_control" {
		t.Fatalf("long-file capability = %q, want local_debt_control", got.GetFindings()[0].GetMaturity().GetCapabilityId())
	}
	if got.GetHighestPriorityCapability().GetCapabilityId() != "local_debt_control" {
		t.Fatalf("warning-only focus = %#v, want local_debt_control", got.GetHighestPriorityCapability())
	}

	blocking, err := buildTidinessMaturityAssessment("demo", []TidinessFinding{
		newTidinessFinding("demo", "duplicated-code", "duplication", "high", "api/server.go", "", 1, "duplicated code", "duplicate", nil, "why", "extract", "duplication"),
	}, spec)
	if err != nil {
		t.Fatalf("buildTidinessMaturityAssessment(blocking) error = %v", err)
	}
	if blocking.GetFindings()[0].GetMaturity().GetCapabilityId() != "duplication_control" {
		t.Fatalf("duplicated-code capability = %q, want duplication_control", blocking.GetFindings()[0].GetMaturity().GetCapabilityId())
	}
	if blocking.GetHighestPriorityCapability().GetCapabilityId() != "duplication_control" {
		t.Fatalf("blocking focus = %#v, want duplication_control", blocking.GetHighestPriorityCapability())
	}
	if blocking.GetLocal().GetCurrentLevel() != "L1" {
		t.Fatalf("blocking assessment current level = %q, want L1", blocking.GetLocal().GetCurrentLevel())
	}
}

func emittedTidinessRuleIDs() []string {
	return []string{
		"LONG_FILE",
		"TECH_DEBT_MARKERS",
		"HIGH_COUPLING",
		"HIGH_COMPLEXITY",
		"DUPLICATED_CODE",
	}
}

// Test handleParseLint
func TestHandleParseLint(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name: "valid lint parse request",
			body: ParseRequest{
				Scenario: "test-scenario",
				Tool:     "eslint",
				Output:   "src/main.ts:10:5: error: Unexpected token",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing scenario",
			body:       ParseRequest{Tool: "eslint", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "missing tool",
			body:       ParseRequest{Scenario: "test", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/parse/lint", tt.body, srv.handleParseLint)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleParseLint()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleParseLint()")
			} else {
				assertParseResponseFields(t, rr, "handleParseLint()")
			}
		})
	}
}

// Test handleParseType
func TestHandleParseType(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name: "valid type parse request",
			body: ParseRequest{
				Scenario: "test-scenario",
				Tool:     "tsc",
				Output:   "src/main.ts(10,5): error TS2304: Cannot find name 'foo'",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing scenario",
			body:       ParseRequest{Tool: "tsc", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "missing tool",
			body:       ParseRequest{Scenario: "test", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/parse/type", tt.body, srv.handleParseType)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleParseType()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleParseType()")
			} else {
				assertParseResponseFields(t, rr, "handleParseType()")
			}
		})
	}
}

// Test handleLightScan
func TestHandleLightScan(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
		db:     nil, // No DB needed for basic tests
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing scenario_path",
			body:       LightScanRequest{},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario_path is required",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/scan/light", tt.body, srv.handleLightScan)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleLightScan()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleLightScan()")
			}
		})
	}
}

// Test handleSmartScan
func TestHandleSmartScan(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing scenario",
			body:       SmartScanRequest{},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario is required",
		},
		{
			name:       "missing files",
			body:       SmartScanRequest{Scenario: "test-scenario"},
			wantStatus: http.StatusBadRequest,
			wantError:  "files list cannot be empty",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/scan/smart", tt.body, srv.handleSmartScan)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleSmartScan()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleSmartScan()")
			}
		})
	}
}
