package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func newHookTestServer(token string) *Server {
	return &Server{
		router:        mux.NewRouter(),
		sessions:      NewSessionManagerWithFactory(newFakePTYFactory()),
		events:        NewEventLogger(100),
		metrics:       NewMetrics(),
		workspace:     NewMemWorkspaceStore(),
		hookAuthToken: token,
		ttsConfig:     TTSConfig{AutoEnabled: true, Backend: "auto", KokoroVoice: "af_heart", KokoroSpeed: 1.0},
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
	if routing["code"] != "tts_target_missing" {
		t.Errorf("expected tts_target_missing, got %v", routing["code"])
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

	fake := newFakePTYWithOutput()
	defer fake.Close()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	srv.sessions = sm
	srv.ttsDedup = newTTSDedup()

	sess, err := sm.Create("", 80, 24)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = sm.Delete(sess.ID) }()

	ttsCh := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ttsCh)

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
	case candidate := <-ttsCh:
		if candidate.Text != "hello from claude" {
			t.Fatalf("expected %q, got %q", "hello from claude", candidate.Text)
		}
		if candidate.SessionID != sess.ID {
			t.Fatalf("expected session %s, got %s", sess.ID, candidate.SessionID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TTS routing")
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	routing, ok := resp["routing"].(map[string]any)
	if !ok {
		t.Fatalf("expected routing object, got %T", resp["routing"])
	}
	if routing["code"] != "tts_candidate_routed" {
		t.Fatalf("expected tts_candidate_routed, got %v", routing["code"])
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
