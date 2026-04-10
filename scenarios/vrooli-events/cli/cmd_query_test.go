package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// [REQ:REQ-CLI-002] Query command calls GET /api/v1/events with filters
func TestCmdQuery_CallsAPI(t *testing.T) {
	var receivedQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"eventId": "evt-1", "eventType": "test.v1", "sourceScenario": "src"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdQuery([]string{"--type", "test.*", "--source", "mysrc", "--limit", "10"}); err != nil {
		t.Fatalf("cmdQuery: %v", err)
	}

	if receivedQuery == "" {
		t.Fatal("expected query params to be sent")
	}
}

// [REQ:REQ-CLI-002] Query command handles empty results
func TestCmdQuery_EmptyResults(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdQuery(nil); err != nil {
		t.Fatalf("cmdQuery with empty results: %v", err)
	}
}

// [REQ:REQ-CLI-002] Query command passes all filter flags to API
func TestCmdQuery_AllFilters(t *testing.T) {
	var receivedPath string
	var receivedQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdQuery([]string{
		"--type", "app.user.**",
		"--source", "my-scenario",
		"--correlation-id", "trace-abc",
		"--since", "100",
		"--limit", "50",
	}); err != nil {
		t.Fatalf("cmdQuery: %v", err)
	}

	if receivedPath != "/api/v1/events" {
		t.Fatalf("expected path /api/v1/events, got %s", receivedPath)
	}
	if !strings.Contains(receivedQuery, "type=app.user") {
		t.Fatalf("expected type param, got query: %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "source=my-scenario") {
		t.Fatalf("expected source param, got query: %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "correlation_id=trace-abc") {
		t.Fatalf("expected correlation_id param, got query: %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "since=100") {
		t.Fatalf("expected since param, got query: %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "limit=50") {
		t.Fatalf("expected limit param, got query: %s", receivedQuery)
	}
}

// [REQ:REQ-CLI-002] Query command supports --json output
func TestCmdQuery_JSONOutput(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"eventId": "evt-1", "eventType": "test.v1"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdQuery([]string{"--json"}); err != nil {
		t.Fatalf("cmdQuery --json: %v", err)
	}
}
