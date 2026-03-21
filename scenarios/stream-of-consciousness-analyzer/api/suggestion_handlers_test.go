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

// [REQ:P1-001] [REQ:P1-001b] Test generate suggestions returns suggestion data
func TestHandleGenerateSuggestions_WithResults(t *testing.T) {
	svc := newMockSuggestions().WithSuggestions([]Suggestion{
		{ID: "sug-1", SourceID: "t1", TargetID: "t2", Label: "related", Confidence: 0.85},
		{ID: "sug-2", SourceID: "t1", TargetID: "t3", Label: "causes", Confidence: 0.72},
	})
	handler := handleGenerateSuggestions(svc)

	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/suggestions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	result := decodeJSON[map[string]interface{}](t, w)
	suggestions, ok := result["suggestions"].([]interface{})
	if !ok {
		t.Fatal("expected suggestions array in response")
	}
	if len(suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(suggestions))
	}
}

// [REQ:P1-001] [REQ:P2-003a] Test get providers returns both primary and fallback
func TestHandleGetProviders_Structure(t *testing.T) {
	svc := newMockSuggestions()
	handler := handleGetProviders(svc)

	req := httptest.NewRequest("GET", "/api/v1/providers", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	providers := decodeJSON[[]LLMProvider](t, w)

	hasPrimary := false
	hasFallback := false
	for _, p := range providers {
		if !p.Fallback {
			hasPrimary = true
		}
		if p.Fallback {
			hasFallback = true
		}
	}
	if !hasPrimary {
		t.Error("expected at least one primary provider in response")
	}
	if !hasFallback {
		t.Error("expected at least one fallback provider in response")
	}
}

// [REQ:P2-001] Test error response from suggestion handler is structured
func TestHandleGenerateSuggestions_ErrorStructure(t *testing.T) {
	svc := newMockSuggestions().WithGenerateError(fmt.Errorf("connection refused"))
	handler := handleGenerateSuggestions(svc)

	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/suggestions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusServiceUnavailable)
	apiErr := decodeJSON[APIError](t, w)
	if apiErr.Category != ErrCategoryDependency {
		t.Errorf("expected category=dependency, got %s", apiErr.Category)
	}
	if !apiErr.Retryable {
		t.Error("expected retryable=true for dependency errors")
	}
}

// [REQ:P1-001] Test get providers response content type
func TestHandleGetProviders_ContentType(t *testing.T) {
	svc := newMockSuggestions()
	handler := handleGetProviders(svc)

	req := httptest.NewRequest("GET", "/api/v1/providers", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}
