package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestTailer creates a CodexTailer with a temp baseDir.
// Returns the tailer and the base dir path.
func newTestTailer(t *testing.T) (*CodexTailer, string) {
	t.Helper()
	baseDir := t.TempDir()

	ct := &CodexTailer{
		baseDir:  baseDir,
		watchers: make(map[string]struct{}),
		stopCh:   make(chan struct{}),
	}
	return ct, baseDir
}

// writeRolloutLine writes a single JSONL line to the given file.
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

func TestCodexTailer_ScanDetectsNewRolloutFile(t *testing.T) {
	ct, baseDir := newTestTailer(t)

	// Create today's date directory with a rollout file.
	now := time.Now()
	dateDir := filepath.Join(baseDir, now.Format("2006"), now.Format("01"), now.Format("02"))
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
	_, found := ct.watchers[rolloutPath]
	ct.mu.Unlock()
	if !found {
		t.Error("expected rollout file to be tracked in watchers")
	}
	// Wait for the goroutine spawned by scanForNewFiles to finish.
	close(ct.stopCh)
	ct.wg.Wait()
}

func TestCodexTailer_IgnoresNonRolloutFiles(t *testing.T) {
	ct, baseDir := newTestTailer(t)

	now := time.Now()
	dateDir := filepath.Join(baseDir, now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create files that should NOT be picked up.
	for _, name := range []string{"session-abc.jsonl", "rollout-abc.txt", "notes.jsonl"} {
		f, err := os.Create(filepath.Join(dateDir, name))
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	ct.scanForNewFiles()

	ct.mu.Lock()
	count := len(ct.watchers)
	ct.mu.Unlock()
	if count != 0 {
		t.Errorf("expected 0 watchers for non-rollout files, got %d", count)
	}
}

func TestCodexTailer_GracefulShutdown(t *testing.T) {
	ct, baseDir := newTestTailer(t)

	now := time.Now()
	dateDir := filepath.Join(baseDir, now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a rollout file so a tail goroutine gets spawned.
	rolloutPath := filepath.Join(dateDir, "rollout-shutdown.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	ct.scanForNewFiles()

	// Stop should return without hanging.
	done := make(chan struct{})
	go func() {
		ct.Stop()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds")
	}
}

func TestCodexTailer_WatcherCleanupOnStop(t *testing.T) {
	ct, baseDir := newTestTailer(t)

	now := time.Now()
	dateDir := filepath.Join(baseDir, now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a rollout file to spawn a tail goroutine.
	rolloutPath := filepath.Join(dateDir, "rollout-cleanup.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	ct.scanForNewFiles()

	// Verify watcher is tracked.
	ct.mu.Lock()
	_, found := ct.watchers[rolloutPath]
	ct.mu.Unlock()
	if !found {
		t.Fatal("expected rollout file to be tracked")
	}

	// Stop should clean up watchers via the defer in tailFile.
	ct.Stop()

	ct.mu.Lock()
	count := len(ct.watchers)
	ct.mu.Unlock()
	if count != 0 {
		t.Errorf("expected watchers map to be empty after Stop, got %d entries", count)
	}
}

func TestExtractAssistantText_Integration_NewLines(t *testing.T) {
	// Simulate writing lines to a file and reading them back via
	// ExtractAssistantText (the parser, not the tailer).
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-integ.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	// Write an assistant line.
	writeRolloutLine(t, f, "response_item", ResponsePayload{
		Role:    "assistant",
		Content: []ContentItem{{Type: "output_text", Text: "TTS this"}},
	})
	// Write a non-assistant line.
	writeRolloutLine(t, f, "event_msg", map[string]string{"message": "ignore"})

	f.Close()

	// Read back and verify.
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
		t.Errorf("expected [\"TTS this\"], got %v", texts)
	}
}

// TestCodexTailer_E2E_DeliverTTS verifies the full pipeline: write a rollout
// JSONL line → tailer detects it → extracts text → deliverTTS → TTS subscriber
// receives the message.
func TestCodexTailer_E2E_DeliverTTS(t *testing.T) {
	// Build a real Server with session + matching output history.
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: true}

	fake := newFakePTYWithOutput()
	defer fake.Close()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	srv.sessions = sm

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	// Set as active pane.
	_ = srv.workspace.UpsertPane(&WorkspacePane{SessionID: sess.ID, Name: "test"})
	_ = srv.workspace.SavePaneOrder(sess.ID, []string{sess.ID})

	// Put the expected TTS text in the session's output buffer.
	const ttsText = "Hello from the tailer"
	sess.mu.Lock()
	sess.outputHistory = []byte("prefix " + ttsText + " suffix")
	sess.mu.Unlock()

	// Subscribe to TTS on this session.
	ttsCh := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ttsCh)

	// Create the tailer with a temp base directory.
	baseDir := t.TempDir()
	ct := &CodexTailer{
		server:   srv,
		baseDir:  baseDir,
		watchers: make(map[string]struct{}),
		stopCh:   make(chan struct{}),
	}

	// Create today's date directory with a rollout file.
	now := time.Now()
	dateDir := filepath.Join(baseDir, now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-e2e.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}

	// Start tailing (seeks to end, so anything written before this is skipped).
	ct.scanForNewFiles()

	// Give the tail goroutine a moment to open and seek the file.
	time.Sleep(200 * time.Millisecond)

	// Now write a rollout line with assistant text.
	writeRolloutLine(t, f, "response_item", ResponsePayload{
		Role:    "assistant",
		Content: []ContentItem{{Type: "output_text", Text: ttsText}},
	})
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	// Wait for TTS delivery.
	select {
	case msg := <-ttsCh:
		if msg != ttsText {
			t.Errorf("expected %q, got %q", ttsText, msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for TTS message from CodexTailer")
	}

	ct.Stop()
}

// TestCodexTailer_E2E_MalformedLineSkipped verifies that a corrupt JSONL line
// does not prevent delivery of subsequent valid assistant lines.
func TestCodexTailer_E2E_MalformedLineSkipped(t *testing.T) {
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: true}

	fake := newFakePTYWithOutput()
	defer fake.Close()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	srv.sessions = sm

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	_ = srv.workspace.UpsertPane(&WorkspacePane{SessionID: sess.ID, Name: "test"})
	_ = srv.workspace.SavePaneOrder(sess.ID, []string{sess.ID})

	const ttsText = "Valid after corrupt"
	sess.mu.Lock()
	sess.outputHistory = []byte("prefix " + ttsText + " suffix")
	sess.mu.Unlock()

	ttsCh := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ttsCh)

	baseDir := t.TempDir()
	ct := &CodexTailer{
		server:   srv,
		baseDir:  baseDir,
		watchers: make(map[string]struct{}),
		stopCh:   make(chan struct{}),
	}

	now := time.Now()
	dateDir := filepath.Join(baseDir, now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-malformed.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}

	ct.scanForNewFiles()
	time.Sleep(200 * time.Millisecond)

	// Write a corrupt line first
	if _, err := f.Write([]byte("{broken json\n")); err != nil {
		t.Fatal(err)
	}
	// Then write a valid assistant line
	writeRolloutLine(t, f, "response_item", ResponsePayload{
		Role:    "assistant",
		Content: []ContentItem{{Type: "output_text", Text: ttsText}},
	})
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-ttsCh:
		if msg != ttsText {
			t.Errorf("expected %q, got %q", ttsText, msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out — malformed line prevented delivery of valid line")
	}

	ct.Stop()
}

func TestCodexTailer_StaleTimeout(t *testing.T) {
	ct, baseDir := newTestTailer(t)
	ct.staleTimeout = 100 * time.Millisecond

	// Create today's date directory with a rollout file.
	now := time.Now()
	dateDir := filepath.Join(baseDir, now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-stale.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	ct.scanForNewFiles()

	// Wait long enough for the stale timeout to fire.
	time.Sleep(150 * time.Millisecond)

	// The watcher should have cleaned itself from the map.
	ct.mu.Lock()
	count := len(ct.watchers)
	ct.mu.Unlock()
	if count != 0 {
		t.Errorf("expected watchers map to be empty after stale timeout, got %d entries", count)
	}

	// Clean up any remaining goroutines.
	close(ct.stopCh)
	ct.wg.Wait()
}

func TestCodexTailer_StopDuringActiveTail(t *testing.T) {
	ct, baseDir := newTestTailer(t)
	ct.staleTimeout = 1 * time.Hour // long timeout so it won't expire

	// Create today's date directory with a rollout file.
	now := time.Now()
	dateDir := filepath.Join(baseDir, now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-active.jsonl")
	f, err := os.Create(rolloutPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	ct.scanForNewFiles()

	// Give the tail goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	// Stop() should return quickly even with a long stale timeout.
	done := make(chan struct{})
	go func() {
		ct.Stop()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return within 2 seconds during active tail")
	}
}

// splitLines splits data on newlines, skipping empty entries.
func splitLines(data []byte) [][]byte {
	var result [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			line := data[start:i]
			if len(line) > 0 {
				result = append(result, line)
			}
			start = i + 1
		}
	}
	if start < len(data) {
		result = append(result, data[start:])
	}
	return result
}
