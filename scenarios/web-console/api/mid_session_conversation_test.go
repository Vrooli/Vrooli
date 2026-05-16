package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
)

// newHookTestServer returns a minimally-wired Server suitable for testing
// the Claude Stop hook + conversation routing path. Replaces the old helper
// that lived in the deleted tts_hook_handler_test.go.
func newHookTestServer(token string) *Server {
	sm := newSessionManagerWithFactory(ptyfake.NewFactory())
	srv := &Server{
		router:          mux.NewRouter(),
		sessions:        sm,
		events:          events.NewLogger(100),
		metrics:         metrics.New(),
		hookAuthToken:   token,
		conversations:   NewConversationStore(),
		lastTTSBySource: map[string]conversationAppendSnapshot{},
		lastTTSAckBySrc: map[string]ttsAckSnapshot{},
		ttsHookConfigState: hookConfigState{
			cfg:  DefaultTTSHookConfig(),
			path: filepath.Join(os.TempDir(), "tts-hook-config-test.json"),
		},
		summarizeAutoPolicy: defaultSummarizeAutoPolicy(),
	}
	srv.fanouts = NewConversationFanoutRegistry().AttachToManager(sm)
	return srv
}

// TestCodexTailer_AttributesMidSessionRolloutToCorrectSession verifies that
// when a user starts a plain shell session and later runs `codex` inside it
// (as opposed to launching via the shortcut), the tailer picks up the
// resulting rollout file and routes the assistant event to the originating
// session — and only that session.
//
// The invariant this test locks in: attribution is bound to the filesystem
// path $CODEX_HOME/sessions/... that was injected into the PTY env at
// session creation time. Two concurrent sessions cannot bleed into each
// other because each gets its own CODEX_HOME.
//
// See docs/guides/CONVERSATION_TRACKING.md for the user-facing contract.
func TestCodexTailer_AttributesMidSessionRolloutToCorrectSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("codex tailer relies on Unix path semantics")
	}
	t.Setenv("HOME", t.TempDir())

	srv := &Server{
		router:        mux.NewRouter(),
		sessions:      newSessionManagerWithFactory(ptyfake.NewFactory()),
		conversations: NewConversationStore(),
		events:        events.NewLogger(100),
		metrics:       metrics.New(),
	}

	sessA, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session A: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(sessA.ID) })

	sessB, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session B: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(sessB.ID) })

	// Simulate codex CLI creating a rollout file inside session A's per-session
	// CODEX_HOME. The file starts empty so the tailer's "seek to EOF on open"
	// behavior leaves the ticker loop waiting for new bytes.
	now := time.Now()
	dateDir := filepath.Join(
		sessionCodexSessionsDir(sessA.ID),
		now.Format("2006"),
		now.Format("01"),
		now.Format("02"),
	)
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatalf("mkdir rollout dir: %v", err)
	}
	rolloutPath := filepath.Join(dateDir, "rollout-mid-session-test.jsonl")
	if f, err := os.Create(rolloutPath); err != nil {
		t.Fatalf("create empty rollout file: %v", err)
	} else {
		_ = f.Close()
	}

	tailer := NewCodexTailer(srv)
	tailer.scanForNewFiles()
	t.Cleanup(tailer.Stop)

	// Give the tailFile goroutine time to open the file and seek to EOF
	// before we append, so the content lands in the tail window (not before
	// the seek point, which would be skipped).
	time.Sleep(200 * time.Millisecond)

	line := []byte(`{"timestamp":"2025-06-01T12:00:00Z","type":"response_item","payload":{"role":"assistant","content":[{"type":"output_text","text":"Hello from mid-session codex"}]}}` + "\n")
	if err := appendFile(rolloutPath, line); err != nil {
		t.Fatalf("append rollout line: %v", err)
	}

	event := waitForFirstEvent(t, srv.conversations, sessA.ID, 3*time.Second)

	if event.Text != "Hello from mid-session codex" {
		t.Fatalf("expected text %q, got %q", "Hello from mid-session codex", event.Text)
	}
	if event.Source != "codex_tailer" {
		t.Fatalf("expected source codex_tailer, got %q", event.Source)
	}
	if event.Role != ConversationRoleAssistant {
		t.Fatalf("expected role assistant, got %v", event.Role)
	}
	if event.SessionID != sessA.ID {
		t.Fatalf("expected event session %s, got %s", sessA.ID, event.SessionID)
	}

	if stateB := srv.conversations.ListSession(sessB.ID); len(stateB.Events) != 0 {
		t.Fatalf("expected session B to have 0 events, got %d (bleed between sessions)", len(stateB.Events))
	}
}

// TestHandleHookStop_MidSessionClaudeAttribution verifies that a session
// created as a plain shell (no shortcut) still accepts a Claude Stop hook
// when the hook payload carries that session's WC_WEB_CONSOLE_SESSION_ID.
//
// This is the symmetric claude-side guarantee to the codex-side test above:
// the attribution path does not depend on how the session was launched,
// only on the env var that session.Create injects unconditionally at
// api/session.go:927-931.
func TestHandleHookStop_MidSessionClaudeAttribution(t *testing.T) {
	srv := newHookTestServer("secret-token")
	srv.conversations = NewConversationStore()

	// Plain-shell session (no shortcut command), matching how a user would
	// start bash and only later type `claude`.
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create plain session: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(sess.ID) })

	result := srv.AppendAssistant("assistant speaking mid-session", sess.ID, "claude_hook")
	if !result.Appended {
		t.Fatalf("expected event appended, got %+v", result)
	}
	if result.Code != "conversation_event_appended" {
		t.Fatalf("expected conversation_event_appended, got %q", result.Code)
	}

	state := srv.conversations.ListSession(sess.ID)
	if len(state.Events) != 1 {
		t.Fatalf("expected 1 event in session store, got %d", len(state.Events))
	}
	if state.Events[0].Text != "assistant speaking mid-session" {
		t.Fatalf("unexpected event text: %q", state.Events[0].Text)
	}
	if state.Events[0].Source != "claude_hook" {
		t.Fatalf("expected source claude_hook, got %q", state.Events[0].Source)
	}
}

// TestAppendConversationEvent_RejectsUnattributedPayload locks in the
// negative attribution guarantee: a Stop hook firing from a shell that
// never had WC_WEB_CONSOLE_SESSION_ID injected (external terminal, SSH'd
// remote host, pre-existing tmux server not managed by web-console) must
// not land an event in any session's store.
func TestAppendConversationEvent_RejectsUnattributedPayload(t *testing.T) {
	srv := newHookTestServer("secret-token")
	srv.conversations = NewConversationStore()

	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(sess.ID) })

	result := srv.AppendAssistant("stray assistant text", "", "claude_hook")
	if result.Appended {
		t.Fatalf("expected unattributed payload to be rejected, got %+v", result)
	}
	if result.Code != "conversation_target_missing" {
		t.Fatalf("expected conversation_target_missing, got %q", result.Code)
	}

	if state := srv.conversations.ListSession(sess.ID); len(state.Events) != 0 {
		t.Fatalf("expected existing session to have 0 events, got %d", len(state.Events))
	}
}

func appendFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func waitForFirstEvent(t *testing.T, store *ConversationStore, sessionID string, timeout time.Duration) ConversationEvent {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state := store.ListSession(sessionID)
		if len(state.Events) > 0 {
			return state.Events[0]
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for conversation event in session %s", timeout, sessionID)
	return ConversationEvent{}
}
