package handlers

import (
	"slices"
	"testing"
)

func TestDebugModeSnapshotSharedAcrossSetAndGetCallers(t *testing.T) {
	defer SetDebugModeState(false, nil, 0)
	components := []string{"recording", "execution"}
	set := SetDebugModeState(true, components, 1)
	get := GetDebugModeSnapshot()
	if !set.Enabled || !get.Enabled {
		t.Fatalf("enabled states = set:%v get:%v; want both enabled", set.Enabled, get.Enabled)
	}
	if !slices.Equal(set.Components, get.Components) || !slices.Equal(get.Components, components) {
		t.Fatalf("component snapshots differ: set=%v get=%v", set.Components, get.Components)
	}
	if !set.ExpiresAt.Equal(get.ExpiresAt) || set.ExpiresAt.IsZero() {
		t.Fatalf("expiry snapshots differ: set=%s get=%s", set.ExpiresAt, get.ExpiresAt)
	}
	if delta := set.RemainingMins - get.RemainingMins; delta < -1 || delta > 1 {
		t.Fatalf("remaining-minute snapshots differ unexpectedly: set=%d get=%d", set.RemainingMins, get.RemainingMins)
	}

	disabled := SetDebugModeState(false, nil, 0)
	readback := GetDebugModeSnapshot()
	if disabled.Enabled || readback.Enabled || !disabled.ExpiresAt.IsZero() || !readback.ExpiresAt.IsZero() {
		t.Fatalf("disabled snapshots inconsistent: set=%+v get=%+v", disabled, readback)
	}
}
