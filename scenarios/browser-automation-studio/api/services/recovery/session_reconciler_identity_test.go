package recovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/database"
)

type identityBoundInventory struct {
	sessions []driver.ObservedSession
	closed   []driver.ObservedSession
}

func (i *identityBoundInventory) ListObservedSessions(context.Context) ([]driver.ObservedSession, error) {
	return append([]driver.ObservedSession(nil), i.sessions...), nil
}

func (i *identityBoundInventory) ForceCloseSession(_ context.Context, sessionID, ownerID, leaseID string) error {
	for _, session := range i.sessions {
		if session.ID == sessionID && session.OwnerExecutionID == ownerID && session.LeaseID == leaseID {
			i.closed = append(i.closed, session)
			return nil
		}
	}
	return errors.New("observed session owner or lease changed before force close")
}

type identityBoundExecutions map[uuid.UUID]*database.ExecutionIndex

func (e identityBoundExecutions) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	return e[id], nil
}

func TestSessionReconcilerRequiresCompleteDurableIdentityBeforeClose(t *testing.T) {
	ownerID := uuid.New()
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-time.Minute).Format(time.RFC3339Nano)
	inventory := &identityBoundInventory{sessions: []driver.ObservedSession{
		{ID: "exact-session", OwnerExecutionID: ownerID.String(), LeaseID: "exact-lease"},
		{OwnerExecutionID: ownerID.String(), LeaseID: "missing-session-id"},
		{ID: "missing-lease", OwnerExecutionID: ownerID.String()},
		{ID: "legacy-soak", OwnerExecutionID: "legacy-owner", WorkflowID: audioToolsSoakWorkflowID, LastUsedAt: stale},
	}}
	repo := identityBoundExecutions{
		ownerID: {ID: ownerID, Status: database.ExecutionStatusFailed, UpdatedAt: now.Add(-time.Hour)},
	}
	reconciler := newSessionReconciler(inventory, repo, nil, WithSessionTerminalGrace(0))
	reconciler.now = func() time.Time { return now }

	result, err := reconciler.ReconcileOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Observed != 4 || result.Closed != 1 || result.Skipped != 3 {
		t.Fatalf("identity reconciliation result = %+v, want observed=4 closed=1 skipped=3", result)
	}
	if len(inventory.closed) != 1 {
		t.Fatalf("driver owner received %d close effects, want exactly one", len(inventory.closed))
	}
	closed := inventory.closed[0]
	if closed.ID != "exact-session" || closed.OwnerExecutionID != ownerID.String() || closed.LeaseID != "exact-lease" {
		t.Fatalf("driver owner closed an unexpected durable lease: %+v", closed)
	}
}

func TestSessionReconcilerNeverGuessesAcrossReassignedLease(t *testing.T) {
	oldOwnerID, newOwnerID := uuid.New(), uuid.New()
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	// Simulate the independent driver assigning the session to a new run after
	// inventory read but before the administrative close reaches its owner.
	repo := identityBoundExecutions{
		oldOwnerID: {ID: oldOwnerID, Status: database.ExecutionStatusFailed, UpdatedAt: now.Add(-time.Hour)},
		newOwnerID: {ID: newOwnerID, Status: database.ExecutionStatusRunning, UpdatedAt: now},
	}
	// The inventory returned the old tuple; the owner now holds the new tuple.
	// Keep a separate observed snapshot so the fake models the exact race.
	observed := []driver.ObservedSession{{ID: "reused-session", OwnerExecutionID: oldOwnerID.String(), LeaseID: "old-lease"}}
	ownerAtEffect := &identityBoundInventory{sessions: []driver.ObservedSession{{ID: "reused-session", OwnerExecutionID: newOwnerID.String(), LeaseID: "new-lease"}}}
	reconciler := newSessionReconciler(reassignedInventory{observed: observed, owner: ownerAtEffect}, repo, nil, WithSessionTerminalGrace(0))
	reconciler.now = func() time.Time { return now }

	result, err := reconciler.ReconcileOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Closed != 0 || result.Skipped != 1 || len(ownerAtEffect.closed) != 0 || len(ownerAtEffect.sessions) != 1 || ownerAtEffect.sessions[0].OwnerExecutionID != newOwnerID.String() || ownerAtEffect.sessions[0].LeaseID != "new-lease" {
		t.Fatalf("stale reconciliation affected the reassigned lease: result=%+v owner=%+v", result, ownerAtEffect)
	}
}

type reassignedInventory struct {
	observed []driver.ObservedSession
	owner    *identityBoundInventory
}

func (i reassignedInventory) ListObservedSessions(context.Context) ([]driver.ObservedSession, error) {
	return append([]driver.ObservedSession(nil), i.observed...), nil
}

func (i reassignedInventory) ForceCloseSession(ctx context.Context, sessionID, ownerID, leaseID string) error {
	return i.owner.ForceCloseSession(ctx, sessionID, ownerID, leaseID)
}
