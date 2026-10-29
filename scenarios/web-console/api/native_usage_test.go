package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/sessionstore"
)

func TestNativeUsageResolvesSessionThreadsAndRetainsUnknown(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "sessions")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "rollout.jsonl")
	record := `{"timestamp":"2026-01-01T00:01:00Z","type":"token_usage_record","payload":{"thread_id":"one","response_id":"r1","usage":{"input_tokens":10,"cached_input_tokens":7,"output_tokens":2}}}` + "\n"
	if err := os.WriteFile(path, []byte(record+record), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(home, "state_5.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE threads(id TEXT PRIMARY KEY, rollout_path TEXT); INSERT INTO threads VALUES ('one', ?)", path); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := readNativeUsage(context.Background(), home, nativeUsageRequest{ThreadIDs: []string{"one", "missing"}, Start: start, End: start.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "partial" || got.Threads[0].Status != "ok" || got.Threads[1].Status != "unknown" || got.Threads[1].Report.Usage != nil || got.Total.Responses != 1 || got.Total.Duplicates != 1 || got.Total.Usage == nil || got.Total.Usage.UncachedInputTokens != 3 {
		t.Fatalf("unexpected report: %+v", got)
	}
	if _, err := openNativeRollout(dir, filepath.Join(home, "outside.jsonl")); err == nil {
		t.Fatal("accepted outside session")
	}
}

func TestNativeUsageReportsPersistedThreadSettings(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "sessions")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "rollout.jsonl")
	record := `{"timestamp":"2026-01-01T00:01:00Z","type":"token_usage_record","payload":{"thread_id":"one","response_id":"r1","usage":{"input_tokens":10,"cached_input_tokens":7,"output_tokens":2}}}` + "\n"
	if err := os.WriteFile(path, []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(home, "state_5.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE threads(id TEXT PRIMARY KEY, rollout_path TEXT, model TEXT, model_provider TEXT, reasoning_effort TEXT); INSERT INTO threads VALUES ('one', ?, 'gpt-6-luna', 'openai', 'high')", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := readNativeUsage(context.Background(), home, nativeUsageRequest{ThreadIDs: []string{"one"}, Start: start, End: start.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	row := got.Threads[0]
	if row.Model != "gpt-6-luna" || row.Provider != "openai" || row.Effort != "high" {
		t.Fatalf("settings = %q/%q/%q", row.Model, row.Provider, row.Effort)
	}
}

func TestNativeUsageRetainedAudit(t *testing.T) {
	auditPath := os.Getenv("CODEX_NATIVE_USAGE_AUDIT")
	home := os.Getenv("CODEX_NATIVE_USAGE_HOME")
	if auditPath == "" || home == "" {
		t.Skip("retained native audit not selected")
	}
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	var audit struct {
		Threads map[string]struct {
			Thread string `json:"thread_id"`
		} `json:"threads"`
		Total struct {
			Input  int64 `json:"input_tokens"`
			Cached int64 `json:"cached_input_tokens"`
			Output int64 `json:"output_tokens"`
		} `json:"six_thread_usage"`
	}
	if err = json.Unmarshal(data, &audit); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, thread := range audit.Threads {
		ids = append(ids, thread.Thread)
	}
	if len(ids) != 6 {
		t.Fatal("expected six threads")
	}
	start := time.Date(2026, 10, 27, 4, 30, 0, 0, time.UTC)
	id := filepath.Base(home)
	root := filepath.Dir(filepath.Dir(home))
	t.Setenv("WC_SESSION_STATE_ROOT", root)
	store := sessionstore.NewInMemory()
	if err = store.Save(t.Context(), sessionstore.Metadata{ID: id}); err != nil {
		t.Fatal(err)
	}
	got, err := (&Server{sessionStore: store}).readSessionNativeUsage(t.Context(), &sessionsv1.ReadNativeUsageRequest{SessionId: id, ThreadIds: ids, Start: start.Format(time.RFC3339), End: time.Date(2026, 10, 27, 12, 32, 0, 0, time.UTC).Format(time.RFC3339)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "ok" || got.Total.Responses != 4052 || got.Total.Usage == nil || got.Total.Usage.InputTokens != audit.Total.Input || got.Total.Usage.CachedInputTokens != audit.Total.Cached || got.Total.Usage.OutputTokens != audit.Total.Output {
		t.Fatalf("unexpected report: %+v total=%+v", got, got.Total.Usage)
	}
	t.Logf("native owner: responses=%d usage=%+v", got.Total.Responses, got.Total.Usage)
}

func TestNativeUsageHandlerSelectsOnlyExistingSession(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_SESSION_STATE_ROOT", root)
	id := "1e5904ed-8378-458a-86d7-8dbb039cff19"
	thread := "01a17b9f-ed59-7a71-964c-4d6214cce350"
	home := filepath.Join(root, "codex", id)
	dir := filepath.Join(home, "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "r.jsonl")
	record := `{"timestamp":"2026-01-01T00:00:30Z","type":"turn_context","payload":{"model":"gpt-6-luna","effort":"high"}}` + "\n" + `{"timestamp":"2026-01-01T00:01:00Z","type":"token_usage_record","payload":{"thread_id":"` + thread + `","response_id":"r1","usage":{"input_tokens":0,"cached_input_tokens":0,"output_tokens":0}}}` + "\n"
	if err := os.WriteFile(path, []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(home, "state_5.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE threads(id TEXT PRIMARY KEY, rollout_path TEXT, model TEXT, model_provider TEXT, reasoning_effort TEXT); INSERT INTO threads VALUES (?, ?, 'gpt-6-luna', 'openai', 'high')", thread, path); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	store := sessionstore.NewInMemory()
	if err = store.Save(t.Context(), sessionstore.Metadata{ID: id}); err != nil {
		t.Fatal(err)
	}
	server := &Server{sessionStore: store}
	req := &sessionsv1.ReadNativeUsageRequest{SessionId: id, ThreadIds: []string{thread}, Start: "2026-01-01T00:00:00Z", End: "2026-01-01T01:00:00Z"}
	result, err := server.readSessionNativeUsage(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" || result.Total.Usage == nil || result.Total.Responses != 1 {
		t.Fatalf("zero was lost: %+v", result)
	}
	if len(result.Threads) != 1 || result.Threads[0].GetObservedModel() != "gpt-6-luna" || result.Threads[0].GetObservedProvider() != "openai" || result.Threads[0].GetObservedEffort() != "high" {
		t.Fatalf("persisted settings not exposed: %+v", result.Threads)
	}
	contexts := result.Threads[0].GetModelContexts()
	if len(contexts) != 1 || contexts[0].GetModel() != "gpt-6-luna" || contexts[0].GetEffort() != "high" || contexts[0].GetCount() != 1 {
		t.Fatalf("recorded model contexts not exposed: %+v", contexts)
	}
	req.SessionId = "00000000-0000-0000-0000-000000000000"
	if _, err = server.readSessionNativeUsage(t.Context(), req); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown session: %v", err)
	}
}

func TestNativeUsageRejectsEscapedSymlink(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "sessions")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(home, "outside.jsonl")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.jsonl")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if f, err := openNativeRollout(dir, link); err == nil {
		f.Close()
		t.Fatal("escaped symlink admitted")
	}
}
