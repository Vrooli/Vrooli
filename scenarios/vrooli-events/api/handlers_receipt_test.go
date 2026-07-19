package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// REQ: receipt observations require verified run correlation and a bounded,
// explicitly declared operation/outcome shape before durable ingestion.
func TestReceiptIngestionRequiresCorrelationAndOperation(t *testing.T) {
	s, _ := newTestServer(t)
	body := `{"eventId":"receipt-1","sourceScenario":"agent-manager","targetScenario":"plan-manager","eventType":"vrooli.receipt.observed.v1","metadata":{"outcome":"success"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleIngest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
