package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	inttts "web-console/internal/tts"
	intworkspace "web-console/internal/workspace"
)

func newHookTestServer(token string) *Server {
	sm := newSessionManagerWithFactory(ptyfake.NewFactory())
	fanouts := NewConversationFanoutRegistry().AttachToManager(sm)
	return &Server{
		router:        mux.NewRouter(),
		sessions:      sm,
		fanouts:       fanouts,
		events:        events.NewLogger(100),
		metrics:       metrics.New(),
		workspace:     intworkspace.NewMemStore(),
		hookAuthToken: token,
		ttsConfig:     inttts.Config{AutoEnabled: true, Backend: "auto", KokoroVoice: "af_heart", KokoroSpeed: 1.0},
	}
}

func TestHandleHookStop_MissingToken(t *testing.T) {
	srv := newHookTestServer("secret-token")
	body := strings.NewReader(`{"assistantResponse":"hello","sessionId":"s1"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/stop", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleHookStop(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if errResp.Code != "unauthorized" {
		t.Errorf("expected code 'unauthorized', got %q", errResp.Code)
	}
}

func TestHandleHookStop_WrongToken(t *testing.T) {
	srv := newHookTestServer("secret-token")
	body := strings.NewReader(`{"assistantResponse":"hello","sessionId":"s1"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/stop", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "wrong-token")
	rec := httptest.NewRecorder()

	srv.handleHookStop(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleHookStop_ValidToken_NoSession(t *testing.T) {
	srv := newHookTestServer("secret-token")
	body := strings.NewReader(`{"assistantResponse":"hello","web_console_session_id":"missing-session"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/stop", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "secret-token")
	rec := httptest.NewRecorder()

	srv.handleHookStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
	routing, ok := resp["routing"].(map[string]any)
	if !ok {
		t.Fatalf("expected routing object, got %T", resp["routing"])
	}
	if routing["code"] != "conversation_target_missing" {
		t.Errorf("expected conversation_target_missing, got %v", routing["code"])
	}
	if resp["routed"] != false {
		t.Errorf("expected routed=false (no session), got %v", resp["routed"])
	}
}

func TestHandleHookStop_AnthropicPayloadShape(t *testing.T) {
	srv := newHookTestServer("secret-token")
	body := strings.NewReader(`{"hook_event_name":"Stop","last_assistant_message":"hello from claude","session_id":"s1","web_console_session_id":"missing-session"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/stop", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "secret-token")
	rec := httptest.NewRecorder()

	srv.handleHookStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleHookStop_RoutesToMappedTerminalSession(t *testing.T) {
	srv := newHookTestServer("secret-token")

	fake := ptyfake.NewFakePTYWithOutput()
	defer fake.Close()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))
	srv.sessions = sm
	srv.fanouts = NewConversationFanoutRegistry().AttachToManager(sm)
	srv.conversations = NewConversationStore()

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	fanout := srv.fanouts.Get(sess.ID)
	eventCh := fanout.Subscribe()
	defer fanout.Unsubscribe(eventCh)

	body := strings.NewReader(`{"hook_event_name":"Stop","last_assistant_message":"hello from claude","session_id":"claude-session-123","web_console_session_id":"` + sess.ID + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/stop", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "secret-token")
	rec := httptest.NewRecorder()

	srv.handleHookStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	select {
	case event := <-eventCh:
		if event.Text != "hello from claude" {
			t.Fatalf("expected %q, got %q", "hello from claude", event.Text)
		}
		if event.SessionID != sess.ID {
			t.Fatalf("expected session %s, got %s", sess.ID, event.SessionID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for conversation event routing")
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	routing, ok := resp["routing"].(map[string]any)
	if !ok {
		t.Fatalf("expected routing object, got %T", resp["routing"])
	}
	if routing["code"] != "conversation_event_appended" {
		t.Fatalf("expected conversation_event_appended, got %v", routing["code"])
	}
}

func TestHandleHookStop_PopulatesAgentInfoForRecovery(t *testing.T) {
	srv := newHookTestServer("secret-token")
	fake := ptyfake.NewFakePTYWithOutput()
	defer fake.Close()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))
	srv.sessions = sm
	srv.conversations = NewConversationStore()
	srv.sessionStore = sessionstore.NewInMemory()
	sm.SetStore(srv.sessionStore)

	sess, err := sm.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	body := strings.NewReader(`{"hook_event_name":"Stop","last_assistant_message":"work in progress","session_id":"claude-uuid-from-hook","cwd":"/repo","web_console_session_id":"` + sess.ID + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/stop", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "secret-token")
	rec := httptest.NewRecorder()
	srv.handleHookStop(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	got, err := srv.sessionStore.Get(sess.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AgentType != sessionstore.AgentClaude {
		t.Errorf("agent_type: got %q want claude", got.AgentType)
	}
	if got.AgentSessionID != "claude-uuid-from-hook" {
		t.Errorf("agent_session_id: got %q", got.AgentSessionID)
	}
	if got.CWD != "/repo" {
		t.Errorf("cwd: got %q", got.CWD)
	}
}

func TestHandleHookStop_MissingAssistantText(t *testing.T) {
	srv := newHookTestServer("secret-token")
	body := strings.NewReader(`{"hook_event_name":"Stop","session_id":"s1"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/stop", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "secret-token")
	rec := httptest.NewRecorder()

	srv.handleHookStop(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
