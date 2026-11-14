package scenarioruntime

import (
	"errors"
	"testing"
	"time"
)

func TestStartOperationFenceRecoveryRetainsOneOwnerBeyondUnlinkedFailures(t *testing.T) {
	store, err := NewSQLiteStore(t.Context(), Config{DBPath: t.TempDir() + "/runtime.db"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	owner, err := store.BeginStartOperation(t.Context(), StartOperation{Scenario: "alpha", FenceOperationID: "closed-fence"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkStartOperationAbandoned(t.Context(), owner.OperationID, "failed replacement"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < StartOperationKeepTerminal+3; i++ {
		op, err := store.BeginStartOperation(t.Context(), StartOperation{Scenario: "alpha"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.MarkStartOperationAbandoned(t.Context(), op.OperationID, "lost before negotiation"); err != nil {
			t.Fatal(err)
		}
	}
	retained, err := store.GetStartOperationFenceRecoveryOwner(t.Context(), "alpha", "live")
	if err != nil || retained.OperationID != owner.OperationID {
		t.Fatalf("retained owner = %+v, %v", retained, err)
	}
	var count int
	if err := store.db.QueryRowContext(t.Context(), "SELECT count(*) FROM runtime_start_operations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	// Pruning runs at begin, before the new current attempt becomes terminal.
	if count > StartOperationKeepTerminal+2 {
		t.Fatalf("unbounded recovery history: %d records", count)
	}
	if _, err := store.GetStartOperationFenceRecoveryOwner(t.Context(), "alpha", "preview"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-variant recovery = %v", err)
	}
	newest, err := store.BeginStartOperation(t.Context(), StartOperation{Scenario: "alpha", FenceOperationID: "new-fence"})
	if err != nil {
		t.Fatal(err)
	}
	newest.Status = StartOperationStatusSucceeded
	if _, err := store.UpdateStartOperation(t.Context(), newest); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetStartOperationFenceRecoveryOwner(t.Context(), "alpha", "live"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("successful linked successor resurrected failed owner: %v", err)
	}
}

func TestLatestStartOperationUsesAdmissionOrderAcrossClockRollback(t *testing.T) {
	store, err := NewSQLiteStore(t.Context(), Config{DBPath: t.TempDir() + "/runtime.db"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.BeginStartOperation(t.Context(), StartOperation{Scenario: "alpha", StartedAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	newest, err := store.BeginStartOperation(t.Context(), StartOperation{Scenario: "alpha", StartedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.GetLatestStartOperation(t.Context(), "alpha", "live")
	if err != nil || got.OperationID != newest.OperationID {
		t.Fatalf("latest = %+v, %v; want newly admitted %s", got, err, newest.OperationID)
	}
}
