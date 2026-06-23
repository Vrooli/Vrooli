package capacity

import (
	"context"
	"testing"
	"time"
)

func TestDefaultPolicyIdleYieldFloorIsBatch(t *testing.T) {
	if got := DefaultPolicy().IdleYieldFloor; got != PriorityBatch {
		t.Errorf("default idle_yield_floor = %d, want batch (%d)", got, PriorityBatch)
	}
}

func TestPolicyIdleYieldFloorRoundTrip(t *testing.T) {
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)))
	ctx := context.Background()

	// Set via tier name; the stored canonical value is the tier name.
	if _, err := store.SetPolicyKey(ctx, "idle_yield_floor", "service"); err != nil {
		t.Fatalf("set idle_yield_floor: %v", err)
	}
	pol, err := store.GetPolicy(ctx)
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if pol.IdleYieldFloor != PriorityService {
		t.Errorf("idle_yield_floor = %d, want service (%d)", pol.IdleYieldFloor, PriorityService)
	}
	if got, _ := pol.Get("idle_yield_floor"); got != "service" {
		t.Errorf("Get(idle_yield_floor) = %q, want service", got)
	}
}

func TestPolicyIdleYieldFloorRejectsGarbage(t *testing.T) {
	if _, err := DefaultPolicy().withKey("idle_yield_floor", "turbo"); err == nil {
		t.Error("idle_yield_floor must reject an unknown tier")
	}
}

func TestPolicyKeysIncludesIdleYieldFloor(t *testing.T) {
	found := false
	for _, k := range PolicyKeys {
		if k == "idle_yield_floor" {
			found = true
		}
	}
	if !found {
		t.Error("PolicyKeys must include idle_yield_floor so `policy get` lists it")
	}
}
