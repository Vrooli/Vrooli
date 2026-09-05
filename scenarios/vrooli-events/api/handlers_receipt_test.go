package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// REQ: receipt observations require a bounded, explicitly declared
// operation/outcome shape before durable ingestion. Agent linkage is optional
// and is established only from verified Agent Manager identity.
func TestReceiptIngestionRequiresCorrelationAndOperation(t *testing.T) {
	s, _ := newTestServer(t)
	body, err := protojson.Marshal(&domain.EventEnvelope{EventId: "receipt-1", EventType: receiptEventType, OccurredAt: timestamppb.Now(), Source: &domain.EventSource{Scenario: "agent-manager"}})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleIngest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
