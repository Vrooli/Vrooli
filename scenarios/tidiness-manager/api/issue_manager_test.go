package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// [REQ:TM-IM-001] Issue manager mark-as-resolved
func TestIssueManager_MarkAsResolved(t *testing.T) {
	// Skip if DATABASE_URL not provided
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database-dependent test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	// Insert test issue
	var issueID int
	err = srv.db.QueryRow(`
		INSERT INTO issues (scenario, file_path, category, severity, title, description, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "test-scenario", "test.go", "lint", "error", "Test issue", "Test description", "open").Scan(&issueID)
	if err != nil {
		t.Fatalf("Failed to insert test issue: %v", err)
	}
	defer srv.db.Exec("DELETE FROM issues WHERE id = $1", issueID)

	// Update issue to resolved
	reqBody := map[string]string{
		"status":           "resolved",
		"resolution_notes": "Fixed manually",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/agent/issues/%d", issueID), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status changed in database
	var status string
	err = srv.db.QueryRow("SELECT status FROM issues WHERE id = $1", issueID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query issue status: %v", err)
	}
	if status != "resolved" {
		t.Errorf("Expected status 'resolved', got '%s'", status)
	}
}

// [REQ:TM-IM-002] Issue manager mark-as-ignored
func TestIssueManager_MarkAsIgnored(t *testing.T) {
	// Skip if DATABASE_URL not provided
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database-dependent test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	// Insert test issue
	var issueID int
	err = srv.db.QueryRow(`
		INSERT INTO issues (scenario, file_path, category, severity, title, description, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "test-scenario", "test.go", "type", "warning", "Test issue", "Test description", "open").Scan(&issueID)
	if err != nil {
		t.Fatalf("Failed to insert test issue: %v", err)
	}
	defer srv.db.Exec("DELETE FROM issues WHERE id = $1", issueID)

	// Update issue to ignored
	reqBody := map[string]string{
		"status":           "ignored",
		"resolution_notes": "False positive",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/agent/issues/%d", issueID), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status changed in database
	var status, notes string
	err = srv.db.QueryRow("SELECT status, resolution_notes FROM issues WHERE id = $1", issueID).Scan(&status, &notes)
	if err != nil {
		t.Fatalf("Failed to query issue: %v", err)
	}
	if status != "ignored" {
		t.Errorf("Expected status 'ignored', got '%s'", status)
	}
	if notes != "False positive" {
		t.Errorf("Expected notes 'False positive', got '%s'", notes)
	}
}

// [REQ:TM-IM-003] Issue filter by status
func TestIssueManager_FilterByStatus(t *testing.T) {
	// Skip if DATABASE_URL not provided
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database-dependent test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	// Insert test issues with different statuses
	var openID, resolvedID, ignoredID int
	srv.db.QueryRow(`
		INSERT INTO issues (scenario, file_path, category, severity, title, description, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, "test-filter", "test.go", "lint", "error", "Open issue", "Description", "open").Scan(&openID)
	srv.db.QueryRow(`
		INSERT INTO issues (scenario, file_path, category, severity, title, description, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, "test-filter", "test2.go", "type", "warning", "Resolved issue", "Description", "resolved").Scan(&resolvedID)
	srv.db.QueryRow(`
		INSERT INTO issues (scenario, file_path, category, severity, title, description, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, "test-filter", "test3.go", "ai", "warning", "Ignored issue", "Description", "ignored").Scan(&ignoredID)

	defer func() {
		srv.db.Exec("DELETE FROM issues WHERE scenario = $1", "test-filter")
	}()

	// Test filtering by status=open
	req := httptest.NewRequest("GET", "/api/v1/agent/issues?scenario=test-filter&status=open", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var issues []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &issues)
	if len(issues) != 1 {
		t.Errorf("Expected 1 open issue, got %d", len(issues))
	}

	// Test filtering by status=resolved
	req = httptest.NewRequest("GET", "/api/v1/agent/issues?scenario=test-filter&status=resolved", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &issues)
	if len(issues) != 1 {
		t.Errorf("Expected 1 resolved issue, got %d", len(issues))
	}
}

// [REQ:TM-IM-004] Issue detail view
func TestIssueManager_ViewDetails(t *testing.T) {
	t.Skip("TM-IM-004: Issue detail view not yet implemented (P1 feature)")
}

// [REQ:TM-IM-005] Issue de-duplication by source
func TestIssueManager_DeDuplication(t *testing.T) {
	t.Skip("TM-IM-005: Issue de-duplication not yet implemented (P1 feature)")
}

// [REQ:TM-IM-006] Trigger one-off light scan
func TestIssueManager_TriggerLightScan(t *testing.T) {
	t.Skip("TM-IM-006: One-off scan triggers not yet implemented (P1 feature)")
}

// [REQ:TM-IM-007] Trigger one-off smart scan
func TestIssueManager_TriggerSmartScan(t *testing.T) {
	t.Skip("TM-IM-007: One-off smart scan triggers not yet implemented (P1 feature)")
}

// [REQ:TM-IM-008] Enable/disable campaign from UI
func TestIssueManager_ToggleCampaign(t *testing.T) {
	t.Skip("TM-IM-008: Campaign toggle not yet implemented (P1 feature)")
}
