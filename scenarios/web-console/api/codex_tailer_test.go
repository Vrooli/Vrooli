package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"web-console/backends/codex"
	"web-console/internal/ptyfake"
	"web-console/session"

	"github.com/gorilla/mux"

	intevents "web-console/internal/events"
	intmetrics "web-console/internal/metrics"
)

func splitLines(data []byte) [][]byte {
	raw := bytes.Split(data, []byte{'\n'})
	lines := make([][]byte, 0, len(raw))
	for _, line := range raw {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func writeRolloutLine(t *testing.T, f *os.File, lineType string, payload interface{}) {
	t.Helper()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	rl := codex.RolloutLine{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      lineType,
		Payload:   json.RawMessage(payloadBytes),
	}
	data, err := json.Marshal(rl)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
}

func newCodexTailerTestServer(t *testing.T) (*Server, *session.Session) {
	t.Helper()
	fake := ptyfake.NewFakePTYWithOutput()
	t.Cleanup(func() { _ = fake.Close() })
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))
	srv := &Server{
		router:          mux.NewRouter(),
		sessions:        sm,
		events:          intevents.NewLogger(100),
		metrics:         intmetrics.New(),
		conversations:   NewConversationStore(),
		lastTTSBySource: map[string]conversationAppendSnapshot{},
		lastTTSAckBySrc: map[string]ttsAckSnapshot{},
		ttsHookConfigState: hookConfigState{
			cfg:  TTSHookConfig{AutoEnabled: true, Backend: "auto"},
			path: filepath.Join(t.TempDir(), "tts-hook-config.json"),
		},
		summarizeAutoPolicy: defaultSummarizeAutoPolicy(),
	}
	srv.hub = NewConversationHub()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(sess.ID) })
	return srv, sess
}

func TestCodexTailer_ScanDetectsPerSessionRolloutFile(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)
	ct := NewCodexTailer(srv)

	now := time.Now()
	dateDir := filepath.Join(sessionCodexSessionsDir(sess.ID), now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-test123.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	ct.scanForNewFiles()
	ct.mu.Lock()
	target, found := ct.watchers[rolloutPath]
	ct.mu.Unlock()
	if !found {
		t.Fatal("expected rollout file to be tracked")
	}
	if target != sess.ID {
		t.Fatalf("expected watcher to map to %s, got %s", sess.ID, target)
	}
	ct.Stop()
}

func TestExtractAssistantText_Integration_NewLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-integ.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	writeRolloutLine(t, f, "response_item", codex.ResponsePayload{
		Role:    "assistant",
		Content: []codex.ContentItem{{Type: "output_text", Text: "TTS this"}},
	})
	writeRolloutLine(t, f, "event_msg", map[string]string{"message": "ignore"})
	f.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := splitLines(data)
	var texts []string
	for _, line := range lines {
		if txt := codex.ExtractAssistantText(line); txt != "" {
			texts = append(texts, txt)
		}
	}
	if len(texts) != 1 || texts[0] != "TTS this" {
		t.Fatalf("expected [\"TTS this\"], got %v", texts)
	}
}

func TestCodexTailer_E2E_RoutesToOwningSession(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)

	sub, _, _ := srv.hub.Subscribe(0)
	defer srv.hub.Unsubscribe(sub)

	ct := NewCodexTailer(srv)
	ct.staleTimeout = 2 * time.Second

	now := time.Now()
	dateDir := filepath.Join(sessionCodexSessionsDir(sess.ID), now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-e2e.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	ct.scanForNewFiles()
	time.Sleep(200 * time.Millisecond)

	writeRolloutLine(t, f, "response_item", codex.ResponsePayload{
		Role:    "assistant",
		Content: []codex.ContentItem{{Type: "output_text", Text: "Hello from the tailer"}},
	})
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	select {
	case env := <-sub.events:
		if env.SessionID != sess.ID {
			t.Fatalf("expected session %s, got %s", sess.ID, env.SessionID)
		}
		payload, ok := env.Payload.(conversationEventPayload)
		if !ok {
			t.Fatalf("expected conversationEventPayload, got %T", env.Payload)
		}
		if payload.Text != "Hello from the tailer" {
			t.Fatalf("expected routed text, got %q", payload.Text)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for conversation event from CodexTailer")
	}

	ct.Stop()
}

func TestCodexTailer_BackfillsExistingRolloutContentWithoutCheckpoint(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)

	now := time.Now()
	dateDir := filepath.Join(sessionCodexSessionsDir(sess.ID), now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-backfill.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	writeRolloutLine(t, f, "response_item", codex.ResponsePayload{
		Role:    "assistant",
		Content: []codex.ContentItem{{Type: "output_text", Text: "Backfilled assistant message"}},
	})
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	ct := NewCodexTailer(srv)
	ct.staleTimeout = 2 * time.Second
	ct.scanForNewFiles()
	t.Cleanup(ct.Stop)

	event := waitForFirstEvent(t, srv.conversations, sess.ID, 3*time.Second)
	if event.Text != "Backfilled assistant message" {
		t.Fatalf("expected backfilled text, got %q", event.Text)
	}
}

func TestCodexTailer_ResumesFromCheckpointOffset(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)
	checkpoints := NewInMemoryCodexCheckpointStore()
	srv.codexCheckpointStore = checkpoints

	now := time.Now()
	dateDir := filepath.Join(sessionCodexSessionsDir(sess.ID), now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-resume.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	writeRolloutLine(t, f, "response_item", codex.ResponsePayload{
		Role:    "assistant",
		Content: []codex.ContentItem{{Type: "output_text", Text: "Old assistant message"}},
	})
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := checkpoints.Save(CodexRolloutCheckpoint{
		Path:      rolloutPath,
		SessionID: sess.ID,
		Offset:    stat.Size(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	ct := NewCodexTailer(srv)
	ct.staleTimeout = 2 * time.Second
	ct.scanForNewFiles()
	t.Cleanup(ct.Stop)

	time.Sleep(200 * time.Millisecond)

	writeRolloutLine(t, f, "response_item", codex.ResponsePayload{
		Role:    "assistant",
		Content: []codex.ContentItem{{Type: "output_text", Text: "New assistant message"}},
	})
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	event := waitForFirstEvent(t, srv.conversations, sess.ID, 3*time.Second)
	if event.Text != "New assistant message" {
		t.Fatalf("expected resumed tailer to skip old backlog and read new text, got %q", event.Text)
	}

	state := srv.conversations.ListSession(sess.ID)
	if len(state.Events) != 1 {
		t.Fatalf("expected exactly one resumed event, got %d", len(state.Events))
	}

	checkpoint, ok, err := checkpoints.Get(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected checkpoint to be saved after reading new rollout content")
	}
	if checkpoint.Offset <= stat.Size() {
		t.Fatalf("expected checkpoint offset to advance beyond %d, got %d", stat.Size(), checkpoint.Offset)
	}
}
