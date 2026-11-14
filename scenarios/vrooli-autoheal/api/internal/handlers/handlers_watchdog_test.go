package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type watchdogControlPlaneStub struct {
	output []byte
	err    error
	args   []string
}

func (s *watchdogControlPlaneStub) OutputCombined(_ context.Context, args ...string) ([]byte, error) {
	s.args = append([]string(nil), args...)
	return s.output, s.err
}

func TestWatchdogTemplate(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/watchdog/template", nil)
	w := httptest.NewRecorder()

	h.WatchdogTemplate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("WatchdogTemplate() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check required fields
	if _, ok := resp["platform"]; !ok {
		t.Error("response should have platform field")
	}

	if _, ok := resp["template"]; !ok {
		t.Error("response should have template field")
	}

	if _, ok := resp["instructions"]; !ok {
		t.Error("response should have instructions field")
	}

	// Template should have substantial content
	template, ok := resp["template"].(string)
	if !ok || len(template) < 50 {
		t.Errorf("template should be a substantial string, got length %d", len(template))
	}
}

func TestWatchdogInstallDelegatesToTypedControlPlaneClient(t *testing.T) {
	h := setupTestHandlers(&mockStore{})
	client := &watchdogControlPlaneStub{output: []byte("applied")}
	h.controlPlane = client

	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchdog/install", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.WatchdogInstall(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("WatchdogInstall() status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	wantArgs := []string{"host", "safeguard", "autoheal_watchdog"}
	if !reflect.DeepEqual(client.args, wantArgs) {
		t.Fatalf("control-plane args = %v, want %v", client.args, wantArgs)
	}
	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if success, _ := result["success"].(bool); !success {
		t.Fatalf("response = %v, want success", result)
	}
}

func TestWatchdogInstallReportsControlPlaneFailure(t *testing.T) {
	h := setupTestHandlers(&mockStore{})
	h.controlPlane = &watchdogControlPlaneStub{
		output: []byte("safeguard refused"),
		err:    errors.New("exit status 1"),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchdog/install", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.WatchdogInstall(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("WatchdogInstall() status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errText, _ := result["error"].(string)
	if !strings.Contains(errText, "safeguard refused") || !strings.Contains(errText, "exit status 1") {
		t.Fatalf("error = %q, want command error and output", errText)
	}
}

// --- Additional Handler Tests ---
