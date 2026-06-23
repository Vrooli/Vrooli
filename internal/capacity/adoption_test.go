package capacity

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func kyutaiSpec() (ResourceClaimSpec, bool, error) {
	return ResourceClaimSpec{
		ResourceKind: ResourceKindVRAM, GPUIndex: gpu(0),
		PreferredBytes: 3 * gib, FloorBytes: 3 * gib, Priority: "service", Protected: true,
	}, true, nil
}

// declaredLoader serves a capacity block only for the named resources.
func declaredLoader(declared map[string]bool) SpecLoader {
	return func(name string) (ResourceClaimSpec, bool, error) {
		if declared[name] {
			return kyutaiSpec()
		}
		return ResourceClaimSpec{}, false, nil
	}
}

// An observed, declared, UNCLAIMED resident is adopted into the ledger.
func TestAdoptCreatesClaimForDeclaredUnclaimed(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	store := newTestStore(t, newFixedClock(now))

	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 2000, ProcessName: "python", UsedBytes: 3 * uint64(gib)})
	attr := fakeAttributor{2000: {ContainerName: "/vrooli-kyutai-stt-1", OwnerID: "kyutai-stt"}}

	created, err := AdoptObservedResidents(ctx, store, snap, attr, DefaultPolicy(), declaredLoader(map[string]bool{"kyutai-stt": true}), now)
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if len(created) != 1 || created[0].OwnerID != "kyutai-stt" {
		t.Fatalf("expected one kyutai-stt claim, got %+v", created)
	}
	if created[0].Status != StatusGranted || created[0].AmountBytes != 3*gib || !created[0].Protected {
		t.Errorf("adopted claim wrong: %+v", created[0])
	}
}

// A second sweep is a no-op (idempotent) — the owner already has an active claim.
func TestAdoptIsIdempotent(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	store := newTestStore(t, newFixedClock(now))
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 2000, ProcessName: "python", UsedBytes: 3 * uint64(gib)})
	attr := fakeAttributor{2000: {ContainerName: "/vrooli-kyutai-stt-1", OwnerID: "kyutai-stt"}}
	loader := declaredLoader(map[string]bool{"kyutai-stt": true})

	if _, err := AdoptObservedResidents(ctx, store, snap, attr, DefaultPolicy(), loader, now); err != nil {
		t.Fatalf("first adopt: %v", err)
	}
	created, err := AdoptObservedResidents(ctx, store, snap, attr, DefaultPolicy(), loader, now)
	if err != nil {
		t.Fatalf("second adopt: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("second sweep must not duplicate, created %+v", created)
	}
	all, _ := store.ListClaims(ctx, ClaimFilter{OwnerID: "kyutai-stt"})
	if len(all) != 1 {
		t.Fatalf("ledger should hold exactly one kyutai-stt claim, has %d", len(all))
	}
}

// An UNDECLARED observed consumer is never adopted (reconcile still warns).
func TestAdoptSkipsUndeclared(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	store := newTestStore(t, newFixedClock(now))
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 3000, ProcessName: "python", UsedBytes: 5 * uint64(gib)})
	attr := fakeAttributor{3000: {ContainerName: "/some-random-job", OwnerID: "random-job"}}

	created, err := AdoptObservedResidents(ctx, store, snap, attr, DefaultPolicy(), declaredLoader(map[string]bool{"kyutai-stt": true}), now)
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("undeclared consumer must not be adopted, created %+v", created)
	}
}
