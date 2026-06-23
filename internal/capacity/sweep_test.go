package capacity

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// residentClaim is a resident (resource) VRAM claim for the named owner.
func residentClaim(owner string) CapacityClaim {
	return CapacityClaim{
		OwnerKind:      OwnerKindResource,
		OwnerID:        owner,
		ResourceKind:   ResourceKindVRAM,
		GPUIndex:       gpu(0),
		AmountBytes:    3 << 30,
		PreferredBytes: 3 << 30,
		FloorBytes:     1 << 30,
		Priority:       PriorityService,
	}
}

// A resident claim whose owner is still observed on the GPU is heartbeated, not
// expired — even after its deadline would otherwise have lapsed.
func TestSweepRefreshesObservedResidentClaim(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	created, err := store.CreateClaim(ctx, residentClaim("whisper"), DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}

	// Time advances past the original deadline; without a sweep the claim would
	// be expired. whisper is still observed holding VRAM.
	clk.Advance(2 * DefaultHeartbeatTTL)
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "whisper", UsedBytes: 3 * uint64(gib)})
	attr := fakeAttributor{1000: {ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}

	result, err := Sweep(ctx, store, snap, attr, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if len(result.Refreshed) != 1 || result.Refreshed[0].ClaimID != created.ClaimID {
		t.Fatalf("refreshed = %+v, want [%s]", result.Refreshed, created.ClaimID)
	}
	if len(result.Expired) != 0 {
		t.Fatalf("expired = %+v, want none (observed owner rescued)", result.Expired)
	}

	got, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.Status != StatusGranted {
		t.Errorf("status = %q, want granted (refreshed)", got.Status)
	}
	wantDeadline := clk.Now().Add(DefaultHeartbeatTTL)
	if got.HeartbeatDeadlineAt == nil || !got.HeartbeatDeadlineAt.Equal(wantDeadline) {
		t.Errorf("deadline = %v, want %v", got.HeartbeatDeadlineAt, wantDeadline)
	}
	// Heartbeat must NOT bump the generation (a re-read signal only).
	if got.Generation != created.Generation {
		t.Errorf("generation = %d, want %d (heartbeat must not bump)", got.Generation, created.Generation)
	}
}

// A resident claim whose owner is no longer observed and whose deadline lapsed
// is swept to expired.
func TestSweepExpiresUnobservedResidentClaim(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	created, err := store.CreateClaim(ctx, residentClaim("kyutai-stt"), DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	clk.Advance(2 * DefaultHeartbeatTTL)

	// No GPU process attributes to kyutai-stt (container gone).
	snap := snapshotWithProcs()
	result, err := Sweep(ctx, store, snap, fakeAttributor{}, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if len(result.Refreshed) != 0 {
		t.Fatalf("refreshed = %+v, want none", result.Refreshed)
	}
	if len(result.Expired) != 1 || result.Expired[0].ClaimID != created.ClaimID {
		t.Fatalf("expired = %+v, want [%s]", result.Expired, created.ClaimID)
	}
	got, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.Status != StatusExpired {
		t.Errorf("status = %q, want expired", got.Status)
	}
}

// Op-scoped claims own their own claim->run->release lifecycle and are NEVER
// presence-refreshed, even when their owner process is observed.
func TestSweepIgnoresOpScopedClaims(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	opClaim := residentClaim("image-tools:job-1")
	opClaim.OwnerKind = OwnerKindOp
	created, err := store.CreateClaim(ctx, opClaim, DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	clk.Advance(2 * DefaultHeartbeatTTL)

	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 2000, ProcessName: "sd", UsedBytes: 3 * uint64(gib)})
	attr := fakeAttributor{2000: {ContainerName: "/vrooli-image-tools-1", OwnerID: "image-tools"}}

	result, err := Sweep(ctx, store, snap, attr, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if len(result.Refreshed) != 0 {
		t.Fatalf("refreshed = %+v, want none (op claims not presence-refreshed)", result.Refreshed)
	}
	// The op claim's deadline lapsed and it was not refreshed, so it expires.
	if len(result.Expired) != 1 || result.Expired[0].ClaimID != created.ClaimID {
		t.Fatalf("expired = %+v, want [%s]", result.Expired, created.ClaimID)
	}
}

// Sweep with a nil attributor never panics and refreshes nothing.
func TestSweepNilAttributorSafe(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)
	if _, err := store.CreateClaim(ctx, residentClaim("whisper"), DefaultHeartbeatTTL); err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "whisper", UsedBytes: 3 * uint64(gib)})
	result, err := Sweep(ctx, store, snap, nil, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if len(result.Refreshed) != 0 {
		t.Errorf("refreshed = %+v, want none (nil attributor => unknown owners)", result.Refreshed)
	}
}
