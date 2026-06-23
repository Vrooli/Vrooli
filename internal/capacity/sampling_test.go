package capacity

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestDecayedPeak(t *testing.T) {
	t0 := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	hl := 10 * time.Minute

	// First sample (no prior): peak == observed.
	if got := DecayedPeak(0, 4*gib, time.Time{}, t0, hl); got != 4*gib {
		t.Errorf("first sample peak = %d, want %d", got, 4*gib)
	}
	// A new, larger observed sample always wins (working-set peak).
	if got := DecayedPeak(2*gib, 5*gib, t0, t0.Add(time.Minute), hl); got != 5*gib {
		t.Errorf("larger observed must win, got %d want %d", got, 5*gib)
	}
	// After exactly one half-life with a smaller observed, the prior peak halves
	// (and the decayed value still dominates the small observed).
	got := DecayedPeak(4*gib, 1*gib, t0, t0.Add(hl), hl)
	want := int64(2 * gib)
	if diff := got - want; diff > gib/100 || diff < -gib/100 {
		t.Errorf("one half-life decay = %d, want ~%d", got, want)
	}
	// The decayed peak never drops below the latest observed sample.
	if got := DecayedPeak(8*gib, 3*gib, t0, t0.Add(100*hl), hl); got != 3*gib {
		t.Errorf("fully decayed peak must floor at observed, got %d want %d", got, 3*gib)
	}
}

// TestSampleObservedUsagePersistsAndDecays proves a sweep sample records the
// observed footprint + peak, and that a later idle reading decays (not erases)
// the peak — the idle-snapshot trap.
func TestSampleObservedUsagePersistsAndDecays(t *testing.T) {
	ctx := context.Background()
	t0 := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	clk := newFixedClock(t0)
	store := newTestStore(t, clk)

	c := sampleClaim() // owner whisper, gpu 0
	created, err := store.CreateClaim(ctx, c, time.Hour)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	attr := fakeAttributor{1000: {ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "whisper", UsedBytes: 4 * uint64(gib)})
	policy := DefaultPolicy()

	// First sample at a real working footprint.
	if n := SampleObservedUsage(ctx, store, []CapacityClaim{created}, snap, attr, policy, t0); n != 1 {
		t.Fatalf("sampled %d claims, want 1", n)
	}
	got, _ := store.GetClaim(ctx, created.ClaimID)
	if got.ObservedBytes != 4*gib || got.ObservedPeakBytes != 4*gib {
		t.Fatalf("after sample observed=%d peak=%d, want 4GiB/4GiB", got.ObservedBytes, got.ObservedPeakBytes)
	}
	if got.Generation != created.Generation {
		t.Errorf("RecordObserved must NOT bump generation: %d -> %d", created.Generation, got.Generation)
	}

	// Later: whisper idle (no GPU process attributes to it). Observed drops to 0
	// but the peak only decays — it must not be erased by the idle reading.
	idleSnap := snapshotWithProcs()
	later := t0.Add(policy.ObservedPeakHalflife) // one half-life on
	SampleObservedUsage(ctx, store, []CapacityClaim{got}, idleSnap, attr, policy, later)
	got2, _ := store.GetClaim(ctx, created.ClaimID)
	if got2.ObservedBytes != 0 {
		t.Errorf("idle observed = %d, want 0", got2.ObservedBytes)
	}
	if got2.ObservedPeakBytes == 0 {
		t.Error("peak must not be erased by a lone idle reading")
	}
	if want := int64(2 * gib); abs(got2.ObservedPeakBytes-want) > gib/50 {
		t.Errorf("decayed peak = %d, want ~%d (half of 4GiB)", got2.ObservedPeakBytes, want)
	}
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
