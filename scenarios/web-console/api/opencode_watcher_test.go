package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"web-console/backends/opencode"
	"web-console/internal/sessionstore"
)

// fakeOpenCodeClient is a programmable opencode.Client for watcher tests.
type fakeOpenCodeClient struct {
	mu       sync.Mutex
	sessions []opencode.Session
	messages map[string][]opencode.MessageWithParts
	// eventsFn, if set, drives the SSE seam; nil blocks until ctx cancel.
	eventsFn   func(ctx context.Context, onEvent func(opencode.Event)) error
	listCalls  int
	eventCalls int
}

func (f *fakeOpenCodeClient) ListSessions(_ context.Context) ([]opencode.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	return append([]opencode.Session(nil), f.sessions...), nil
}

func (f *fakeOpenCodeClient) SessionMessages(_ context.Context, sessionID string) ([]opencode.MessageWithParts, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]opencode.MessageWithParts(nil), f.messages[sessionID]...), nil
}

func (f *fakeOpenCodeClient) Events(ctx context.Context, onEvent func(opencode.Event)) error {
	f.mu.Lock()
	f.eventCalls++
	fn := f.eventsFn
	f.mu.Unlock()
	if fn != nil {
		return fn(ctx, onEvent)
	}
	<-ctx.Done()
	return ctx.Err()
}

// ocSession builds a session stamped as created now (after the test pane).
func ocSession(id, dir string) opencode.Session {
	s := opencode.Session{ID: id, Directory: dir}
	s.Time.Created = time.Now().UnixMilli()
	return s
}

func ocUser(created int64, text string) opencode.MessageWithParts {
	m := opencode.MessageWithParts{Parts: []opencode.Part{{Type: "text", Text: text}}}
	m.Info.Role = "user"
	m.Info.Time.Created = created
	return m
}

func ocAssistant(created, completed int64, text string) opencode.MessageWithParts {
	m := opencode.MessageWithParts{Parts: []opencode.Part{{Type: "text", Text: text}}}
	m.Info.Role = "assistant"
	m.Info.Time.Created = created
	m.Info.Time.Completed = completed
	return m
}

// newOpenCodeWatcherTest wires a watcher with an in-memory checkpoint store and
// a sessionStore seeded with one live opencode pane (CWD /work).
func newOpenCodeWatcherTest(t *testing.T) (*Server, *OpenCodeWatcher, string) {
	t.Helper()
	srv, sess := newCodexTailerTestServer(t)
	srv.agentCheckpointStore = NewInMemoryAgentTranscriptCheckpointStore()
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), sessionstore.Metadata{
		ID:        sess.ID,
		AgentType: sessionstore.AgentOpenCode,
		CWD:       "/work",
		Created:   time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	srv.sessionStore = store
	w := &OpenCodeWatcher{
		server:      srv,
		checkpoints: srv.agentCheckpointStore,
		claimed:     map[string]string{},
	}
	return srv, w, sess.ID
}

func TestOpenCodeWatcher_BackfillAttributesAndReconciles(t *testing.T) {
	srv, w, paneID := newOpenCodeWatcherTest(t)
	client := &fakeOpenCodeClient{
		sessions: []opencode.Session{ocSession("ses_a", "/work")},
		messages: map[string][]opencode.MessageWithParts{
			"ses_a": {ocUser(10, "hi"), ocAssistant(20, 30, "hello back")},
		},
	}

	w.reconcileAll(context.Background(), client)

	state := srv.conversations.ListSession(context.Background(), paneID)
	if len(state.Events) != 2 {
		t.Fatalf("expected 2 events, got %d: %+v", len(state.Events), state.Events)
	}
	if state.Events[0].Source != opencodeSource || state.Events[0].Text != "hi" {
		t.Fatalf("user event wrong: %+v", state.Events[0])
	}
	if state.Events[1].Text != "hello back" {
		t.Fatalf("assistant event wrong: %+v", state.Events[1])
	}
	// Agent identity persisted for recovery.
	meta, _ := srv.sessionStore.Get(context.Background(), paneID)
	if meta.AgentSessionID != "ses_a" {
		t.Fatalf("expected agent session id ses_a, got %q", meta.AgentSessionID)
	}
}

func TestOpenCodeWatcher_ReconcileIsIdempotent(t *testing.T) {
	srv, w, paneID := newOpenCodeWatcherTest(t)
	client := &fakeOpenCodeClient{
		sessions: []opencode.Session{ocSession("ses_a", "/work")},
		messages: map[string][]opencode.MessageWithParts{
			"ses_a": {ocUser(10, "hi"), ocAssistant(20, 30, "hello")},
		},
	}
	ctx := context.Background()
	w.reconcileAll(ctx, client)
	w.reconcileAll(ctx, client)
	w.reconcileAll(ctx, client)

	state := srv.conversations.ListSession(context.Background(), paneID)
	if len(state.Events) != 2 {
		t.Fatalf("re-running reconciliation must not duplicate; got %d: %+v", len(state.Events), state.Events)
	}
}

