package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mustNewApp(t *testing.T) *App {
	t.Helper()
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app
}

func TestNewApp(t *testing.T) {
	app := mustNewApp(t)
	if app == nil || app.core == nil {
		t.Fatal("expected initialized app core")
	}
}

func TestAPIPath(t *testing.T) {
	app := mustNewApp(t)

	cases := []struct {
		input    string
		expected string
	}{
		{"/events", "/api/v1/events"},
		{"events", "/api/v1/events"},
		{"/events/subscribe", "/api/v1/events/subscribe"},
		{"", ""},
	}

	for _, tc := range cases {
		if got := app.core.APIPath(tc.input); got != tc.expected {
			t.Errorf("APIPath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRunRegistersCommands(t *testing.T) {
	app := mustNewApp(t)

	commands := [][]string{
		{"ingest"},
		{"query"},
		{"subscribe"},
		{"stats"},
		{"status"},
		{"configure"},
	}

	for _, args := range commands {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			err := app.Run(args)
			if err != nil && strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("expected command %q to be registered", strings.Join(args, " "))
			}
		})
	}
}

func TestIngestAndQueryCommands(t *testing.T) {
	var receivedBody map[string]any
	var receivedQuery string
	mux := http.NewServeMux()
	// NeedsAPI commands preflight-probe /health via ensureAPIReachable; without this
	// the ingest/query handlers never run and the test sees a 404 recovery report.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "healthy"})
	})
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "eventId": "evt-1"})
		case http.MethodGet:
			receivedQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{"eventId": "evt-1", "eventType": "test.v1", "sourceScenario": "src"},
			})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app := mustNewApp(t)

	if err := app.Run([]string{
		"ingest",
		"--event-id", "evt-1",
		"--type", "app.user.created.v1",
		"--source", "test-scenario",
		"--target", "other-scenario",
		"--correlation-id", "corr-123",
	}); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	if receivedBody["eventId"] != "evt-1" {
		t.Fatalf("expected eventId evt-1, got %v", receivedBody["eventId"])
	}

	if err := app.Run([]string{"query", "--type", "test.*", "--source", "mysrc", "--limit", "10"}); err != nil {
		t.Fatalf("query: %v", err)
	}
	if receivedQuery == "" || !strings.Contains(receivedQuery, "limit=10") {
		t.Fatalf("expected query params, got %q", receivedQuery)
	}
}

func TestStatsUsesRootHealth(t *testing.T) {
	var receivedPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "healthy",
			"subscribers": 3,
			"store": map[string]any{
				"totalEvents":       100,
				"totalPayloadBytes": 2048,
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app := mustNewApp(t)

	if err := app.Run([]string{"stats"}); err != nil {
		t.Fatalf("stats: %v", err)
	}
	if receivedPath != "/health" {
		t.Fatalf("expected /health, got %s", receivedPath)
	}
}

func TestValidationErrors(t *testing.T) {
	app := mustNewApp(t)

	tests := []struct {
		name string
		args []string
	}{
		{"ingest", []string{"ingest"}},
		{"ingest missing type", []string{"ingest", "--event-id", "e1", "--source", "s"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := app.Run(tc.args); err == nil {
				t.Fatalf("expected validation error for %q", strings.Join(tc.args, " "))
			}
		})
	}
}
