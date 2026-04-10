package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:REQ-CLI-002] Ingest command sends POST /api/v1/events
func TestCmdIngest_Success(t *testing.T) {
	var receivedBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"id": 1, "eventId": "test-1"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("VROOLI_EVENTS_API_BASE", srv.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if err := app.cmdIngest([]string{
		"--event-id", "test-1",
		"--type", "app.user.created.v1",
		"--source", "test-scenario",
		"--target", "other-scenario",
		"--correlation-id", "corr-123",
	}); err != nil {
		t.Fatalf("cmdIngest: %v", err)
	}

	if receivedBody["eventId"] != "test-1" {
		t.Fatalf("expected eventId=test-1, got %v", receivedBody["eventId"])
	}
	if receivedBody["sourceScenario"] != "test-scenario" {
		t.Fatalf("expected source=test-scenario, got %v", receivedBody["sourceScenario"])
	}
}

// [REQ:REQ-CLI-002] Ingest command rejects missing required fields
func TestCmdIngest_MissingRequired(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	// Missing all required flags
	if err := app.cmdIngest(nil); err == nil {
		t.Fatal("expected error for missing required flags")
	}

	// Missing --type
	if err := app.cmdIngest([]string{"--event-id", "e1", "--source", "s"}); err == nil {
		t.Fatal("expected error for missing --type")
	}
}