func TestOpenCodeWatcher_FiltersByDirectory(t *testing.T) {
	srv, w, paneID := newOpenCodeWatcherTest(t)
	client := &fakeOpenCodeClient{
		// Session in a DIFFERENT directory than the pane's cwd.
		sessions: []opencode.Session{ocSession("ses_other", "/somewhere/else")},
		messages: map[string][]opencode.MessageWithParts{
			"ses_other": {ocUser(10, "not mine")},
		},
	}
	w.reconcileAll(context.Background(), client)

	if state := srv.conversations.ListSession(context.Background(), paneID); len(state.Events) != 0 {
		t.Fatalf("session from another directory must not be attributed; got %+v", state.Events)
	}
}

func TestOpenCodeWatcher_SkipsAmbiguousSameDirCandidates(t *testing.T) {
	srv, w, paneID := newOpenCodeWatcherTest(t)
	// Add a SECOND opencode pane in the same cwd → a single session matches both
	// panes and must not be routed to either.
	store := srv.sessionStore
	if err := store.Save(context.Background(), sessionstore.Metadata{
		ID:        "pane-2",
		AgentType: sessionstore.AgentOpenCode,
		CWD:       "/work",
		Created:   time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	client := &fakeOpenCodeClient{
		sessions: []opencode.Session{ocSession("ses_a", "/work")},
		messages: map[string][]opencode.MessageWithParts{
			"ses_a": {ocUser(10, "ambiguous")},
		},
	}
	w.reconcileAll(context.Background(), client)

	if state := srv.conversations.ListSession(context.Background(), paneID); len(state.Events) != 0 {
		t.Fatalf("ambiguous candidate must not be routed; got %+v", state.Events)
	}
	if state := srv.conversations.ListSession(context.Background(), "pane-2"); len(state.Events) != 0 {
		t.Fatalf("ambiguous candidate must not be routed; got %+v", state.Events)
	}
}

func TestOpenCodeWatcher_ReconnectsAndReconciles(t *testing.T) {
	srv, _, paneID := newOpenCodeWatcherTest(t)

	client := &fakeOpenCodeClient{
		sessions: []opencode.Session{ocSession("ses_a", "/work")},
		messages: map[string][]opencode.MessageWithParts{
			"ses_a": {ocUser(10, "hi"), ocAssistant(20, 30, "hello")},
		},
	}
	// First Events call returns immediately (simulating a dropped stream); the
	// run loop must reconnect and call Events again.
	var once sync.Once
	client.eventsFn = func(ctx context.Context, _ func(opencode.Event)) error {
		var first bool
		once.Do(func() { first = true })
		if first {
			return context.DeadlineExceeded // stream dropped
		}
		<-ctx.Done()
		return ctx.Err()
	}

	w := &OpenCodeWatcher{
		server:            srv,
		checkpoints:       srv.agentCheckpointStore,
		claimed:           map[string]string{},
		startServer:       func() (string, func(), error) { return "http://127.0.0.1:0", func() {}, nil },
		newClient:         func(string) opencode.Client { return client },
		reconcileInterval: 50 * time.Millisecond,
		reconnectBackoff:  20 * time.Millisecond,
		stopCh:            make(chan struct{}),
	}
	w.Start()
	t.Cleanup(w.Stop)

	state := waitForEventCount(t, srv.conversations, paneID, 2, 3*time.Second)
	if state.Events[1].Text != "hello" {
		t.Fatalf("unexpected events after reconnect: %+v", state.Events)
	}
	client.mu.Lock()
	calls := client.eventCalls
	client.mu.Unlock()
	if calls < 2 {
		t.Fatalf("expected at least one reconnect (>=2 Events calls), got %d", calls)
	}
}

func TestOpenCodeWatcher_RestoresClaimsAfterRestart(t *testing.T) {
	srv, _, paneID := newOpenCodeWatcherTest(t)
	// Simulate a pane already attributed before restart.
	if err := srv.sessionStore.UpdateAgentInfo(context.Background(), paneID, sessionstore.AgentInfo{
		AgentType:      sessionstore.AgentOpenCode,
		AgentSessionID: "ses_a",
	}); err != nil {
		t.Fatal(err)
	}
	w := &OpenCodeWatcher{
		server:      srv,
		checkpoints: srv.agentCheckpointStore,
		claimed:     map[string]string{},
	}
	w.loadExistingClaims()

	client := &fakeOpenCodeClient{
		sessions: []opencode.Session{ocSession("ses_a", "/work")},
		messages: map[string][]opencode.MessageWithParts{
			"ses_a": {ocUser(10, "resumed"), ocAssistant(20, 30, "after restart")},
		},
	}
	w.reconcileAll(context.Background(), client)

	state := srv.conversations.ListSession(context.Background(), paneID)
	if len(state.Events) != 2 || state.Events[0].Text != "resumed" {
		t.Fatalf("restored claim should reconcile existing session; got %+v", state.Events)
	}
}
