package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	env := setupTestDirectory(t)
	return env.Server
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
		"status":   "active",
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
	if newFolder != "active" {
		t.Fatalf("expected issue folder 'active', got %s", newFolder)
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
	if created.Attachments[0].Category == "" || created.Attachments[1].Category == "" {
		t.Fatalf("expected attachment categories to be populated: %#v", created.Attachments)
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

func TestGetIssueAttachmentHandlerServesContent(t *testing.T) {
	server := newTestServer(t)

	artifactContent := "line one\nline two"
	payload := map[string]any{
		"title":       "Attachment fetch",
		"description": "desc",
		"app_id":      "app-dashboard",
		"artifacts": []map[string]string{
			{
				"name":         "Execution Logs",
				"category":     "logs",
				"content":      artifactContent,
				"encoding":     "plain",
				"content_type": "text/plain",
			},
			{
				"name":         "Screenshot",
				"category":     "screenshot",
				"content":      base64.StdEncoding.EncodeToString([]byte("fake-image")),
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
	var createResp struct {
		Success bool `json:"success"`
		Data    struct {
			Issue Issue `json:"issue"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if !createResp.Success {
		t.Fatalf("expected create success")
	}
	if len(createResp.Data.Issue.Attachments) == 0 {
		t.Fatalf("expected at least one attachment")
	}
	attachment := createResp.Data.Issue.Attachments[0]
	attachmentPath := filepath.ToSlash(attachment.Path)
	requestPath := fmt.Sprintf("/api/v1/issues/%s/attachments/%s", createResp.Data.Issue.ID, attachmentPath)
	attachReq := httptest.NewRequest(http.MethodGet, requestPath, nil)
	attachReq = mux.SetURLVars(attachReq, map[string]string{
		"id":         createResp.Data.Issue.ID,
		"attachment": attachmentPath,
	})
	attachRes := httptest.NewRecorder()
	server.getIssueAttachmentHandler(attachRes, attachReq)
	if attachRes.Code != http.StatusOK {
		t.Fatalf("unexpected attachment status code: got %d, want %d", attachRes.Code, http.StatusOK)
	}
	if ct := attachRes.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("unexpected content type: %s", ct)
	}
	data, err := io.ReadAll(attachRes.Body)
	if err != nil {
		t.Fatalf("failed to read attachment body: %v", err)
	}
	if string(data) != artifactContent {
		t.Fatalf("unexpected attachment body: %q", string(data))
	}
}

func TestGetIssueAgentConversationHandler(t *testing.T) {
	server := newTestServer(t)

	issue := &Issue{
		ID:    "issue-xyz",
		Title: "Transcript check",
		Metadata: struct {
			CreatedAt  string            `yaml:"created_at" json:"created_at"`
			UpdatedAt  string            `yaml:"updated_at" json:"updated_at"`
			ResolvedAt string            `yaml:"resolved_at,omitempty" json:"resolved_at,omitempty"`
			Tags       []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
			Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
			Watchers   []string          `yaml:"watchers,omitempty" json:"watchers,omitempty"`
			Extra      map[string]string `yaml:"extra,omitempty" json:"extra,omitempty"`
		}{
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			Extra:     make(map[string]string),
		},
	}

	transcriptDir := filepath.Join(server.config.ScenarioRoot, "tmp", "codex")
	if err := os.MkdirAll(transcriptDir, 0755); err != nil {
		t.Fatalf("failed to create transcript directory: %v", err)
	}

	transcriptPath := filepath.Join(transcriptDir, "issue-xyz-conversation.jsonl")
	transcriptLines := []string{
		`{"sandbox":"workspace-write","provider":"openai","model":"gpt-5-codex"}`,
		`{"prompt":"Fix the failing test\\n"}`,
		`{"id":"0","msg":{"type":"task_started"}}`,
		`{"id":"0","msg":{"type":"agent_message","message":"All good now."}}`,
	}
	if err := os.WriteFile(transcriptPath, []byte(strings.Join(transcriptLines, "\n")), 0644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	lastMessagePath := filepath.Join(transcriptDir, "issue-xyz-last.txt")
	if err := os.WriteFile(lastMessagePath, []byte("All good now."), 0644); err != nil {
		t.Fatalf("failed to write last message: %v", err)
	}

	issue.Metadata.Extra["agent_transcript_path"] = transcriptPath
	issue.Metadata.Extra["agent_last_message_path"] = lastMessagePath
	issue.Investigation.AgentID = "codex"

	if _, err := server.saveIssue(issue, "completed"); err != nil {
		t.Fatalf("failed to save issue: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/issue-xyz/agent/conversation", nil)
	req = mux.SetURLVars(req, map[string]string{
		"id": "issue-xyz",
	})

	resp := httptest.NewRecorder()
	server.getIssueAgentConversationHandler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d (%s)", resp.Code, resp.Body.String())
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Conversation AgentConversationPayload `json:"conversation"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !body.Success {
		t.Fatalf("expected success response")
	}

	conversation := body.Data.Conversation
	if !conversation.Available {
		t.Fatalf("expected transcript to be available")
	}
	if conversation.Provider != "codex" {
		t.Fatalf("expected provider codex, got %s", conversation.Provider)
	}
	if conversation.LastMessage != "All good now." {
		t.Fatalf("unexpected last message: %s", conversation.LastMessage)
	}
	if len(conversation.Entries) == 0 {
		t.Fatalf("expected conversation entries")
	}
	if conversation.Entries[0].Kind != "prompt" {
		t.Fatalf("expected first entry to be prompt, got %s", conversation.Entries[0].Kind)
	}
}

func TestGetIssueAgentConversationHandlerWithoutTranscript(t *testing.T) {
	server := newTestServer(t)

	issue := &Issue{
		ID:    "issue-no-transcript",
		Title: "No transcript available",
	}
	if _, err := server.saveIssue(issue, "open"); err != nil {
		t.Fatalf("failed to save issue: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/issue-no-transcript/agent/conversation", nil)
	req = mux.SetURLVars(req, map[string]string{"id": issue.ID})
	resp := httptest.NewRecorder()
	server.getIssueAgentConversationHandler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d", resp.Code)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Conversation AgentConversationPayload `json:"conversation"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !body.Success {
		t.Fatalf("expected success response")
	}
	if body.Data.Conversation.Available {
		t.Fatalf("expected transcript to be unavailable")
	}
}

func TestProcessorStateInitialization(t *testing.T) {
	server := newTestServer(t)

	state := server.currentProcessorState()

	if state.Active {
		t.Errorf("expected processor to be inactive by default")
	}
	if state.ConcurrentSlots != 2 {
		t.Errorf("expected 2 concurrent slots, got %d", state.ConcurrentSlots)
	}
	if state.RefreshInterval != 45 {
		t.Errorf("expected refresh interval of 45s, got %d", state.RefreshInterval)
	}
	if state.MaxIssues != 0 {
		t.Errorf("expected max issues to be 0 (unlimited), got %d", state.MaxIssues)
	}
	if !state.MaxIssuesDisabled {
		t.Errorf("expected max issues to be disabled by default")
	}
}

func TestProcessorStateUpdates(t *testing.T) {
	server := newTestServer(t)

	active := true
	slots := 4
	interval := 60
	maxIssues := 10
	maxIssuesDisabled := false

	server.updateProcessorState(&active, &slots, &interval, &maxIssues, &maxIssuesDisabled)

	state := server.currentProcessorState()

	if !state.Active {
		t.Errorf("expected processor to be active")
	}
	if state.ConcurrentSlots != 4 {
		t.Errorf("expected 4 concurrent slots, got %d", state.ConcurrentSlots)
	}
	if state.RefreshInterval != 60 {
		t.Errorf("expected refresh interval of 60s, got %d", state.RefreshInterval)
	}
	if state.MaxIssues != 10 {
		t.Errorf("expected max issues to be 10, got %d", state.MaxIssues)
	}
	if state.MaxIssuesDisabled {
		t.Errorf("expected max issues to be enabled")
	}
}
