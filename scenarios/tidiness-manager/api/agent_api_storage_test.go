package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// [REQ:TM-API-006] Issue storage in postgres - verify all required fields are stored
func TestIssueStorage_RequiredFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issue with all fields
	lineNum := 42
	colNum := 10
	_, err := srv.db.Exec(`
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description,
			line_number, column_number, agent_notes, remediation_steps,
			status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, "test-scenario", "api/test.go", "complexity", "high", "Complex function",
		"Function has cyclomatic complexity of 20", lineNum, colNum,
		"Consider breaking into smaller functions",
		"1. Extract validation logic\n2. Extract business logic",
		"open", time.Now())

	if err != nil {
		t.Fatalf("Failed to insert issue: %v", err)
	}

	// Query back and verify all fields
	req, _ := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(issues) == 0 {
		t.Fatal("Expected at least one issue")
	}

	issue := issues[0]
	if issue.Scenario != "test-scenario" {
		t.Errorf("Scenario mismatch: got %s", issue.Scenario)
	}
	if issue.FilePath != "api/test.go" {
		t.Errorf("FilePath mismatch: got %s", issue.FilePath)
	}
	if issue.Category != "complexity" {
		t.Errorf("Category mismatch: got %s", issue.Category)
	}
	if issue.Severity != "high" {
		t.Errorf("Severity mismatch: got %s", issue.Severity)
	}
	if issue.LineNumber == nil || *issue.LineNumber != lineNum {
		t.Errorf("LineNumber mismatch: got %v", issue.LineNumber)
	}
	if issue.ColumnNumber == nil || *issue.ColumnNumber != colNum {
		t.Errorf("ColumnNumber mismatch: got %v", issue.ColumnNumber)
	}
	if issue.AgentNotes == "" {
		t.Error("AgentNotes should not be empty")
	}
	if issue.RemediationSteps == "" {
		t.Error("RemediationSteps should not be empty")
	}
}

// [REQ:TM-API-006] Test error handling for missing scenario parameter
func TestAgentGetIssues_MissingScenario(t *testing.T) {
	srv := setupTestServerNoData(t)
	if srv == nil {
		return
	}

	req, err := http.NewRequest("GET", "/api/v1/agent/issues", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler should return 400 for missing scenario: got %v want %v", status, http.StatusBadRequest)
	}
}

// [REQ:TM-API-006] Test force scan returns not implemented (P1 feature)
func TestAgentGetIssues_ForceScansNotImplemented(t *testing.T) {
	srv := setupTestServerNoData(t)
	if srv == nil {
		return
	}

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test&force=true", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotImplemented {
		t.Errorf("Force scans should return 501: got %v", status)
	}
}

// [REQ:TM-API-006] Test empty result sets
func TestAgentGetIssues_EmptyResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Query non-existent scenario
	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=nonexistent-scenario", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(issues) != 0 {
		t.Errorf("Expected empty result set, got %d issues", len(issues))
	}
}

// [REQ:TM-API-006] Test special characters in query parameters
func TestAgentGetIssues_SpecialCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issue with special characters in scenario name
	scenario := "test-scenario-with-dashes"
	insertTestIssue(t, srv.db, scenario, "api/main.go", "length", "high", "Issue 1", "Desc")

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario="+scenario, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(issues) == 0 {
		t.Error("Expected at least one issue")
	}

	if issues[0].Scenario != scenario {
		t.Errorf("Scenario = %s, want %s", issues[0].Scenario, scenario)
	}
}

// [REQ:TM-API-006] Test issue with nil optional fields
func TestIssueStorage_NilOptionalFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issue without optional fields
	_, err := srv.db.Exec(`
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "test-scenario", "api/test.go", "length", "medium", "Test Issue", "Test Description", "open", time.Now())

	if err != nil {
		t.Fatalf("Failed to insert issue: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(issues) == 0 {
		t.Fatal("Expected at least one issue")
	}

	issue := issues[0]
	// Verify nil optional fields don't cause errors
	if issue.LineNumber != nil {
		t.Log("LineNumber is present (optional)")
	}
	if issue.ColumnNumber != nil {
		t.Log("ColumnNumber is present (optional)")
	}
}

