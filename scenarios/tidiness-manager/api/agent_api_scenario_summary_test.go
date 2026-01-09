package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:TM-API-001] Scenario summary endpoint - Test ability to request top N issues for a scenario
func TestAgentGetIssues_BasicScenarioQuery(t *testing.T) {
	srv := setupTestServerOrSkip(t)
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "File too long", "File exceeds 500 lines")

	rr := makeRequest(t, srv.router, "GET", "/api/v1/agent/issues?scenario=test-scenario&limit=10", nil)
	assertStatusCode(t, rr, http.StatusOK)

	var issues []AgentIssue
	unmarshalResponse(t, rr, &issues)

	if len(issues) == 0 {
		t.Error("Expected at least one issue, got none")
	}

	if len(issues) > 0 && issues[0].Scenario != "test-scenario" {
		t.Errorf("Expected scenario 'test-scenario', got '%s'", issues[0].Scenario)
	}
}

// [REQ:TM-API-001] Test limit parameter
func TestAgentGetIssues_LimitParameter(t *testing.T) {
	srv := setupTestServerOrSkip(t)

	// Insert multiple test issues
	for i := 1; i <= 5; i++ {
		insertTestIssue(t, srv.db, "test-scenario", "api/file.go", "length", "high", "Issue", "Description")
	}

	rr := makeRequest(t, srv.router, "GET", "/api/v1/agent/issues?scenario=test-scenario&limit=2", nil)

	var issues []AgentIssue
	unmarshalResponse(t, rr, &issues)

	if len(issues) > 2 {
		t.Errorf("Expected max 2 issues, got %d", len(issues))
	}
}

// [REQ:TM-API-001] Test invalid limit parameter
func TestAgentGetIssues_InvalidLimit(t *testing.T) {
	srv := setupTestServerNoData(t)
	if srv == nil {
		return
	}

	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{
			name:       "negative limit",
			url:        "/api/v1/agent/issues?scenario=test&limit=-5",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero limit",
			url:        "/api/v1/agent/issues?scenario=test&limit=0",
			wantStatus: http.StatusOK, // Might return empty array
		},
		{
			name:       "non-numeric limit",
			url:        "/api/v1/agent/issues?scenario=test&limit=abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "excessively large limit",
			url:        "/api/v1/agent/issues?scenario=test&limit=999999",
			wantStatus: http.StatusOK, // Should be clamped or allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			srv.router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

// [REQ:TM-API-001] Test concurrent requests
func TestAgentGetIssues_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "Issue 1", "Desc 1")

	const numRequests = 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario", nil)
			rr := httptest.NewRecorder()
			srv.router.ServeHTTP(rr, req)
			results <- rr.Code
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		status := <-results
		if status != http.StatusOK {
			t.Errorf("Request %d failed with status %d", i, status)
		}
	}
}

// [REQ:TM-API-001] Test handleAgentGetScenarios - list all scenarios
func TestHandleAgentGetScenarios(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil {
		return
	}
	defer srv.db.Close()

	// Insert test scenarios with issues
	_, err := srv.db.Exec(`
		INSERT INTO issues (scenario, file_path, category, severity, status, title)
		VALUES
			('scenario-a', 'file1.go', 'lint', 'error', 'open', 'Issue 1'),
			('scenario-a', 'file2.go', 'lint', 'warning', 'open', 'Issue 2'),
			('scenario-b', 'file3.go', 'type', 'error', 'resolved', 'Issue 3')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/agent/scenarios", nil)
	rr := httptest.NewRecorder()

	srv.handleAgentGetScenarios(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	scenarios, ok := response["scenarios"].([]interface{})
	if !ok {
		t.Fatal("Response should include scenarios array")
	}

	if len(scenarios) < 2 {
		t.Errorf("Expected at least 2 scenarios, got %d", len(scenarios))
	}
}

// [REQ:TM-API-001] Test handleAgentGetScenarioDetail - get specific scenario
func TestHandleAgentGetScenarioDetail(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil {
		return
	}
	defer srv.db.Close()

	// Insert test data
	_, err := srv.db.Exec(`
		INSERT INTO issues (scenario, file_path, category, severity, status, title)
		VALUES ('test-scenario', 'file1.go', 'lint', 'error', 'open', 'Test Issue')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/agent/scenarios/test-scenario", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": "test-scenario"})
	rr := httptest.NewRecorder()

	srv.handleAgentGetScenarioDetail(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["scenario"] != "test-scenario" {
		t.Errorf("Expected scenario 'test-scenario', got %v", response["scenario"])
	}

	// Check for expected fields
	if _, ok := response["lightIssues"]; !ok {
		t.Error("Response should include lightIssues count")
	}
	if _, ok := response["aiIssues"]; !ok {
		t.Error("Response should include aiIssues count")
	}
	if _, ok := response["longFiles"]; !ok {
		t.Error("Response should include longFiles count")
	}
	if _, ok := response["files"]; !ok {
		t.Error("Response should include files array")
	}
}

