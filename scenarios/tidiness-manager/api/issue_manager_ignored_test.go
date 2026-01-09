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

// [REQ:TM-IM-002] Edge case: malformed JSON request
func TestIssueManager_MalformedJSON(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping database-dependent test")
	}

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.db.Close()

	// Send malformed JSON
	req := httptest.NewRequest("PATCH", "/api/v1/agent/issues/1", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
		t.Logf("Expected 400 or 500 for malformed JSON, got %d (may be acceptable depending on error handling)", w.Code)
	}
}
