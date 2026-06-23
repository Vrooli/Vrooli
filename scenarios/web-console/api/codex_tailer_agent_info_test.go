package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"web-console/internal/backend"
	"web-console/internal/sessionstore"
)

func TestCodexTailer_PopulatesAgentInfoOnFirstRollout(t *testing.T) {
	useIsolatedSessionState(t)
	srv, sess := newCodexTailerTestServer(t)
	srv.sessionStore = sessionstore.NewInMemory()
	srv.sessions.SetStore(srv.sessionStore)
	if err := srv.sessionStore.Save(sessionstore.Metadata{
		ID:       sess.ID,
		Backend:  backend.Persistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: true,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	now := time.Now()
	dateDir := filepath.Join(sessionCodexSessionsDir(sess.ID), now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-meta-test.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	// First line: codex's session_meta with the codex session id and cwd.
	meta := struct {
		Timestamp string                 `json:"timestamp"`
		Type      string                 `json:"type"`
		Payload   map[string]interface{} `json:"payload"`
	}{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      "session_meta",
		Payload: map[string]interface{}{
			"id":  "codex-uuid-from-rollout",
			"cwd": "/some/project",
		},
	}
	encoded, _ := json.Marshal(meta)
	encoded = append(encoded, '\n')
	if _, err := f.Write(encoded); err != nil {
		t.Fatal(err)
	}
	f.Close()

	ct := NewCodexTailer(srv)
	ct.captureAgentInfo(rolloutPath, sess.ID)

	got, err := srv.sessionStore.Get(sess.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.AgentType != sessionstore.AgentCodex {
		t.Errorf("agent_type: got %q want codex", got.AgentType)
	}
	if got.AgentSessionID != "codex-uuid-from-rollout" {
		t.Errorf("agent_session_id: got %q", got.AgentSessionID)
	}
	if got.CWD != "/some/project" {
		t.Errorf("cwd: got %q", got.CWD)
	}
	if got.LastRolloutPath != rolloutPath {
		t.Errorf("last_rollout_path: got %q", got.LastRolloutPath)
	}
}

func TestCodexTailer_AgentInfoIgnoresNonSessionMeta(t *testing.T) {
	useIsolatedSessionState(t)
	srv, sess := newCodexTailerTestServer(t)
	srv.sessionStore = sessionstore.NewInMemory()
	srv.sessions.SetStore(srv.sessionStore)
	if err := srv.sessionStore.Save(sessionstore.Metadata{
		ID:       sess.ID,
		Backend:  backend.Persistent,
		Shell:    "/bin/bash",
		Cols:     80,
		Rows:     24,
		Created:  time.Now(),
		Detached: true,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "rollout-bogus.jsonl")
	if err := os.WriteFile(tmp, []byte(`{"type":"output_text","payload":{"text":"hello"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ct := NewCodexTailer(srv)
	ct.captureAgentInfo(tmp, sess.ID)

	got, _ := srv.sessionStore.Get(sess.ID)
	if got.AgentType != sessionstore.AgentNone {
		t.Errorf("agent_type changed despite non-session_meta first line: %q", got.AgentType)
	}
}
