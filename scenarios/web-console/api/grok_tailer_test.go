package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"web-console/internal/sessionstore"
)

// waitForEventCount blocks until the session has at least n conversation
// events, returning the session state. Shared by the Grok/OpenCode ingestion
// tests where a turn produces a user + assistant pair.
func waitForEventCount(t *testing.T, store *ConversationStore, sessionID string, n int, timeout time.Duration) ConversationSessionState {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state := store.ListSession(context.Background(), sessionID)
		if len(state.Events) >= n {
			return state
		}
		time.Sleep(50 * time.Millisecond)
	}
	state := store.ListSession(context.Background(), sessionID)
	t.Fatalf("timed out waiting for %d events in session %s; have %d: %+v", n, sessionID, len(state.Events), state.Events)
	return state
}

// grokTurn writes one complete grok turn (user chunk, assistant chunk,
// turn_completed) to f. Mirrors the ACP shape grok appends to updates.jsonl.
func grokTurn(t *testing.T, f *os.File, sessionID, user, assistant string) {
	t.Helper()
	lines := []string{
		fmt.Sprintf(`{"timestamp":1,"method":"session/update","params":{"sessionId":%q,"update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":%q}}}}`, sessionID, user),
		fmt.Sprintf(`{"timestamp":2,"method":"session/update","params":{"sessionId":%q,"update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":%q}}}}`, sessionID, assistant),
		fmt.Sprintf(`{"timestamp":3,"method":"_x.ai/session/update","params":{"sessionId":%q,"update":{"sessionUpdate":"turn_completed","stop_reason":"end_turn"}}}`, sessionID),
	}
	for _, l := range lines {
		if _, err := f.WriteString(l + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	_ = f.Sync()
}

// newGrokTranscript creates the per-session grok transcript file at
// <sessionsDir>/<encoded-cwd>/<grok-session-id>/updates.jsonl and returns it.
func newGrokTranscript(t *testing.T, wcSessionID, grokSessionID string) (string, *os.File) {
	t.Helper()
	dir := filepath.Join(sessionGrokSessionsDir(wcSessionID), "encoded-cwd", grokSessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "updates.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return path, f
}

func TestGrokTailer_BackfillsCompletedTurnOnce(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())
	srv, sess := newCodexTailerTestServer(t)
	srv.agentCheckpointStore = NewInMemoryAgentTranscriptCheckpointStore()

	_, f := newGrokTranscript(t, sess.ID, "grok-1")
	grokTurn(t, f, "grok-1", "hello grok", "hi from grok")

	gt := NewGrokTailer(srv)
	gt.staleTimeout = 2 * time.Second
	gt.scanForNewFiles()
	t.Cleanup(gt.Stop)

	user := waitForFirstEvent(t, srv.conversations, sess.ID, 3*time.Second)
	if user.Text != "hello grok" || user.Role != ConversationRoleUser {
		t.Fatalf("expected user event, got %+v", user)
	}

	state := waitForEventCount(t, srv.conversations, sess.ID, 2, 3*time.Second)
	if state.Events[1].Text != "hi from grok" || state.Events[1].Source != grokSource {
		t.Fatalf("expected assistant grok event, got %+v", state.Events[1])
	}
}

func TestGrokTailer_ResumesFromCheckpointWithoutDuplicating(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())
	srv, sess := newCodexTailerTestServer(t)
	checkpoints := NewInMemoryAgentTranscriptCheckpointStore()
	srv.agentCheckpointStore = checkpoints

	path, f := newGrokTranscript(t, sess.ID, "grok-2")
	grokTurn(t, f, "grok-2", "old user", "old assistant")

	// Simulate a prior run that already consumed the first turn: checkpoint at
	// the current end-of-file (a turn boundary).
	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := checkpoints.Save(context.Background(), AgentTranscriptCheckpoint{
		Source:    grokSource,
		SourceKey: path,
		SessionID: sess.ID,
		Cursor:    strconv.FormatInt(stat.Size(), 10),
	}); err != nil {
		t.Fatal(err)
	}

	gt := NewGrokTailer(srv)
	gt.staleTimeout = 2 * time.Second
	gt.scanForNewFiles()
	t.Cleanup(gt.Stop)

	time.Sleep(200 * time.Millisecond)
	grokTurn(t, f, "grok-2", "new user", "new assistant")

	state := waitForEventCount(t, srv.conversations, sess.ID, 2, 3*time.Second)
	if len(state.Events) != 2 {
		t.Fatalf("expected exactly the resumed turn (2 events), got %d: %+v", len(state.Events), state.Events)
	}
	if state.Events[0].Text != "new user" || state.Events[1].Text != "new assistant" {
		t.Fatalf("resumed tailer should skip backlog, got %+v", state.Events)
	}
}

func TestGrokTailer_DoesNotConsumePartialTrailingLine(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())
	srv, sess := newCodexTailerTestServer(t)
	srv.agentCheckpointStore = NewInMemoryAgentTranscriptCheckpointStore()

	_, f := newGrokTranscript(t, sess.ID, "grok-3")
	grokTurn(t, f, "grok-3", "complete user", "complete assistant")
	// A partial line with no trailing newline: must not be parsed/emitted yet.
	if _, err := f.WriteString(`{"timestamp":9,"method":"session/update","params":{"sessionId":"grok-3","update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":"half`); err != nil {
		t.Fatal(err)
	}
	_ = f.Sync()

	gt := NewGrokTailer(srv)
	gt.staleTimeout = 2 * time.Second
	gt.scanForNewFiles()
	t.Cleanup(gt.Stop)

	// Wait for the complete turn, then give the partial line time to (wrongly)
	// surface before asserting it did not.
	waitForEventCount(t, srv.conversations, sess.ID, 2, 3*time.Second)
	time.Sleep(300 * time.Millisecond)
	state := srv.conversations.ListSession(context.Background(), sess.ID)
	if len(state.Events) != 2 {
		t.Fatalf("partial line must not produce an event; got %d: %+v", len(state.Events), state.Events)
	}
}

