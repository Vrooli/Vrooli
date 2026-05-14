package main

import (
	"testing"
	"time"

	"github.com/gorilla/mux"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
)

func newTTSTestServer() *Server {
	sm := newSessionManagerWithFactory(ptyfake.NewFactory())
	fanouts := NewConversationFanoutRegistry().AttachToManager(sm)
	return &Server{
		router:          mux.NewRouter(),
		sessions:        sm,
		fanouts:         fanouts,
		events:          events.NewLogger(100),
		metrics:         metrics.New(),
		workspace:       NewMemWorkspaceStore(),
		conversations:   NewConversationStore(),
		lastTTSBySource: make(map[string]conversationAppendSnapshot),
		lastTTSAckBySrc: make(map[string]ttsAckSnapshot),
	}
}

func TestAppendConversationEvent_TargetMissing(t *testing.T) {
	srv := newTTSTestServer()

	result := srv.appendConversationEvent("some text", "", "test")
	if result.Appended {
		t.Fatal("expected append to fail without a mapped target")
	}
	if result.Code != "conversation_target_missing" {
		t.Fatalf("expected conversation_target_missing, got %s", result.Code)
	}
}

func TestAppendConversationEvent_RoutesToMappedSession(t *testing.T) {
	srv := newTTSTestServer()

	fake := ptyfake.NewFakePTYWithOutput()
	defer fake.Close()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))
	srv.sessions = sm
	srv.fanouts = NewConversationFanoutRegistry().AttachToManager(sm)

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	fanout := srv.fanouts.Get(sess.ID)
	eventCh := fanout.Subscribe()
	defer fanout.Unsubscribe(eventCh)

	result := srv.appendConversationEvent("The answer is 42", sess.ID, "test")
	if !result.Appended {
		t.Fatalf("expected append to succeed, got %+v", result)
	}
	if result.Code != "conversation_event_appended" {
		t.Fatalf("expected conversation_event_appended, got %s", result.Code)
	}

	select {
	case event := <-eventCh:
		if event.Text != "The answer is 42" {
			t.Fatalf("expected routed text, got %q", event.Text)
		}
		if event.SessionID != sess.ID {
			t.Fatalf("expected session %s, got %s", sess.ID, event.SessionID)
		}
		if event.ID == "" {
			t.Fatal("expected event id to be populated")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for conversation event")
	}
}

func TestAppendConversationEvent_DeduplicatesByEventIdentity(t *testing.T) {
	srv := newTTSTestServer()

	fake := ptyfake.NewFakePTYWithOutput()
	defer fake.Close()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))
	srv.sessions = sm
	srv.fanouts = NewConversationFanoutRegistry().AttachToManager(sm)

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	fanout := srv.fanouts.Get(sess.ID)
	eventCh := fanout.Subscribe()
	defer fanout.Unsubscribe(eventCh)

	first := srv.appendConversationEvent("duplicate me", sess.ID, "test")
	second := srv.appendConversationEvent("duplicate me", sess.ID, "test")
	if !first.Appended {
		t.Fatalf("expected first append to succeed, got %+v", first)
	}
	if !second.Appended || !second.Duplicate || second.Code != "conversation_duplicate" {
		t.Fatalf("expected duplicate result, got %+v", second)
	}

	select {
	case <-eventCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first conversation event")
	}

	select {
	case <-eventCh:
		t.Fatal("did not expect duplicate event to be republished")
	case <-time.After(100 * time.Millisecond):
	}
}
