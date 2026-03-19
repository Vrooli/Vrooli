package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleHookPromptSubmit_MissingToken(t *testing.T) {
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret"

	body := strings.NewReader(`{"userPrompt":"hello","webConsoleSessionId":"sess1"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/prompt-submit", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleHookPromptSubmit(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", rec.Code)
	}
}

func TestHandleHookPromptSubmit_WrongToken(t *testing.T) {
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret"

	body := strings.NewReader(`{"userPrompt":"hello","webConsoleSessionId":"sess1"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/prompt-submit", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "wrong")
	rec := httptest.NewRecorder()

	srv.handleHookPromptSubmit(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong token, got %d", rec.Code)
	}
}

func TestHandleHookPromptSubmit_EmptyPrompt(t *testing.T) {
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret"

	body := strings.NewReader(`{"userPrompt":"","webConsoleSessionId":"sess1"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/prompt-submit", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "secret")
	rec := httptest.NewRecorder()

	srv.handleHookPromptSubmit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty prompt, got %d", rec.Code)
	}
}

func TestHandleHookPromptSubmit_EmptySessionID(t *testing.T) {
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret"

	body := strings.NewReader(`{"userPrompt":"hello","webConsoleSessionId":""}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/prompt-submit", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "secret")
	rec := httptest.NewRecorder()

	srv.handleHookPromptSubmit(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty session ID, got %d", rec.Code)
	}
}

func TestHandleHookPromptSubmit_ValidRequest(t *testing.T) {
	srv := newFakeTestServer()
	srv.hookAuthToken = "secret"
	srv.conversations = NewConversationStore()

	body := strings.NewReader(`{"userPrompt":"hello world","webConsoleSessionId":"sess1"}`)
	req := httptest.NewRequest("POST", "/api/v1/hooks/prompt-submit", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hook-Token", "secret")
	rec := httptest.NewRecorder()

	srv.handleHookPromptSubmit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
