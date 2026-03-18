package main

import (
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func newTTSTestServer() *Server {
	return &Server{
		router:          mux.NewRouter(),
		sessions:        NewSessionManagerWithFactory(newFakePTYFactory()),
		events:          NewEventLogger(100),
		metrics:         NewMetrics(),
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

	fake := newFakePTYWithOutput()
	defer fake.Close()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	srv.sessions = sm

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	eventCh := sess.SubscribeConversation()
	defer sess.UnsubscribeConversation(eventCh)

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

	fake := newFakePTYWithOutput()
	defer fake.Close()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	srv.sessions = sm

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	eventCh := sess.SubscribeConversation()
	defer sess.UnsubscribeConversation(eventCh)

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
