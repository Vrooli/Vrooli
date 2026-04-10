package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/testutil"
)

// [REQ:REQ-API-001] Event ingestion returns 202 Accepted with id and eventId
func TestIngestSuccess(t *testing.T) {
	_, ts := newTestServer(t)

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

	ingestResp := decodeJSON[map[string]any](t, resp)
	if ingestResp["eventId"] != "test-evt-1" {
		t.Fatalf("expected eventId=test-evt-1, got %v", ingestResp["eventId"])
	}
	if ingestResp["id"] == nil {
		t.Fatal("expected id in response")
	}
}

// [REQ:REQ-API-001] Validation rejects malformed events
func TestIngestValidation(t *testing.T) {
	_, ts := newTestServer(t)

	tests := []struct {
		name string
		body string
		code string
	}{
		{"missing eventId", `{"sourceScenario":"s","eventType":"t.v1"}`, ErrCodeMissingField},
		{"missing eventType", `{"eventId":"e","sourceScenario":"s"}`, ErrCodeMissingField},
		{"missing source", `{"eventId":"e","eventType":"t.v1"}`, ErrCodeMissingField},
		{"invalid json", `not json`, ErrCodeInvalidBody},
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

			body := decodeJSON[map[string]string](t, resp)
			if body["code"] != tt.code {
				t.Fatalf("expected code %s, got %s", tt.code, body["code"])
			}
		})
	}
}

// [REQ:REQ-API-001] Ingest returns store error as 500
func TestIngest_StoreError(t *testing.T) {
	ms := (&testutil.MockStore{}).WithInsertResult(0, fmt.Errorf("disk full"))
	mb := testutil.NewMockBroker()
	ts := newMockedServer(t, ms, mb)

	body := `{"eventId":"e1","sourceScenario":"src","eventType":"test.v1"}`
	resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeStoreWrite {
		t.Fatalf("expected %s, got %s", ErrCodeStoreWrite, errBody["code"])
	}
}

// [REQ:REQ-ES-002] Duplicate event_id returns 409 Conflict
func TestIngest_DuplicateEvent(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{"eventId":"dup-evt-1","sourceScenario":"src","eventType":"test.v1"}`

	// First insert succeeds
	resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("first insert: expected 202, got %d", resp.StatusCode)
	}

	// Second insert with same event_id returns 409
	resp, err = http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate insert: expected 409, got %d", resp.StatusCode)
	}

	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeDuplicate {
		t.Fatalf("expected code %s, got %s", ErrCodeDuplicate, errBody["code"])
	}
}

// [REQ:REQ-ES-002] Duplicate event via mock store returns 409
func TestIngest_DuplicateEvent_Mock(t *testing.T) {
	ms := (&testutil.MockStore{}).WithInsertResult(0, fmt.Errorf("%w: evt-1", store.ErrDuplicateEvent))
	mb := testutil.NewMockBroker()
	ts := newMockedServer(t, ms, mb)

	body := `{"eventId":"evt-1","sourceScenario":"src","eventType":"test.v1"}`
	resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeDuplicate {
		t.Fatalf("expected %s, got %s", ErrCodeDuplicate, errBody["code"])
	}
}

// [REQ:REQ-API-001] Ingest rejects oversized request body
func TestIngest_OversizedBody(t *testing.T) {
	ms := &testutil.MockStore{}
	mb := testutil.NewMockBroker()
	ts := newMockedServer(t, ms, mb)

	bigBody := strings.Repeat("x", 1<<20+1)
	resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(bigBody))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-API-001] Dry-run ingest validates without persisting
func TestIngest_DryRun(t *testing.T) {
	ms := &testutil.MockStore{}
	mb := testutil.NewMockBroker()
	ts := newMockedServer(t, ms, mb)

	body := `{"eventId":"dry-1","sourceScenario":"src","eventType":"test.dryrun.v1"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for dry-run, got %d", resp.StatusCode)
	}

	result := decodeJSON[map[string]any](t, resp)
	if result["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got %v", result["dry_run"])
	}
	if result["eventId"] != "dry-1" {
		t.Fatalf("expected eventId=dry-1, got %v", result["eventId"])
	}

	// Verify store was NOT called
	if ms.InsertCallCount() != 0 {
		t.Fatalf("expected 0 insert calls for dry-run, got %d", ms.InsertCallCount())
	}
}