func TestGrokTailer_DuplicateReplayDoesNotDuplicate(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())
	srv, sess := newCodexTailerTestServer(t)
	srv.agentCheckpointStore = NewInMemoryAgentTranscriptCheckpointStore()

	_, f := newGrokTranscript(t, sess.ID, "grok-4")
	grokTurn(t, f, "grok-4", "dupe user", "dupe assistant")

	gt := NewGrokTailer(srv)
	gt.staleTimeout = 2 * time.Second
	gt.scanForNewFiles()
	t.Cleanup(gt.Stop)
	waitForEventCount(t, srv.conversations, sess.ID, 2, 3*time.Second)

	// A second tailer replaying from offset 0 (checkpoint lost) must not
	// re-append: the ConversationStore dedup is the second line of defense.
	srv.agentCheckpointStore = NewInMemoryAgentTranscriptCheckpointStore()
	gt2 := NewGrokTailer(srv)
	gt2.staleTimeout = 2 * time.Second
	gt2.scanForNewFiles()
	t.Cleanup(gt2.Stop)

	time.Sleep(400 * time.Millisecond)
	state := srv.conversations.ListSession(context.Background(), sess.ID)
	if len(state.Events) != 2 {
		t.Fatalf("replay must not duplicate; got %d events: %+v", len(state.Events), state.Events)
	}
}

func TestGrokTailer_CapturesAgentInfoFromSummary(t *testing.T) {
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())
	srv, sess := newCodexTailerTestServer(t)
	srv.agentCheckpointStore = NewInMemoryAgentTranscriptCheckpointStore()
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), sessionstore.Metadata{ID: sess.ID}); err != nil {
		t.Fatal(err)
	}
	srv.sessionStore = store

	path, f := newGrokTranscript(t, sess.ID, "grok-5")
	grokTurn(t, f, "grok-5", "u", "a")
	summary := `{"info":{"id":"grok-5","cwd":"/home/u/proj"}}`
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "summary.json"), []byte(summary), 0o644); err != nil {
		t.Fatal(err)
	}

	gt := NewGrokTailer(srv)
	gt.staleTimeout = 2 * time.Second
	gt.scanForNewFiles()
	t.Cleanup(gt.Stop)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		meta, err := store.Get(context.Background(), sess.ID)
		if err == nil && meta.AgentSessionID == "grok-5" {
			if meta.AgentType != sessionstore.AgentGrok {
				t.Fatalf("agent type = %q, want grok", meta.AgentType)
			}
			if meta.CWD != "/home/u/proj" {
				t.Fatalf("cwd = %q", meta.CWD)
			}
			if meta.LastRolloutPath != path {
				t.Fatalf("rollout path = %q, want %q", meta.LastRolloutPath, path)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("agent info was not captured from summary.json")
}
