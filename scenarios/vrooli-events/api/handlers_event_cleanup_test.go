package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

func TestDeleteObsoleteReceiptEventsMaintainsPayloadBytesAndSizePrune(t *testing.T) {
	eventStore, err := store.NewSQLiteStore(context.Background(), store.SQLiteConfig{MaxSizeBytes: 4})
	if err != nil {
		t.Fatalf("new event store: %v", err)
	}
	t.Cleanup(func() { _ = eventStore.Close() })

	srv := &Server{store: eventStore}

	ctx := context.Background()
	for _, event := range []store.Event{
		{EventID: "obsolete-receipt", EventType: obsoleteReceiptObservedEventType, Payload: []byte("1234567")},
		{EventID: "current-receipt", EventType: receiptEventType, Payload: []byte("abc")},
	} {
		if _, err := eventStore.Insert(ctx, event); err != nil {
			t.Fatalf("insert %s: %v", event.EventID, err)
		}
	}

	before, err := eventStore.Stats(ctx)
	if err != nil {
		t.Fatalf("stats before cleanup: %v", err)
	}
	if before.TotalEvents != 2 || before.TotalPayloadBytes != 10 {
		t.Fatalf("initial stats = %+v, want 2 events / 10 payload bytes", before)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/events?type="+obsoleteReceiptObservedEventType, nil)
	response := httptest.NewRecorder()
	srv.handleDeleteEventsByType(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("cleanup status = %d, want %d", response.Code, http.StatusOK)
	}
	var result struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode cleanup response: %v", err)
	}
	if result.Deleted != 1 {
		t.Fatalf("deleted = %d, want 1", result.Deleted)
	}

	after, err := eventStore.Stats(ctx)
	if err != nil {
		t.Fatalf("stats after cleanup: %v", err)
	}
	if after.TotalEvents != 1 || after.TotalPayloadBytes != 3 {
		t.Fatalf("stats after cleanup = %+v, want 1 event / 3 payload bytes", after)
	}

	pruned, err := eventStore.Prune(ctx)
	if err != nil {
		t.Fatalf("prune after cleanup: %v", err)
	}
	if pruned.SizeDeletedCount != 0 {
		t.Fatalf("size prune deleted %d events, want 0 below the 4-byte cap", pruned.SizeDeletedCount)
	}
	remaining, err := eventStore.Query(ctx, store.QueryFilters{})
	if err != nil {
		t.Fatalf("query after prune: %v", err)
	}
	if len(remaining) != 1 || remaining[0].EventID != "current-receipt" {
		encoded, _ := json.Marshal(remaining)
		t.Fatalf("remaining events = %s, want current-receipt", encoded)
	}

	// Keep this endpoint's domain restriction as part of the regression contract.
	invalidRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/events?type=other.event.v1", nil)
	invalidResponse := httptest.NewRecorder()
	srv.handleDeleteEventsByType(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid cleanup status = %d, want %d", invalidResponse.Code, http.StatusBadRequest)
	}
}
