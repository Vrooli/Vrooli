package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRow struct {
	values []any
	err    error
}

func (f fakeRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(dest) != len(f.values) {
		return sql.ErrNoRows
	}
	for i, v := range f.values {
		switch d := dest[i].(type) {
		case *string:
			*d = v.(string)
		case *time.Time:
			*d = v.(time.Time)
		case *sql.NullString:
			*d = v.(sql.NullString)
		default:
			return sql.ErrNoRows
		}
	}
	return nil
}

type fakeStore struct {
	row rowScanner

	calls []execCall
}

func (s *fakeStore) QueryRow(_ string, _ ...any) rowScanner {
	return s.row
}

func (s *fakeStore) Exec(query string, args ...any) (sql.Result, error) {
	s.calls = append(s.calls, execCall{query: query, args: args})
	return noopResult{}, nil
}

func TestExecuteAIGenerateDraftValidation(t *testing.T) {
	store := &fakeStore{row: fakeRow{err: sql.ErrNoRows}}
	resp, status := executeAIGenerateDraft(store, AIGenerateDraftRequest{
		EntityType: "nope",
		EntityName: "test",
		Section:    "🎯 Full PRD",
	})

	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if resp.Success {
		t.Fatalf("success = true, want false")
	}
	if !strings.Contains(resp.Message, "Invalid entity type") {
		t.Fatalf("message %q missing expected text", resp.Message)
	}
}

func TestExecuteAIGenerateDraftCreatesDraftAndPersistsGenerated(t *testing.T) {
	oldKey := os.Getenv("OPENROUTER_API_KEY")
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Cleanup(func() {
		if oldKey != "" {
			_ = os.Setenv("OPENROUTER_API_KEY", oldKey)
		}
	})

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"anthropic/claude-3.5-sonnet","choices":[{"message":{"role":"assistant","content":"# Generated PRD\n\nHello"}}]}`))
	}))
	defer mockServer.Close()
	t.Setenv("RESOURCE_OPENROUTER_URL", mockServer.URL)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}
	if err := os.Chdir(apiDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	store := &fakeStore{row: fakeRow{err: sql.ErrNoRows}}
	resp, status := executeAIGenerateDraft(store, AIGenerateDraftRequest{
		EntityType: "scenario",
		EntityName: "new-scenario",
		Section:    "🎯 Full PRD",
		Context:    "Make it great",
	})

	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d, message=%q", status, http.StatusOK, resp.Message)
	}
	if !resp.Success {
		t.Fatalf("success = false, want true, message=%q", resp.Message)
	}
	if resp.DraftID == "" {
		t.Fatalf("draft_id is empty")
	}
	if !resp.SavedToDraft {
		t.Fatalf("saved_to_draft = false, want true")
	}
	if resp.DraftFilePath == "" {
		t.Fatalf("draft_file_path is empty")
	}
	if !strings.Contains(resp.GeneratedText, "# Generated PRD") {
		t.Fatalf("generated_text missing expected content: %q", resp.GeneratedText)
	}

	// Verify the draft file was written.
	fileBytes, err := os.ReadFile(resp.DraftFilePath)
	if err != nil {
		t.Fatalf("failed to read draft file: %v", err)
	}
	if strings.TrimSpace(string(fileBytes)) != strings.TrimSpace(resp.GeneratedText) {
		t.Fatalf("draft file content mismatch: got %q want %q", string(fileBytes), resp.GeneratedText)
	}

	// Ensure we recorded an insert and an update.
	if len(store.calls) < 2 {
		t.Fatalf("expected at least 2 Exec calls, got %d", len(store.calls))
	}
	if !strings.Contains(store.calls[0].query, "INSERT INTO drafts") {
		t.Fatalf("first Exec query expected INSERT, got %q", store.calls[0].query)
	}
	foundUpdate := false
	for _, call := range store.calls {
		if strings.Contains(call.query, "UPDATE drafts") {
			foundUpdate = true
		}
	}
	if !foundUpdate {
		t.Fatalf("expected an UPDATE drafts Exec call")
	}
}

func TestExecuteAIGenerateDraftNoPersist(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"anthropic/claude-3.5-sonnet","choices":[{"message":{"role":"assistant","content":"Generated"}}]}`))
	}))
	defer mockServer.Close()
	t.Setenv("RESOURCE_OPENROUTER_URL", mockServer.URL)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}
	if err := os.Chdir(apiDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	store := &fakeStore{row: fakeRow{err: sql.ErrNoRows}}
	save := false
	resp, status := executeAIGenerateDraft(store, AIGenerateDraftRequest{
		EntityType:           "scenario",
		EntityName:           "no-persist",
		Section:              "🎯 Full PRD",
		SaveGeneratedToDraft: &save,
	})

	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d, message=%q", status, http.StatusOK, resp.Message)
	}
	if resp.SavedToDraft {
		t.Fatalf("saved_to_draft = true, want false")
	}
	if resp.DraftFilePath != "" {
		t.Fatalf("draft_file_path = %q, want empty", resp.DraftFilePath)
	}
	if len(store.calls) != 1 || !strings.Contains(store.calls[0].query, "INSERT INTO drafts") {
		t.Fatalf("expected only INSERT Exec call, got %d calls", len(store.calls))
	}
	if _, err := os.Stat(getDraftPath("scenario", "no-persist")); err == nil {
		t.Fatalf("expected no draft file to be written")
	}
}

