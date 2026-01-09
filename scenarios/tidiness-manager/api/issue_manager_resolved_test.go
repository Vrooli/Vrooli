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

// [REQ:TM-IM-001] Edge case: mark non-existent issue as resolved
func TestIssueManager_MarkAsResolved_NotFound(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database-dependent test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	// Try to update non-existent issue
	reqBody := map[string]string{
		"status":           "resolved",
		"resolution_notes": "Fixed manually",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/api/v1/agent/issues/999999", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	// Should return 404 or error
	if w.Code == http.StatusOK {
		t.Errorf("Expected error status for non-existent issue, got 200")
	}
}

// [REQ:TM-IM-001] Edge case: invalid status transition
func TestIssueManager_InvalidStatus(t *testing.T) {
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

	// Try to set invalid status
	reqBody := map[string]string{
		"status": "invalid-status",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/agent/issues/%d", issueID), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	// Should reject invalid status
	if w.Code == http.StatusOK {
		var status string
		srv.db.QueryRow("SELECT status FROM issues WHERE id = $1", issueID).Scan(&status)
		if status == "invalid-status" {
			t.Error("Invalid status was accepted - should be rejected")
		}
	}
}
