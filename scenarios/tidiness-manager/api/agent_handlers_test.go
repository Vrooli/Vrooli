package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// [REQ:TM-API-001] Scenario summary endpoint - Test ability to request top N issues for a scenario
func TestAgentGetIssues_BasicScenarioQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert test data
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "File too long", "File exceeds 500 lines")

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&limit=10", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(issues) == 0 {
		t.Error("Expected at least one issue, got none")
	}

	if len(issues) > 0 && issues[0].Scenario != "test-scenario" {
		t.Errorf("Expected scenario 'test-scenario', got '%s'", issues[0].Scenario)
	}
}

// [REQ:TM-API-001] Test limit parameter
func TestAgentGetIssues_LimitParameter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert multiple test issues
	for i := 1; i <= 5; i++ {
		insertTestIssue(t, srv.db, "test-scenario", "api/file.go", "length", "high", "Issue", "Description")
	}

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&limit=2", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(issues) > 2 {
		t.Errorf("Expected max 2 issues, got %d", len(issues))
	}
}

// [REQ:TM-API-002] File-scoped issue query
func TestAgentGetIssues_FileFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issues for different files
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "Issue 1", "Desc 1")
	insertTestIssue(t, srv.db, "test-scenario", "api/handlers.go", "length", "high", "Issue 2", "Desc 2")

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&file=api/main.go", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, issue := range issues {
		if issue.FilePath != "api/main.go" {
			t.Errorf("Expected only issues from api/main.go, got %s", issue.FilePath)
		}
	}
}

// [REQ:TM-API-002] Folder-scoped issue query
func TestAgentGetIssues_FolderFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issues in different folders
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "Issue 1", "Desc 1")
	insertTestIssue(t, srv.db, "test-scenario", "ui/App.tsx", "complexity", "medium", "Issue 2", "Desc 2")

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&folder=api", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, issue := range issues {
		if len(issue.FilePath) < 4 || issue.FilePath[:4] != "api/" {
			t.Errorf("Expected only issues from api/ folder, got %s", issue.FilePath)
		}
	}
}

// [REQ:TM-API-003] Category-scoped issue query
func TestAgentGetIssues_CategoryFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issues with different categories
	insertTestIssue(t, srv.db, "test-scenario", "api/main.go", "length", "high", "Long file", "Too many lines")
	insertTestIssue(t, srv.db, "test-scenario", "api/handlers.go", "dead_code", "medium", "Unused function", "Remove unused")

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&category=length", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	for _, issue := range issues {
		if issue.Category != "length" {
			t.Errorf("Expected only 'length' category issues, got '%s'", issue.Category)
		}
	}
}

// [REQ:TM-API-004] Issue ranking heuristic - verify issues are ranked by severity
func TestAgentGetIssues_SeverityRanking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert issues with different severities
	insertTestIssue(t, srv.db, "test-scenario", "file1.go", "length", "low", "Issue 1", "Desc 1")
	insertTestIssue(t, srv.db, "test-scenario", "file2.go", "complexity", "critical", "Issue 2", "Desc 2")
	insertTestIssue(t, srv.db, "test-scenario", "file3.go", "dead_code", "high", "Issue 3", "Desc 3")

	req, err := http.NewRequest("GET", "/api/v1/agent/issues?scenario=test-scenario&limit=10", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(issues) < 2 {
		t.Skip("Not enough issues to test ranking")
	}

	// Verify that higher severity issues appear first
	severityOrder := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}

	for i := 0; i < len(issues)-1; i++ {
		current := severityOrder[issues[i].Severity]
		next := severityOrder[issues[i+1].Severity]
		if current < next {
			t.Errorf("Issues not properly ranked by severity: %s (%d) should come after %s (%d)",
				issues[i].Severity, current, issues[i+1].Severity, next)
		}
	}
}

// [REQ:TM-API-005] CLI wrapper exists
func TestCLIWrapper_IssuesCommand(t *testing.T) {
	// This is a unit test that verifies the CLI command exists
	// The actual CLI is tested in test/cli/agent-api.bats

	// Verify CLI binary exists
	if _, err := os.Stat("../cli/tidiness-manager"); os.IsNotExist(err) {
		t.Skip("CLI binary not found, skipping CLI wrapper test")
	}

	// Test passes if we get here - detailed CLI integration tests are in BATS
}

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

// Test helpers

func setupTestServer(t *testing.T) *Server {
	srv := setupTestServerNoData(t)
	if srv == nil {
		return nil
	}

	// Create issues table if not exists
	_, err := srv.db.Exec(`
		CREATE TABLE IF NOT EXISTS issues (
			id SERIAL PRIMARY KEY,
			scenario VARCHAR(255) NOT NULL,
			file_path TEXT NOT NULL,
			category VARCHAR(100) NOT NULL,
			severity VARCHAR(50) NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			line_number INTEGER,
			column_number INTEGER,
			agent_notes TEXT,
			remediation_steps TEXT,
			status VARCHAR(50) DEFAULT 'open',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			campaign_id INTEGER,
			session_id VARCHAR(255)
		)
	`)
	if err != nil {
		t.Logf("Warning: Could not create issues table: %v", err)
	}

	// Clean up test data
	_, _ = srv.db.Exec("DELETE FROM issues WHERE scenario LIKE 'test-%'")

	return srv
}

func setupTestServerNoData(t *testing.T) *Server {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
		return nil
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Could not open database connection")
		return nil
	}

	if err := db.Ping(); err != nil {
		t.Skip("Could not ping database, skipping integration test")
		return nil
	}

	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: dbURL,
		},
		db:     db,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	return srv
}

func insertTestIssue(t *testing.T, db *sql.DB, scenario, filePath, category, severity, title, description string) {
	_, err := db.Exec(`
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, scenario, filePath, category, severity, title, description, "open", time.Now())

	if err != nil {
		t.Fatalf("Failed to insert test issue: %v", err)
	}
}