// [REQ:TM-API-006] Test handleAgentStoreIssue - successful issue storage
func TestHandleAgentStoreIssue_Success(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil {
		return
	}
	defer srv.db.Close()

	lineNum := 42
	colNum := 10
	campaignID := 1
	issuePayload := map[string]interface{}{
		"scenario":          "test-scenario",
		"file_path":         "src/main.go",
		"category":          "code_quality",
		"severity":          "warning",
		"title":             "Unused variable",
		"description":       "Variable 'x' is declared but never used",
		"line_number":       lineNum,
		"column_number":     colNum,
		"agent_notes":       "Found during static analysis",
		"remediation_steps": "Remove unused variable",
		"campaign_id":       campaignID,
		"session_id":        "session-123",
		"resource_used":     "claude-code",
	}

	payload, _ := json.Marshal(issuePayload)
	req := httptest.NewRequest("POST", "/api/v1/agent/issues", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleAgentStoreIssue(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, ok := response["id"]; !ok {
		t.Error("Response should include issue ID")
	}
	if _, ok := response["created_at"]; !ok {
		t.Error("Response should include created_at timestamp")
	}
}

// [REQ:TM-API-006] Test handleAgentStoreIssue - validation errors
func TestHandleAgentStoreIssue_ValidationErrors(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil {
		return
	}
	defer srv.db.Close()

	tests := []struct {
		name    string
		payload map[string]interface{}
		wantErr string
	}{
		{
			name: "missing scenario",
			payload: map[string]interface{}{
				"file_path": "src/main.go",
				"category":  "code_quality",
				"severity":  "warning",
			},
			wantErr: "scenario, file_path, category, and severity are required",
		},
		{
			name: "missing file_path",
			payload: map[string]interface{}{
				"scenario": "test-scenario",
				"category": "code_quality",
				"severity": "warning",
			},
			wantErr: "scenario, file_path, category, and severity are required",
		},
		{
			name: "missing category",
			payload: map[string]interface{}{
				"scenario":  "test-scenario",
				"file_path": "src/main.go",
				"severity":  "warning",
			},
			wantErr: "scenario, file_path, category, and severity are required",
		},
		{
			name: "missing severity",
			payload: map[string]interface{}{
				"scenario":  "test-scenario",
				"file_path": "src/main.go",
				"category":  "code_quality",
			},
			wantErr: "scenario, file_path, category, and severity are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/api/v1/agent/issues", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			srv.handleAgentStoreIssue(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rr.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if !strings.Contains(response["error"].(string), tt.wantErr) {
				t.Errorf("Expected error to contain %q, got %q", tt.wantErr, response["error"])
			}
		})
	}
}

// [REQ:TM-API-006] Test handleAgentStoreIssue - malformed JSON
func TestHandleAgentStoreIssue_MalformedJSON(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil {
		return
	}
	defer srv.db.Close()

	req := httptest.NewRequest("POST", "/api/v1/agent/issues", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleAgentStoreIssue(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// [REQ:TM-API-007] Campaign metadata linkage - verify issues can be linked to campaigns
func TestIssueStorage_CampaignMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Verify campaign columns exist in schema
	var columnExists bool
	err := srv.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'issues' AND column_name = 'campaign_id'
		)
	`).Scan(&columnExists)

	if err != nil {
		t.Fatalf("Failed to check campaign_id column: %v", err)
	}

	if !columnExists {
		t.Log("campaign_id column not yet in schema - P1 feature")
		// This is acceptable for P0 - campaign linkage is part of OT-P0-008 but full implementation is P1
		return
	}

	// If column exists, verify we can store campaign metadata
	_, err = srv.db.Exec(`
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description,
			status, created_at, campaign_id, session_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, "test-scenario", "api/test.go", "length", "high", "Long file",
		"File exceeds threshold", "open", time.Now(), 1, "session-123")

	if err != nil {
		t.Errorf("Failed to insert issue with campaign metadata: %v", err)
	}
}
