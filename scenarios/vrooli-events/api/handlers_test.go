package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	eventStore, err := store.NewSQLiteStore(context.Background(), store.SQLiteConfig{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { eventStore.Close() })

	eventBroker := broker.NewBroker(eventStore)
	t.Cleanup(func() { eventBroker.Close() })

	srv := &Server{store: eventStore, broker: eventBroker}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)

	return srv, ts
}

func TestHealthEndpoint(t *testing.T) {
	_, ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "healthy" {
		t.Fatalf("expected healthy, got %v", body["status"])
	}
}

func TestIngestAndQuery(t *testing.T) {
	_, ts := newTestServer(t)

	// Ingest an event
	eventJSON := `{
		"eventId": "test-evt-1",
		"sourceScenario": "test-source",
		"targetScenario": "test-target",
		"eventType": "test.domain.action.v1",
		"correlationId": "corr-1",
		"metadata": {"env": "test"}
	}`
	resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(eventJSON))
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	var ingestResp map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&ingestResp)
	if ingestResp["eventId"] != "test-evt-1" {
		t.Fatalf("expected eventId=test-evt-1, got %v", ingestResp["eventId"])
	}

	// Query events
	resp, err = http.Get(ts.URL + "/api/v1/events?source=test-source")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var events []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&events)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0]["eventType"] != "test.domain.action.v1" {
		t.Fatalf("expected event type, got %v", events[0]["eventType"])
	}
}

func TestIngestValidation(t *testing.T) {
	_, ts := newTestServer(t)

	tests := []struct {
		name string
		body string
		code string
	}{
		{"missing eventId", `{"sourceScenario":"s","eventType":"t.v1"}`, "MISSING_EVENT_ID"},
		{"missing eventType", `{"eventId":"e","sourceScenario":"s"}`, "MISSING_EVENT_TYPE"},
		{"missing source", `{"eventId":"e","eventType":"t.v1"}`, "MISSING_SOURCE"},
		{"invalid json", `not json`, "INVALID_BODY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}

			var body map[string]string
			_ = json.NewDecoder(resp.Body).Decode(&body)
			if body["code"] != tt.code {
				t.Fatalf("expected code %s, got %s", tt.code, body["code"])
			}
		})
	}
}

func TestQueryWithFilters(t *testing.T) {
	_, ts := newTestServer(t)

	// Insert multiple events
	for i := 1; i <= 3; i++ {
		body := fmt.Sprintf(`{"eventId":"evt-%d","sourceScenario":"src-%d","eventType":"app.domain.action.v1","correlationId":"corr-1"}`, i, i%2+1)
		_, _ = http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
	}

	// Query with limit
	resp, _ := http.Get(ts.URL + "/api/v1/events?limit=2")
	var events []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&events)
	resp.Body.Close()
	if len(events) != 2 {
		t.Fatalf("limit: expected 2, got %d", len(events))
	}

	// Query with source filter
	resp, _ = http.Get(ts.URL + "/api/v1/events?source=src-1")
	_ = json.NewDecoder(resp.Body).Decode(&events)
	resp.Body.Close()
	if len(events) != 1 {
		t.Fatalf("source filter: expected 1, got %d", len(events))
	}
}

func TestSSESubscription(t *testing.T) {
	_, ts := newTestServer(t)

	// Start SSE subscriber in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	// Read the retry line
	scanner := bufio.NewScanner(resp.Body)
	if scanner.Scan() {
		line := scanner.Text()
		if line != "retry: 5000" {
			t.Fatalf("expected retry: 5000, got %q", line)
		}
	}

	// Ingest an event
	eventJSON := `{"eventId":"sse-evt-1","sourceScenario":"test","eventType":"test.sse.v1"}`
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(eventJSON))

	// Read SSE events (skip empty lines)
	var gotEvent bool
	deadline := time.After(3 * time.Second)
	for !gotEvent {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for SSE event")
		default:
		}
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			if eventType == "test.sse.v1" {
				gotEvent = true
			}
		}
	}

	if !gotEvent {
		t.Fatal("did not receive SSE event")
	}
}

func TestEmptyQueryResult(t *testing.T) {
	_, ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/events")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	var body json.RawMessage
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if string(body) != "[]" {
		t.Fatalf("expected [], got %s", body)
	}
}
