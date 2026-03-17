package main

import (
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func newTTSTestServer() *Server {
	return &Server{
		router:    mux.NewRouter(),
		sessions:  NewSessionManagerWithFactory(newFakePTYFactory()),
		events:    NewEventLogger(100),
		metrics:   NewMetrics(),
		workspace: NewMemWorkspaceStore(),
		ttsDedup:  newTTSDedup(),
	}
}

func TestDeliverTTS_Disabled(t *testing.T) {
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: false}

	result := srv.deliverTTS("some text", "", "test")
	if result.Delivered {
		t.Error("expected false when auto-TTS is disabled")
	}
	if result.Code != "tts_auto_disabled" {
		t.Errorf("expected tts_auto_disabled, got %s", result.Code)
	}
}

func TestDeliverTTS_NoActiveSession(t *testing.T) {
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: true}
	// workspace has no active pane by default

	result := srv.deliverTTS("some text", "", "test")
	if result.Delivered {
		t.Error("expected false when no active session")
	}
	if result.Code != "tts_delivery_target_missing" {
		t.Errorf("expected tts_delivery_target_missing, got %s", result.Code)
	}
}

func TestDeliverTTS_NoTextMatch(t *testing.T) {
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

	// Set active pane (UpsertPane first, then SavePaneOrder to set IsActive)
	_ = srv.workspace.UpsertPane(&WorkspacePane{SessionID: sess.ID, Name: "test"})
	_ = srv.workspace.SavePaneOrder(sess.ID, []string{sess.ID})

	// Session buffer has different text
	sess.mu.Lock()
	sess.outputHistory = []byte("completely different content")
	sess.mu.Unlock()

	result := srv.deliverTTS("not in buffer at all", "", "test")
	if result.Delivered {
		t.Error("expected false when text not in buffer")
	}
	if result.Code != "tts_correlation_failed" {
		t.Errorf("expected tts_correlation_failed, got %s", result.Code)
	}
}

func TestDeliverTTS_MatchDelivers(t *testing.T) {
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

	// Set active pane (UpsertPane first, then SavePaneOrder to set IsActive)
	_ = srv.workspace.UpsertPane(&WorkspacePane{SessionID: sess.ID, Name: "test"})
	_ = srv.workspace.SavePaneOrder(sess.ID, []string{sess.ID})

	// Put matching text in session buffer
	responseText := "The answer is 42"
	sess.mu.Lock()
	sess.outputHistory = []byte("some prefix " + responseText + " some suffix")
	sess.mu.Unlock()

	// Subscribe to TTS before delivery
	ttsCh := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ttsCh)

	if !srv.deliverTTS(responseText, "", "test").Delivered {
		t.Error("expected true when text matches and TTS is enabled")
	}

	// Verify message was sent
	select {
	case msg := <-ttsCh:
		if msg != responseText {
			t.Errorf("expected %q, got %q", responseText, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TTS message")
	}
}

func TestDeliverTTS_StripsANSI(t *testing.T) {
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

	// Put ANSI-laden text in session buffer (simulates colored terminal output)
	ansiText := "\x1b[31mHello\x1b[0m world"
	sess.mu.Lock()
	sess.outputHistory = []byte(ansiText)
	sess.mu.Unlock()

	ttsCh := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ttsCh)

	if !srv.deliverTTS(ansiText, "", "test").Delivered {
		t.Error("expected true when ANSI text matches")
	}

	// Verify the subscriber receives CLEAN text (ANSI stripped)
	select {
	case msg := <-ttsCh:
		if msg != "Hello world" {
			t.Errorf("expected clean text %q, got %q", "Hello world", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TTS message")
	}
}

// setupDeliveryServer creates a test server with a session whose output
// history contains responseText, a TTS subscriber, and auto-TTS enabled.
func setupDeliveryServer(t *testing.T, responseText string) (*Server, *Session, chan string) {
	t.Helper()
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: true}

	fake := newFakePTYWithOutput()
	t.Cleanup(func() { fake.Close() })
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	srv.sessions = sm

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(sess.ID) })

	_ = srv.workspace.UpsertPane(&WorkspacePane{SessionID: sess.ID, Name: "test"})
	_ = srv.workspace.SavePaneOrder(sess.ID, []string{sess.ID})

	sess.mu.Lock()
	sess.outputHistory = []byte("prefix " + responseText + " suffix")
	sess.mu.Unlock()

	ttsCh := sess.SubscribeTTS()
	t.Cleanup(func() { sess.UnsubscribeTTS(ttsCh) })

	return srv, sess, ttsCh
}

