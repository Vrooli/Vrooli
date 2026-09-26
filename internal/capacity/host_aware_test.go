package capacity

import (
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// These cover the signals admission already received on every snapshot and
// discarded: Load and Swap. Before this, CPU "used" was a literal 0, so a
// saturated host admitted work exactly as readily as an idle one.

func hostAwareNow() time.Time { return time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC) }

func cpuRequest(milli int64) CapacityRequest {
	return CapacityRequest{
		OwnerKind:      OwnerKindScenario,
		OwnerID:        "test-genie",
		ResourceKind:   ResourceKindCPU,
		PreferredBytes: milli,
		FloorBytes:     milli,
		Priority:       PriorityBatch,
	}
}

func TestDecideGrantsCPUOnAnIdleHost(t *testing.T) {
	snapshot := hostinventory.Snapshot{
		CPU:  hostinventory.CPU{Cores: 8},
		Load: hostinventory.Load{NormalizedLoad1: 0.05},
	}
	// 8 cores = 8000 millicores; a 2-core request fits easily on an idle host.
	v := Decide(cpuRequest(2000), snapshot, nil, DefaultPolicy(), hostAwareNow())
	if v.Kind != VerdictGrant {
		t.Fatalf("idle host verdict = %q (%s), want grant", v.Kind, v.Reason)
	}
}

// TestDecideWithholdsCPUOnALoadedHost is the behaviour change that matters:
// the same request on the same hardware must not be granted when the host is
// already saturated.
func TestDecideWithholdsCPUOnALoadedHost(t *testing.T) {
	req := cpuRequest(4000) // half the host
	idle := hostinventory.Snapshot{
		CPU:  hostinventory.CPU{Cores: 8},
		Load: hostinventory.Load{NormalizedLoad1: 0.05},
	}
	loaded := hostinventory.Snapshot{
		CPU: hostinventory.CPU{Cores: 8},
		// Run queue at 0.95 per core: 7600 of 8000 millicores are spoken for.
		Load: hostinventory.Load{NormalizedLoad1: 0.95},
	}

	if v := Decide(req, idle, nil, DefaultPolicy(), hostAwareNow()); v.Kind != VerdictGrant {
		t.Fatalf("idle verdict = %q (%s), want grant", v.Kind, v.Reason)
	}
	v := Decide(req, loaded, nil, DefaultPolicy(), hostAwareNow())
	if v.Kind == VerdictGrant {
		t.Fatalf("a saturated host granted %d millicores; measured load was ignored", req.PreferredBytes)
	}
}

func TestDecideFallsBackToRawLoadWhenNormalizedIsAbsent(t *testing.T) {
	// Some collectors fill Load1 but not NormalizedLoad1.
	loaded := hostinventory.Snapshot{
		CPU:  hostinventory.CPU{Cores: 4},
		Load: hostinventory.Load{Load1: 3.9},
	}
	v := Decide(cpuRequest(3000), loaded, nil, DefaultPolicy(), hostAwareNow())
	if v.Kind == VerdictGrant {
		t.Fatalf("raw Load1 was ignored: verdict = %q", v.Kind)
	}
}

// TestDecideDegradesVisiblyWhenLoadIsUnknown keeps the old behaviour available
// but no longer silent: a snapshot with no load signal is treated as idle and
// says so.
func TestDecideDegradesVisiblyWhenLoadIsUnknown(t *testing.T) {
	snapshot := hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 4}}
	used, warn := observedCPUMillis(snapshot)
	if used != 0 {
		t.Fatalf("unknown load should report 0 used, got %d", used)
	}
	if warn == "" {
		t.Fatal("an unknown host load must be reported, not assumed silently")
	}
}

func TestObservedCPUIsClampedToCapacity(t *testing.T) {
	// A run queue above 1.0 per core means processes are waiting. "More than
	// fully busy" is still just fully busy for remaining-capacity purposes.
	snapshot := hostinventory.Snapshot{
		CPU:  hostinventory.CPU{Cores: 2},
		Load: hostinventory.Load{NormalizedLoad1: 6.0},
	}
	used, _ := observedCPUMillis(snapshot)
	if used != 2000 {
		t.Fatalf("used = %d, want it clamped to the 2000 millicore total", used)
	}
}

