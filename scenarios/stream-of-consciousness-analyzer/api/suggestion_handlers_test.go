package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// --- Suggestion Handler Behavioral Tests ---

// [REQ:P1-001] Test get providers handler
func TestHandleGetProviders(t *testing.T) {
	svc := newMockSuggestions()
	handler := handleGetProviders(svc)

	req := httptest.NewRequest("GET", "/api/v1/providers", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	providers := decodeJSON[[]LLMProvider](t, w)
	if len(providers) < 1 {
		t.Error("expected at least one provider")
	}
}

// [REQ:P1-001] Test generate suggestions handler success
func TestHandleGenerateSuggestions_Success(t *testing.T) {
	svc := newMockSuggestions()
	handler := handleGenerateSuggestions(svc)

	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/suggestions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	result := decodeJSON[map[string]interface{}](t, w)
	if result["provider"] != "ollama" {
		t.Errorf("expected provider=ollama, got %v", result["provider"])
	}
}

// [REQ:P2-001] Test generate suggestions handler returns 503 on provider error
func TestHandleGenerateSuggestions_ServiceUnavailable(t *testing.T) {
	svc := newMockSuggestions().WithGenerateError(fmt.Errorf("no LLM provider available"))
	handler := handleGenerateSuggestions(svc)

	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/suggestions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusServiceUnavailable)
}

// [REQ:P1-003] Test generate suggestions handler factory
func TestHandleGenerateSuggestions_Init(t *testing.T) {
	handler := handleGenerateSuggestions(newMockSuggestions())
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}
