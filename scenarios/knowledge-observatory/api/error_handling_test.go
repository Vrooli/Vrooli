package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestErrorHandlingResponses validates consistent error payloads. [REQ:KO-API-004]
func TestErrorHandlingResponses(t *testing.T) {
	srv := newTestServer()
	srv.setupRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected json content type, got %q", w.Header().Get("Content-Type"))
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatalf("expected error message")
	}
}