// [REQ:TM-API-001] Test handleAgentGetScenarioDetail - missing scenario parameter
func TestHandleAgentGetScenarioDetail_MissingScenario(t *testing.T) {
	srv := setupTestServer(t)
	if srv == nil {
		return
	}
	defer srv.db.Close()

	req := httptest.NewRequest("GET", "/api/v1/agent/scenarios/", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario": ""})
	rr := httptest.NewRecorder()

	srv.handleAgentGetScenarioDetail(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// [REQ:TM-API-001] Test parseAgentIssuesRequest with various inputs
func TestParseAgentIssuesRequest(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantReq   AgentIssuesRequest
		wantError bool
	}{
		{
			name: "basic scenario query",
			url:  "/api/v1/agent/issues?scenario=test-scenario",
			wantReq: AgentIssuesRequest{
				Scenario: "test-scenario",
				Limit:    10,
			},
			wantError: false,
		},
		{
			name: "with limit parameter",
			url:  "/api/v1/agent/issues?scenario=test&limit=5",
			wantReq: AgentIssuesRequest{
				Scenario: "test",
				Limit:    5,
			},
			wantError: false,
		},
		{
			name: "with file filter",
			url:  "/api/v1/agent/issues?scenario=test&file=api/main.go&limit=20",
			wantReq: AgentIssuesRequest{
				Scenario: "test",
				File:     "api/main.go",
				Limit:    20,
			},
			wantError: false,
		},
		{
			name: "with folder filter",
			url:  "/api/v1/agent/issues?scenario=test&folder=api/",
			wantReq: AgentIssuesRequest{
				Scenario: "test",
				Folder:   "api/",
				Limit:    10,
			},
			wantError: false,
		},
		{
			name: "with category filter",
			url:  "/api/v1/agent/issues?scenario=test&category=dead_code",
			wantReq: AgentIssuesRequest{
				Scenario: "test",
				Category: "dead_code",
				Limit:    10,
			},
			wantError: false,
		},
		{
			name: "with force flag",
			url:  "/api/v1/agent/issues?scenario=test&force=true",
			wantReq: AgentIssuesRequest{
				Scenario: "test",
				Force:    true,
				Limit:    10,
			},
			wantError: false,
		},
		{
			name: "with all filters",
			url:  "/api/v1/agent/issues?scenario=test&file=api/main.go&category=length&limit=15",
			wantReq: AgentIssuesRequest{
				Scenario: "test",
				File:     "api/main.go",
				Category: "length",
				Limit:    15,
			},
			wantError: false,
		},
		{
			name:      "invalid limit - not a number",
			url:       "/api/v1/agent/issues?scenario=test&limit=abc",
			wantError: true,
		},
		{
			name:      "invalid limit - negative",
			url:       "/api/v1/agent/issues?scenario=test&limit=-5",
			wantError: true,
		},
		{
			name: "zero limit defaults to 10",
			url:  "/api/v1/agent/issues?scenario=test&limit=0",
			wantReq: AgentIssuesRequest{
				Scenario: "test",
				Limit:    10,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			parsed := parseAgentIssuesRequest(req)

			if tt.wantError {
				if parsed.Error == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if parsed.Error != nil {
					t.Errorf("Unexpected error: %v", parsed.Error)
				}
				if parsed.Request.Scenario != tt.wantReq.Scenario {
					t.Errorf("Scenario = %v, want %v", parsed.Request.Scenario, tt.wantReq.Scenario)
				}
				if parsed.Request.File != tt.wantReq.File {
					t.Errorf("File = %v, want %v", parsed.Request.File, tt.wantReq.File)
				}
				if parsed.Request.Folder != tt.wantReq.Folder {
					t.Errorf("Folder = %v, want %v", parsed.Request.Folder, tt.wantReq.Folder)
				}
				if parsed.Request.Category != tt.wantReq.Category {
					t.Errorf("Category = %v, want %v", parsed.Request.Category, tt.wantReq.Category)
				}
				if parsed.Request.Limit != tt.wantReq.Limit {
					t.Errorf("Limit = %v, want %v", parsed.Request.Limit, tt.wantReq.Limit)
				}
				if parsed.Request.Force != tt.wantReq.Force {
					t.Errorf("Force = %v, want %v", parsed.Request.Force, tt.wantReq.Force)
				}
			}
		})
	}
}
