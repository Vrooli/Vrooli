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
		ttsDedup:        newTTSDedup(),
		lastTTSBySource: make(map[string]ttsRoutingSnapshot),
		lastTTSAckBySrc: make(map[string]ttsAckSnapshot),
	}
}

func TestRouteTTSCandidate_Disabled(t *testing.T) {
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: false}

	result := srv.routeTTSCandidate("some text", "s1", "test")
	if result.Routed {
		t.Fatal("expected routing to be skipped when auto-TTS is disabled")
	}
	if result.Code != "tts_auto_disabled" {
		t.Fatalf("expected tts_auto_disabled, got %s", result.Code)
	}
}

func TestRouteTTSCandidate_TargetMissing(t *testing.T) {
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: true}

	result := srv.routeTTSCandidate("some text", "", "test")
	if result.Routed {
		t.Fatal("expected routing to fail without a mapped target")
	}
	if result.Code != "tts_target_missing" {
		t.Fatalf("expected tts_target_missing, got %s", result.Code)
	}
}

func TestRouteTTSCandidate_RoutesToMappedSession(t *testing.T) {
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

	ttsCh := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ttsCh)

	result := srv.routeTTSCandidate("The answer is 42", sess.ID, "test")
	if !result.Routed {
		t.Fatalf("expected routing to succeed, got %+v", result)
	}
	if result.Code != "tts_candidate_routed" {
		t.Fatalf("expected tts_candidate_routed, got %s", result.Code)
	}

	select {
	case candidate := <-ttsCh:
		if candidate.Text != "The answer is 42" {
			t.Fatalf("expected routed text, got %q", candidate.Text)
		}
		if candidate.SessionID != sess.ID {
			t.Fatalf("expected session %s, got %s", sess.ID, candidate.SessionID)
		}
		if candidate.EventID == "" {
			t.Fatal("expected event id to be populated")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TTS candidate")
	}
}

func TestRouteTTSCandidate_DeduplicatesByEventIdentity(t *testing.T) {
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

	ttsCh := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ttsCh)

	first := srv.routeTTSCandidate("duplicate me", sess.ID, "test")
	second := srv.routeTTSCandidate("duplicate me", sess.ID, "test")
	if !first.Routed {
		t.Fatalf("expected first routing to succeed, got %+v", first)
	}
	if !second.Routed || !second.Duplicate || second.Code != "tts_duplicate" {
		t.Fatalf("expected duplicate result, got %+v", second)
	}

	select {
	case <-ttsCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first TTS candidate")
	}

	select {
	case <-ttsCh:
		t.Fatal("did not expect duplicate candidate to be re-routed")
	case <-time.After(100 * time.Millisecond):
	}
}