// --- swap pressure ------------------------------------------------------

func ramRequest(bytes int64) CapacityRequest {
	return CapacityRequest{
		OwnerKind:      OwnerKindScenario,
		OwnerID:        "test-genie",
		ResourceKind:   ResourceKindRAM,
		PreferredBytes: bytes,
		FloorBytes:     bytes,
		Priority:       PriorityBatch,
	}
}

func TestDecideDeniesRAMUnderSwapPressure(t *testing.T) {
	const gib = int64(1) << 30
	// RAM looks healthy — which is exactly the trap. Once pages are on disk,
	// AvailableBytes can read fine while the machine thrashes.
	snapshot := hostinventory.Snapshot{
		Memory: hostinventory.Memory{TotalBytes: 32 * uint64(gib), AvailableBytes: 16 * uint64(gib)},
		Swap:   hostinventory.Swap{TotalBytes: 8 * uint64(gib), FreeBytes: 1 * uint64(gib)}, // 87.5% used
	}
	v := Decide(ramRequest(gib), snapshot, nil, DefaultPolicy(), hostAwareNow())
	if v.Kind != VerdictDeny {
		t.Fatalf("verdict = %q (%s), want deny under swap pressure", v.Kind, v.Reason)
	}
	if v.Reason == "" {
		t.Fatal("a denial must say why")
	}
}

func TestDecideGrantsRAMWhenSwapIsHealthy(t *testing.T) {
	const gib = int64(1) << 30
	snapshot := hostinventory.Snapshot{
		Memory: hostinventory.Memory{TotalBytes: 32 * uint64(gib), AvailableBytes: 16 * uint64(gib)},
		Swap:   hostinventory.Swap{TotalBytes: 8 * uint64(gib), FreeBytes: 8 * uint64(gib)},
	}
	if v := Decide(ramRequest(gib), snapshot, nil, DefaultPolicy(), hostAwareNow()); v.Kind != VerdictGrant {
		t.Fatalf("verdict = %q (%s), want grant", v.Kind, v.Reason)
	}
}

// TestSwapPressureEdgeCases pins the two cases that would otherwise produce a
// wrong answer rather than a conservative one.
func TestSwapPressureEdgeCases(t *testing.T) {
	// A host with no swap configured is not under swap pressure. Reading the
	// fraction here would divide by zero.
	noSwap := hostinventory.Snapshot{Swap: hostinventory.Swap{}}
	if SwapPressure(noSwap, DefaultSwapPressureThreshold) {
		t.Fatal("a host with no swap must not report swap pressure")
	}

	// A zero threshold disables the check rather than denying everything.
	full := hostinventory.Snapshot{Swap: hostinventory.Swap{TotalBytes: 100, FreeBytes: 0}}
	if SwapPressure(full, 0) {
		t.Fatal("a zero threshold must disable the check")
	}
	if !SwapPressure(full, DefaultSwapPressureThreshold) {
		t.Fatal("fully consumed swap must report pressure")
	}
}

func TestSwapPressureThresholdIsTunable(t *testing.T) {
	p := DefaultPolicy()
	got, err := p.Get("swap_pressure_threshold")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == "" {
		t.Fatal("swap_pressure_threshold must be readable")
	}
	updated, err := p.withKey("swap_pressure_threshold", "80")
	if err != nil {
		t.Fatalf("withKey: %v", err)
	}
	if updated.SwapPressureThresholdPct != 80 {
		t.Fatalf("threshold = %d, want 80", updated.SwapPressureThresholdPct)
	}
	if _, err := p.withKey("swap_pressure_threshold", "101"); err == nil {
		t.Fatal("a percent above 100 must be rejected")
	}
	if _, err := p.withKey("swap_pressure_threshold", "-1"); err == nil {
		t.Fatal("a negative percent must be rejected")
	}
}
