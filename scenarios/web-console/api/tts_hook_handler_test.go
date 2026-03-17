package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	body := strings.NewReader(`{"assistantResponse":"hello","sessionId":"s1"}`)
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
	delivery, ok := resp["delivery"].(map[string]any)
	if !ok {
		t.Fatalf("expected delivery object, got %T", resp["delivery"])
	}
	if delivery["code"] != "tts_delivery_target_missing" {
		t.Errorf("expected tts_delivery_target_missing, got %v", delivery["code"])
	}
	if resp["delivered"] != false {
		t.Errorf("expected delivered=false (no session), got %v", resp["delivered"])
	}
}

func TestHandleHookStop_AnthropicPayloadShape(t *testing.T) {
	srv := newHookTestServer("secret-token")
	body := strings.NewReader(`{"hook_event_name":"Stop","last_assistant_message":"hello from claude","session_id":"s1"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/stop", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "secret-token")
	rec := httptest.NewRecorder()

	srv.handleHookStop(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
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
