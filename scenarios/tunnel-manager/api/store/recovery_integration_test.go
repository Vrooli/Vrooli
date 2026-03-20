//go:build integration

package store_test

import (
	"testing"

	"tunnel-manager/domain"
	"tunnel-manager/store"
	"tunnel-manager/testutil"
)

// [REQ:RECOVER-006] Recovery event persistence at store level

func TestRecoveryStore_PersistAndList(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	rs := store.NewRecoveryStore(db)

	// Persist an event
	evt := &domain.RecoveryEvent{
		TriggerType: "ready_failure",
		Action:      "systemctl_restart",
		Outcome:     "success",
		Details:     "test recovery",
	}
	if err := rs.PersistEvent(evt); err != nil {
		t.Fatalf("PersistEvent: %v", err)
	}

	// List events
	events, err := rs.ListEvents(10)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one persisted event")
	}

	found := events[0]
	if found.TriggerType != "ready_failure" {
		t.Errorf("trigger_type = %q, want ready_failure", found.TriggerType)
	}
	if found.Action != "systemctl_restart" {
		t.Errorf("action = %q, want systemctl_restart", found.Action)
	}
	if found.Outcome != "success" {
		t.Errorf("outcome = %q, want success", found.Outcome)
	}
	if found.Details != "test recovery" {
		t.Errorf("details = %q, want %q", found.Details, "test recovery")
	}
}

func TestRecoveryStore_ListEventsLimit(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	rs := store.NewRecoveryStore(db)

	// Persist multiple events
	for i := 0; i < 5; i++ {
		if err := rs.PersistEvent(&domain.RecoveryEvent{
			TriggerType: "ready_failure",
			Action:      "systemctl_restart",
			Outcome:     "success",
		}); err != nil {
			t.Fatalf("PersistEvent[%d]: %v", i, err)
		}
	}

	events, err := rs.ListEvents(3)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestRecoveryStore_PersistEventNilDB(t *testing.T) {
	rs := store.NewRecoveryStore(nil)

	// Should not panic and should return nil (nil db short-circuits)
	if err := rs.PersistEvent(&domain.RecoveryEvent{
		TriggerType: "test",
		Action:      "test",
		Outcome:     "success",
	}); err != nil {
		t.Errorf("expected nil error for nil DB, got: %v", err)
	}
}
