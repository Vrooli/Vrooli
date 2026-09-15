package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"web-console/internal/backend"
	"web-console/internal/sessionstore"
)

func newClaudeTailerTestServer(t *testing.T) (*Server, string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".vrooli", "cache", "vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	srv, sess := newCodexTailerTestServer(t)
	srv.sessionStore = sessionstore.NewInMemory()
	srv.agentCheckpointStore = NewInMemoryAgentTranscriptCheckpointStore()
	cwd := "/workspace/claude-project"
	if err := srv.sessionStore.Save(context.Background(), sessionstore.Metadata{
		ID: sess.ID, Backend: backend.Persistent, Shell: "/bin/sh", Cols: 80, Rows: 24,
		Created: time.Now(), Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude,
		AgentSessionID: "claude-session", CWD: cwd,
	}); err != nil {
		t.Fatalf("save session metadata: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return srv, sess.ID, claudeTranscriptPath(home, cwd, "claude-session")
}

func writeClaudeTranscript(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeTailer_EmitsOrderedTextAndSkipsToolResults(t *testing.T) {
	srv, sessionID, path := newClaudeTailerTestServer(t)
	writeClaudeTranscript(t, path,
		`{"type":"user","message":{"content":[{"type":"text","text":"question"}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"before tool"},{"type":"tool_use"},{"type":"text","text":"after tool"}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","text":"ignored"}]}}`,
	)

	tailer := NewClaudeTailer(srv)
	tailer.scan()
	state := srv.conversations.ListSession(context.Background(), sessionID)
	if len(state.Events) != 3 {
		t.Fatalf("events = %+v, want 3", state.Events)
	}
	got := []string{state.Events[0].Text, state.Events[1].Text, state.Events[2].Text}
	want := []string{"question", "before tool", "after tool"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestClaudeTailer_ResumesFromCheckpoint(t *testing.T) {
	srv, sessionID, path := newClaudeTailerTestServer(t)
	writeClaudeTranscript(t, path, `{"type":"assistant","message":{"content":[{"type":"text","text":"first"}]}}`)
	tailer := NewClaudeTailer(srv)
	tailer.scan()
	writeClaudeTranscript(t, path, `{"type":"assistant","message":{"content":[{"type":"text","text":"resumed"}]}}`)
	tailer.scan()
	state := srv.conversations.ListSession(context.Background(), sessionID)
	if len(state.Events) != 2 || state.Events[0].Text != "first" || state.Events[1].Text != "resumed" {
		t.Fatalf("checkpoint events = %+v", state.Events)
	}
}

func TestClaudeTailer_DeduplicatesHookDelivery(t *testing.T) {
	srv, sessionID, path := newClaudeTailerTestServer(t)
	if result := srv.AppendAssistant("same response", sessionID, "claude_hook"); !result.Appended {
		t.Fatalf("hook append failed: %+v", result)
	}
	writeClaudeTranscript(t, path, `{"type":"assistant","message":{"content":[{"type":"text","text":"same response"}]}}`)
	tailer := NewClaudeTailer(srv)
	tailer.scan()
	state := srv.conversations.ListSession(context.Background(), sessionID)
	if len(state.Events) != 1 || state.Events[0].Source != "claude_hook" {
		t.Fatalf("dedup events = %+v", state.Events)
	}
}

func TestClaudeTailer_DeduplicatesWhenTailerArrivesFirst(t *testing.T) {
	srv, sessionID, path := newClaudeTailerTestServer(t)
	writeClaudeTranscript(t, path, `{"type":"assistant","message":{"content":[{"type":"text","text":"same response"}]}}`)
	tailer := NewClaudeTailer(srv)
	tailer.scan()
	if result := srv.AppendAssistant("same response", sessionID, "claude_hook"); !result.Duplicate {
		t.Fatalf("hook should be duplicate after tailer delivery: %+v", result)
	}
	state := srv.conversations.ListSession(context.Background(), sessionID)
	if len(state.Events) != 1 || state.Events[0].Source != claudeTailerSource {
		t.Fatalf("dedup events = %+v", state.Events)
	}
}

func TestClaudeTailer_MissingTranscriptDoesNotCreateEvents(t *testing.T) {
	srv, sessionID, _ := newClaudeTailerTestServer(t)
	tailer := NewClaudeTailer(srv)
	tailer.scan()
	if state := srv.conversations.ListSession(context.Background(), sessionID); len(state.Events) != 0 {
		t.Fatalf("missing transcript emitted events: %+v", state.Events)
	}
}