func TestDeliverTTS_DeduplicatesSameText(t *testing.T) {
	srv, _, ttsCh := setupDeliveryServer(t, "duplicate me")

	// First call delivers.
	if !srv.deliverTTS("duplicate me", "", "test").Delivered {
		t.Fatal("first delivery should succeed")
	}
	// Second call with identical text is deduplicated.
	result := srv.deliverTTS("duplicate me", "", "test")
	if !result.Delivered {
		t.Fatal("dedup call should still return true")
	}
	if !result.Duplicate {
		t.Fatal("expected duplicate flag on second delivery")
	}

	// Only one message should be on the channel.
	select {
	case <-ttsCh:
	case <-time.After(time.Second):
		t.Fatal("expected one TTS message")
	}
	select {
	case msg := <-ttsCh:
		t.Fatalf("expected no second message, got %q", msg)
	case <-time.After(50 * time.Millisecond):
		// Good — no duplicate.
	}
}

func TestDeliverTTS_AllowsAfterTTLExpires(t *testing.T) {
	srv, _, ttsCh := setupDeliveryServer(t, "expire me")

	// Use a very short TTL for the dedup cache.
	srv.ttsDedup.ttl = 10 * time.Millisecond

	if !srv.deliverTTS("expire me", "", "test").Delivered {
		t.Fatal("first delivery should succeed")
	}
	<-ttsCh // drain

	// Wait for TTL to expire.
	time.Sleep(20 * time.Millisecond)

	if !srv.deliverTTS("expire me", "", "test").Delivered {
		t.Fatal("delivery after TTL expiry should succeed")
	}
	select {
	case <-ttsCh:
	case <-time.After(time.Second):
		t.Fatal("expected second TTS message after TTL expiry")
	}
}

func TestDeliverTTS_DifferentTextNotDeduplicated(t *testing.T) {
	// Put both texts in history so both match.
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

	sess.mu.Lock()
	sess.outputHistory = []byte("first text and also second text")
	sess.mu.Unlock()

	ttsCh := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ttsCh)

	if !srv.deliverTTS("first text", "", "test").Delivered {
		t.Fatal("first delivery should succeed")
	}
	if !srv.deliverTTS("second text", "", "test").Delivered {
		t.Fatal("different text should not be deduplicated")
	}

	// Both messages should arrive.
	for i := 0; i < 2; i++ {
		select {
		case <-ttsCh:
		case <-time.After(time.Second):
			t.Fatalf("expected message %d", i+1)
		}
	}
}

func TestDeliverTTS_UsesTargetSession(t *testing.T) {
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: true}

	fake := newFakePTYWithOutput()
	defer fake.Close()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	srv.sessions = sm

	activeSess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("create active session: %v", err)
	}
	targetSess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("create target session: %v", err)
	}
	defer func() {
		_ = sm.Delete(activeSess.ID)
		_ = sm.Delete(targetSess.ID)
	}()

	_ = srv.workspace.UpsertPane(&WorkspacePane{SessionID: activeSess.ID, Name: "active"})
	_ = srv.workspace.UpsertPane(&WorkspacePane{SessionID: targetSess.ID, Name: "target"})
	_ = srv.workspace.SavePaneOrder(activeSess.ID, []string{activeSess.ID, targetSess.ID})

	targetText := "deliver to target pane"
	targetSess.mu.Lock()
	targetSess.outputHistory = []byte("prefix " + targetText + " suffix")
	targetSess.mu.Unlock()

	activeCh := activeSess.SubscribeTTS()
	targetCh := targetSess.SubscribeTTS()
	defer activeSess.UnsubscribeTTS(activeCh)
	defer targetSess.UnsubscribeTTS(targetCh)

	result := srv.deliverTTS(targetText, targetSess.ID, "test")
	if !result.Delivered {
		t.Fatalf("expected delivery to target session, got %+v", result)
	}
	if !result.UsedTargetSession {
		t.Fatal("expected usedTargetSession=true")
	}

	select {
	case msg := <-targetCh:
		if msg != targetText {
			t.Fatalf("expected target message %q, got %q", targetText, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for target TTS message")
	}
	select {
	case msg := <-activeCh:
		t.Fatalf("expected active pane to receive nothing, got %q", msg)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDeliverTTS_TargetSessionMissingDoesNotFallback(t *testing.T) {
	srv := newTTSTestServer()
	srv.ttsConfig = TTSConfig{AutoEnabled: true}

	result := srv.deliverTTS("hello", "missing-session", "test")
	if result.Delivered {
		t.Fatal("expected missing target session to fail")
	}
	if result.Code != "tts_delivery_target_missing" {
		t.Fatalf("expected tts_delivery_target_missing, got %s", result.Code)
	}
	if !result.UsedTargetSession {
		t.Fatal("expected usedTargetSession=true for explicit missing target")
	}
}
