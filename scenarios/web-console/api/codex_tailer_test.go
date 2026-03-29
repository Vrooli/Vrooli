package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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
	rl := RolloutLine{
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

func newCodexTailerTestServer(t *testing.T) (*Server, *Session) {
	t.Helper()
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: true}

	fake := newFakePTYWithOutput()
	t.Cleanup(func() { _ = fake.Close() })
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	srv.sessions = sm

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

	writeRolloutLine(t, f, "response_item", ResponsePayload{
		Role:    "assistant",
		Content: []ContentItem{{Type: "output_text", Text: "TTS this"}},
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
		if txt := ExtractAssistantText(line); txt != "" {
			texts = append(texts, txt)
		}
	}
	if len(texts) != 1 || texts[0] != "TTS this" {
		t.Fatalf("expected [\"TTS this\"], got %v", texts)
	}
}

func TestCodexTailer_E2E_RoutesToOwningSession(t *testing.T) {
	srv, sess := newCodexTailerTestServer(t)

	eventCh := sess.SubscribeConversation()
	defer sess.UnsubscribeConversation(eventCh)

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

	writeRolloutLine(t, f, "response_item", ResponsePayload{
		Role:    "assistant",
		Content: []ContentItem{{Type: "output_text", Text: "Hello from the tailer"}},
	})
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-eventCh:
		if event.Text != "Hello from the tailer" {
			t.Fatalf("expected routed text, got %q", event.Text)
		}
		if event.SessionID != sess.ID {
			t.Fatalf("expected session %s, got %s", sess.ID, event.SessionID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for conversation event from CodexTailer")
	}

	ct.Stop()
}
