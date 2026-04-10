package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// [REQ:REQ-P0-001] Helper function tests

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"key": "value"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	if !strings.Contains(w.Body.String(), `"key":"value"`) {
		t.Fatalf("body = %q, expected to contain key:value", w.Body.String())
	}
}

func TestWriteResourceLoadError(t *testing.T) {
	w := httptest.NewRecorder()
	writeResourceLoadError(w, errors.New("disk full"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	body := w.Body.String()
	if !strings.Contains(body, "failed to load resources") {
		t.Fatalf("body missing prefix: %s", body)
	}
	if !strings.Contains(body, "disk full") {
		t.Fatalf("body missing error: %s", body)
	}
}

func TestDecodeJSONBodySuccess(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"test"}`))

	var dst struct{ Name string }
	ok := decodeJSONBody(w, r, &dst)
	if !ok {
		t.Fatal("decodeJSONBody returned false, want true")
	}
	if dst.Name != "test" {
		t.Fatalf("Name = %q, want %q", dst.Name, "test")
	}
}

func TestDecodeJSONBodyMalformed(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{not valid json`))

	var dst struct{ Name string }
	ok := decodeJSONBody(w, r, &dst)
	if ok {
		t.Fatal("decodeJSONBody returned true for invalid JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "invalid JSON body") {
		t.Fatalf("body = %q, expected invalid JSON error", w.Body.String())
	}
}

func TestDecodeJSONBodyEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

	var dst struct{ Name string }
	ok := decodeJSONBody(w, r, &dst)
	if ok {
		t.Fatal("decodeJSONBody returned true for empty body")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestRecoveryMiddleware verifies the handler wraps panics.
func TestRecoveryMiddleware(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VROOLI_ROOT", dir)
	writeResourcesFile(t, dir, []map[string]string{testResPostgres})

	srv := NewServer(nil)
	handler := srv.Handler()

	// Inject a panicking route for testing
	srv.router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}).Methods("GET")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic recovery: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
