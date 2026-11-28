package persistence

import (
	"context"
	"encoding/json"
	"testing"
)

// TestHealthAlertStateRoundTripsAcrossStoreInstances [REQ:STC-P0-036] proves
// that transition context survives an API/store restart.
func TestHealthAlertStateRoundTripsAcrossStoreInstances(t *testing.T) {
	ctx := context.Background()
	db := openIdentityTestDB(t, "health-alert-state")
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	first := NewHealthAlertStateStore(repo.DB())
	expected := json.RawMessage(`[{"incident_id":"dep-1:certificate_expiring","status":"open","detected_at":"2026-11-19T12:01:00Z"}]`)
	if err := first.Save(ctx, "dep-1", expected); err != nil {
		t.Fatalf("save: %v", err)
	}
	restarted := NewHealthAlertStateStore(repo.DB())
	got, err := restarted.Load(ctx, "dep-1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(got) != string(expected) {
		t.Fatalf("round trip = %+v, want %+v", got, expected)
	}
	if err := restarted.Save(ctx, "dep-1", json.RawMessage("[]")); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, err = first.Load(ctx, "dep-1")
	if err != nil || string(got) != "[]" {
		t.Fatalf("cleared state = %+v, err=%v", got, err)
	}
}
