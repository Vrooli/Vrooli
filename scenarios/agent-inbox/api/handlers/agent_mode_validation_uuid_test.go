package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func TestStartAgentMode_InvalidJSON(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString("{invalid json")
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartAgentMode_MissingMessage(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString(`{"project_path": "/tmp"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartAgentMode_MissingProjectPath(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString(`{"message": "hello"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartAgentMode_NonexistentProjectPath(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString(`{"message": "hello", "project_path": "/nonexistent/path/xyz"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("does not exist")) {
		t.Errorf("expected 'does not exist' in response, got: %s", w.Body.String())
	}
}

func TestStartAgentMode_FileNotDirectory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := &Handlers{}
	body := bytes.NewBufferString(`{"message": "hello", "project_path": "` + tmpFile + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/start", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.StartAgentMode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("not a directory")) {
		t.Errorf("expected 'not a directory' in response, got: %s", w.Body.String())
	}
}

func TestSendAgentMessage_EmptyMessage(t *testing.T) {
	h := &Handlers{}
	body := bytes.NewBufferString(`{"message": ""}`)
	req := httptest.NewRequest("POST", "/api/v1/chats/550e8400-e29b-41d4-a716-446655440000/agent-mode/message", body)
	req = mux.SetURLVars(req, map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	w := httptest.NewRecorder()

	h.SendAgentMessage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
