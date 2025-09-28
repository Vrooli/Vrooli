package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	tempDir := t.TempDir()

	// Ensure folder structure exists for tests
	for _, folder := range []string{"open", "investigating", "in-progress", "fixed", "closed", "failed", "templates"} {
		if err := os.MkdirAll(filepath.Join(tempDir, folder), 0o755); err != nil {
			t.Fatalf("failed to create test folder %s: %v", folder, err)
		}
	}

	cfg := &Config{IssuesDir: tempDir}
	return &Server{config: cfg}
}

func TestGetIssueHandlerReturnsIssue(t *testing.T) {
	server := newTestServer(t)

	issue := &Issue{
		ID:       "issue-123",
		Title:    "Login fails",
		Priority: "high",
		Type:     "bug",
		AppID:    "app-x",
	}

	if _, err := server.saveIssue(issue, "open"); err != nil {
		t.Fatalf("failed to seed issue: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/issue-123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": issue.ID})

	rr := httptest.NewRecorder()
	server.getIssueHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rr.Code, http.StatusOK)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Issue Issue `json:"issue"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success response")
	}

	if resp.Data.Issue.ID != issue.ID {
		t.Fatalf("unexpected issue id: got %s want %s", resp.Data.Issue.ID, issue.ID)
	}
	if resp.Data.Issue.Status != "open" {
		t.Fatalf("expected status 'open', got %s", resp.Data.Issue.Status)
	}
}

func TestUpdateIssueHandlerMovesIssueAndUpdatesFields(t *testing.T) {
	server := newTestServer(t)

	issue := &Issue{
		ID:       "issue-456",
		Title:    "Search is broken",
		Priority: "medium",
		Type:     "bug",
		AppID:    "app-y",
	}

	if _, err := server.saveIssue(issue, "open"); err != nil {
		t.Fatalf("failed to seed issue: %v", err)
	}

	payload := map[string]any{
		"title":    "Search endpoint fails",
		"status":   "investigating",
		"watchers": []string{" dev-team "},
		"notes":    "Reproduced on staging",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/issues/issue-456", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": issue.ID})

	rr := httptest.NewRecorder()
	server.updateIssueHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rr.Code, http.StatusOK)
	}

	refreshedDir, newFolder, err := server.findIssueDirectory(issue.ID)
	if err != nil {
		t.Fatalf("failed to locate updated issue: %v", err)
	}
	if newFolder != "investigating" {
		t.Fatalf("expected issue folder 'investigating', got %s", newFolder)
	}

	updatedIssue, err := server.loadIssueFromDir(refreshedDir)
	if err != nil {
		t.Fatalf("failed to load updated issue: %v", err)
	}

	if updatedIssue.Title != "Search endpoint fails" {
		t.Fatalf("expected updated title, got %s", updatedIssue.Title)
	}
	if updatedIssue.Metadata.Watchers == nil || len(updatedIssue.Metadata.Watchers) != 1 || updatedIssue.Metadata.Watchers[0] != "dev-team" {
		t.Fatalf("unexpected watchers: %#v", updatedIssue.Metadata.Watchers)
	}
	if strings.TrimSpace(updatedIssue.Notes) != "Reproduced on staging" {
		t.Fatalf("unexpected notes: %q", updatedIssue.Notes)
	}
}

func TestDeleteIssueHandlerRemovesIssue(t *testing.T) {
	server := newTestServer(t)

	issue := &Issue{
		ID:       "issue-789",
		Title:    "UI glitch",
		Priority: "low",
		Type:     "bug",
		AppID:    "app-z",
	}

	issueDir, err := server.saveIssue(issue, "open")
	if err != nil {
		t.Fatalf("failed to seed issue: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/issues/issue-789", nil)
	req = mux.SetURLVars(req, map[string]string{"id": issue.ID})

	rr := httptest.NewRecorder()
	server.deleteIssueHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rr.Code, http.StatusOK)
	}

	if _, err := os.Stat(issueDir); !os.IsNotExist(err) {
		t.Fatalf("expected issue directory to be removed, stat err: %v", err)
	}
}

func TestCreateIssueHandlerStoresArtifacts(t *testing.T) {
	server := newTestServer(t)

	artifactContent := "line one\nline two"
	imagePayload := base64.StdEncoding.EncodeToString([]byte("fake-image"))

	payload := map[string]any{
		"title":       "UI glitch in dashboard",
		"description": "Steps to reproduce...",
		"app_id":      "app-dashboard",
		"metadata_extra": map[string]string{
			"source": "unit-test",
		},
		"artifacts": []map[string]string{
			{
				"name":         "Lifecycle Logs",
				"category":     "logs",
				"content":      artifactContent,
				"encoding":     "plain",
				"content_type": "text/plain",
			},
			{
				"name":         "Captured Screenshot",
				"category":     "screenshot",
				"content":      imagePayload,
				"encoding":     "base64",
				"content_type": "image/png",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	server.createIssueHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rr.Code, http.StatusOK)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Issue Issue `json:"issue"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success response")
	}

	created := resp.Data.Issue
	if created.ID == "" {
		t.Fatalf("expected issue id in response")
	}
	if created.Metadata.Extra == nil || created.Metadata.Extra["source"] != "unit-test" {
		t.Fatalf("metadata extra not persisted: %#v", created.Metadata.Extra)
	}
	if len(created.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(created.Attachments))
	}

	issueDir, _, err := server.findIssueDirectory(created.ID)
	if err != nil {
		t.Fatalf("failed to locate stored issue: %v", err)
	}
	for _, attachment := range created.Attachments {
		fullPath := filepath.Join(issueDir, attachment.Path)
		if _, err := os.Stat(fullPath); err != nil {
			t.Fatalf("expected attachment file %s to exist: %v", fullPath, err)
		}
	}

	storedIssue, err := server.loadIssueFromDir(issueDir)
	if err != nil {
		t.Fatalf("failed to load issue metadata: %v", err)
	}
	if len(storedIssue.Attachments) != 2 {
		t.Fatalf("expected attachments persisted to metadata, got %d", len(storedIssue.Attachments))
	}
}
