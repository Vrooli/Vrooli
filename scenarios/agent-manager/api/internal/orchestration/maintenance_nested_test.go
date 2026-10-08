package orchestration

import (
	"context"
	"errors"
	"sync"
	"testing"

	"agent-manager/internal/maintenance"
	"agent-manager/internal/orchestration/testutil"
)

func TestMaintenanceNestedAdmissionIsOwnerBoundAndRevoked(t *testing.T) {
	db, closeDB := testutil.SetupTestDB(t)
	t.Cleanup(closeDB)
	gate := maintenance.NewGate(maintenance.NewRepository(db))
	o := &Orchestrator{maintenanceGate: gate}
	other := &Orchestrator{maintenanceGate: gate}
	ctx, release, err := o.admitMaintenanceContext(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := gate.Enter(t.Context(), "fixture-owner", "close during admitted operation"); err != nil {
		t.Fatal(err)
	}
	for _, nestedCtx := range []context.Context{ctx, context.WithoutCancel(ctx)} {
		nestedRelease, err := o.admitMaintenance(nestedCtx)
		if err != nil {
			t.Fatalf("already-admitted owner was refused by its nested gate: %v", err)
		}
		nestedRelease()
	}
	childCtx, childRelease, err := o.admitMaintenanceContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	childRelease()
	standing, err := gate.Status(t.Context())
	if err != nil || standing.Admitting != 1 || !o.hasMaintenanceAdmission(ctx) {
		t.Fatalf("nested admission released or duplicated the outer hold: %+v %v", standing, err)
	}
	if _, err := other.admitMaintenance(ctx); !errors.Is(err, maintenance.ErrClosed) {
		t.Fatalf("another owner borrowed the private admission: %v", err)
	}
	if _, _, err := other.admitMaintenanceContext(ctx); !errors.Is(err, maintenance.ErrClosed) {
		t.Fatalf("another owner extended the private admission: %v", err)
	}
	if _, err := o.admitMaintenance(t.Context()); !errors.Is(err, maintenance.ErrClosed) {
		t.Fatalf("unmarked request escaped maintenance: %v", err)
	}
	// Release may be called by multiple cleanup paths. It revokes every derived
	// context, including ones that retain values after cancellation is removed.
	var releases sync.WaitGroup
	for range 8 {
		releases.Go(release)
	}
	releases.Wait()
	standing, err = gate.Status(t.Context())
	if err != nil || standing.Admitting != 0 || o.hasMaintenanceAdmission(ctx) {
		t.Fatalf("release did not revoke exactly one admission: %+v %v", standing, err)
	}
	for _, expiredCtx := range []context.Context{ctx, childCtx, context.WithoutCancel(ctx)} {
		if _, err := o.admitMaintenance(expiredCtx); !errors.Is(err, maintenance.ErrClosed) {
			t.Fatalf("released context bypassed maintenance: %v", err)
		}
		if _, _, err := o.admitMaintenanceContext(expiredCtx); !errors.Is(err, maintenance.ErrClosed) {
			t.Fatalf("released context renewed a closed admission: %v", err)
		}
	}
}
