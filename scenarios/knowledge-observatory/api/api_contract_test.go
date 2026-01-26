package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAPIVersioningContract ensures versioned endpoints remain stable. [REQ:KO-API-002]
func TestAPIVersioningContract(t *testing.T) {
	srv := newTestServerWithServices()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected v1 search to respond, got %d", w.Code)
	}

	missing := httptest.NewRequest(http.MethodPost, "/api/v2/knowledge/search", strings.NewReader("{}"))
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, missing)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected v2 search to 404, got %d", w.Code)
	}
}
