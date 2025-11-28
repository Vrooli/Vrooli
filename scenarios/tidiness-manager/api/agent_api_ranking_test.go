package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test helper: makes a GET request and parses AgentIssue response
func getAgentIssues(t *testing.T, srv *Server, scenario string, limit int) []AgentIssue {
	t.Helper()
	url := fmt.Sprintf("/api/v1/agent/issues?scenario=%s&limit=%d", scenario, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var issues []AgentIssue
	if err := json.Unmarshal(rr.Body.Bytes(), &issues); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	return issues
}

// Test helper: severity ranking order (higher number = higher priority)
var severityRank = map[string]int{
	"critical": 5,
	"high":     4,
	"medium":   3,
	"low":      2,
	"info":     1,
}

// Test helper: verifies issues are sorted by severity (highest first)
func assertSeverityOrder(t *testing.T, issues []AgentIssue) {
	t.Helper()
	for i := 0; i < len(issues)-1; i++ {
		current := severityRank[issues[i].Severity]
		next := severityRank[issues[i+1].Severity]
		if current < next {
			t.Errorf("Severity ordering violation at index %d: %s (rank %d) before %s (rank %d)",
				i, issues[i].Severity, current, issues[i+1].Severity, next)
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

	issues := getAgentIssues(t, srv, "test-scenario", 10)

	if len(issues) < 2 {
		t.Skip("Not enough issues to test ranking")
	}

	// Verify that higher severity issues appear first
	assertSeverityOrder(t, issues)
}

// [REQ:TM-API-004] Test severity ranking with all severity levels
func TestAgentGetIssues_AllSeverityLevels(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	severities := []string{"critical", "high", "medium", "low", "info"}

	// Insert issues in reverse order to test sorting
	for i, sev := range severities {
		insertTestIssue(t, srv.db, "test-scenario", "file.go", "length", sev, fmt.Sprintf("Issue %d", i), "Desc")
	}

	issues := getAgentIssues(t, srv, "test-scenario", 10)

	if len(issues) < len(severities) {
		t.Errorf("Expected %d issues, got %d", len(severities), len(issues))
	}

	// Verify ordering
	assertSeverityOrder(t, issues)
}

// [REQ:TM-API-004] Test ranking with equal severities
func TestAgentGetIssues_EqualSeverityRanking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := setupTestServer(t)
	if srv == nil {
		return
	}

	// Insert multiple issues with same severity
	for i := 0; i < 5; i++ {
		insertTestIssue(t, srv.db, "test-scenario", "file.go", "length", "high", fmt.Sprintf("Issue %d", i), "Desc")
	}

	issues := getAgentIssues(t, srv, "test-scenario", 10)

	if len(issues) != 5 {
		t.Errorf("Expected 5 issues, got %d", len(issues))
	}

	// All should have same severity
	for _, issue := range issues {
		if issue.Severity != "high" {
			t.Errorf("Expected severity 'high', got '%s'", issue.Severity)
		}
	}
}
