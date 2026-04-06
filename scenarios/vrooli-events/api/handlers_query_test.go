package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/testutil"
)

// [REQ:REQ-API-002] Event query returns stored events
// [REQ:REQ-ES-002] Event schema matches expected fields
func TestQueryReturnsStoredEvents(t *testing.T) {
	_, ts := newTestServer(t)

	// Ingest an event first
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
	resp.Body.Close()

	// Query events
	resp, err = http.Get(ts.URL + "/api/v1/events?source=test-source")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	events := decodeJSON[[]map[string]any](t, resp)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0]["eventType"] != "test.domain.action.v1" {
		t.Fatalf("expected event type, got %v", events[0]["eventType"])
	}
}

// [REQ:REQ-API-002] Query supports limit and source filters
func TestQueryWithFilters(t *testing.T) {
	_, ts := newTestServer(t)

	for i := 1; i <= 3; i++ {
		body := fmt.Sprintf(`{"eventId":"evt-%d","sourceScenario":"src-%d","eventType":"app.domain.action.v1","correlationId":"corr-1"}`, i, i%2+1)
		_, _ = http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
	}

	// Query with limit
	resp, _ := http.Get(ts.URL + "/api/v1/events?limit=2")
	events := decodeJSON[[]map[string]any](t, resp)
	resp.Body.Close()
	if len(events) != 2 {
		t.Fatalf("limit: expected 2, got %d", len(events))
	}

	// Query with source filter
	resp, _ = http.Get(ts.URL + "/api/v1/events?source=src-1")
	events = decodeJSON[[]map[string]any](t, resp)
	resp.Body.Close()
	if len(events) != 1 {
		t.Fatalf("source filter: expected 1, got %d", len(events))
	}
}

// [REQ:REQ-API-002] Empty query returns empty array
func TestEmptyQueryResult(t *testing.T) {
	_, ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/events")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	body := decodeJSON[json.RawMessage](t, resp)
	if string(body) != "[]" {
		t.Fatalf("expected [], got %s", body)
	}
}

// [REQ:REQ-API-002] Query supports correlation_id filter
func TestQuery_CorrelationFilter(t *testing.T) {
	_, ts := newTestServer(t)

	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"c1","sourceScenario":"s","eventType":"t.v1","correlationId":"trace-aaa"}`))
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"c2","sourceScenario":"s","eventType":"t.v1","correlationId":"trace-bbb"}`))
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"c3","sourceScenario":"s","eventType":"t.v1","correlationId":"trace-aaa"}`))

	resp, err := http.Get(ts.URL + "/api/v1/events?correlation_id=trace-aaa")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	events := decodeJSON[[]map[string]any](t, resp)
	if len(events) != 2 {
		t.Fatalf("expected 2 events with correlation trace-aaa, got %d", len(events))
	}
}

// [REQ:REQ-API-002] Query returns store error as 500
func TestQuery_StoreError(t *testing.T) {
	ms := (&testutil.MockStore{}).WithQueryResult(nil, fmt.Errorf("corrupt index"))
	mb := testutil.NewMockBroker()
	ts := newMockedServer(t, ms, mb)

	resp, err := http.Get(ts.URL + "/api/v1/events")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeStoreRead {
		t.Fatalf("expected %s, got %s", ErrCodeStoreRead, errBody["code"])
	}
}

// [REQ:REQ-API-002] Query rejects invalid since parameter
func TestQuery_InvalidSince(t *testing.T) {
	ms := &testutil.MockStore{}
	mb := testutil.NewMockBroker()
	ts := newMockedServer(t, ms, mb)

	resp, err := http.Get(ts.URL + "/api/v1/events?since=notanumber")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeInvalidParam {
		t.Fatalf("expected %s, got %s", ErrCodeInvalidParam, errBody["code"])
	}
}

// [REQ:REQ-API-002] Query rejects invalid limit parameter
func TestQuery_InvalidLimit(t *testing.T) {
	ms := &testutil.MockStore{}
	mb := testutil.NewMockBroker()
	ts := newMockedServer(t, ms, mb)

	resp, err := http.Get(ts.URL + "/api/v1/events?limit=abc")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeInvalidParam {
		t.Fatalf("expected %s, got %s", ErrCodeInvalidParam, errBody["code"])
	}
}

// [REQ:REQ-API-002] Query with type filter returns matching events
func TestQuery_TypeFilter(t *testing.T) {
	_, ts := newTestServer(t)

	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"t1","sourceScenario":"s","eventType":"order.created.v1"}`))
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"t2","sourceScenario":"s","eventType":"user.signup.v1"}`))
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"t3","sourceScenario":"s","eventType":"order.shipped.v1"}`))

	resp, err := http.Get(ts.URL + "/api/v1/events?type=order.created.v1")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	events := decodeJSON[[]map[string]any](t, resp)
	if len(events) != 1 {
		t.Fatalf("expected 1 event for type filter, got %d", len(events))
	}
	if events[0]["eventType"] != "order.created.v1" {
		t.Fatalf("expected order.created.v1, got %v", events[0]["eventType"])
	}
}

// [REQ:REQ-API-002] Query with combined filters narrows results
func TestQuery_CombinedFilters(t *testing.T) {
	_, ts := newTestServer(t)

	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"cf1","sourceScenario":"alpha","eventType":"deploy.v1","correlationId":"trace-x"}`))
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"cf2","sourceScenario":"beta","eventType":"deploy.v1","correlationId":"trace-x"}`))
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"cf3","sourceScenario":"alpha","eventType":"build.v1","correlationId":"trace-y"}`))

	// Filter by source + type
	resp, _ := http.Get(ts.URL + "/api/v1/events?source=alpha&type=deploy.v1")
	events := decodeJSON[[]map[string]any](t, resp)
	resp.Body.Close()
	if len(events) != 1 {
		t.Fatalf("expected 1 event for source=alpha&type=deploy.v1, got %d", len(events))
	}

	// Filter by correlation + limit
	resp, _ = http.Get(ts.URL + "/api/v1/events?correlation_id=trace-x&limit=1")
	events = decodeJSON[[]map[string]any](t, resp)
	resp.Body.Close()
	if len(events) != 1 {
		t.Fatalf("expected 1 event with limit=1, got %d", len(events))
	}
}

// [REQ:REQ-API-002] Query with since filter returns events after given ID
func TestQuery_SinceFilter(t *testing.T) {
	_, ts := newTestServer(t)

	// Insert 3 events
	var firstID float64
	for i := 1; i <= 3; i++ {
		body := fmt.Sprintf(`{"eventId":"since-%d","sourceScenario":"s","eventType":"t.v1"}`, i)
		resp, _ := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
		if i == 1 {
			result := decodeJSON[map[string]any](t, resp)
			firstID = result["id"].(float64)
		}
		resp.Body.Close()
	}

	// Query since the first event
	resp, _ := http.Get(fmt.Sprintf("%s/api/v1/events?since=%d", ts.URL, int64(firstID)))
	events := decodeJSON[[]map[string]any](t, resp)
	resp.Body.Close()

	// Should return events after the first one (2 events)
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events after since=%d, got %d", int64(firstID), len(events))
	}
}
