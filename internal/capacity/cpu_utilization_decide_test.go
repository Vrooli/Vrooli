package capacity

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// The defect: load average 75-100 on 32 cores while the CPUs were mostly idle
// (67% iowait, 53-64 tasks blocked on disk). Load counts blocked tasks, so every
// 250m job was denied "floor 250 exceeds the 0 free".
func thrashingHost(busy, iowait float64) hostinventory.Snapshot {
	return hostinventory.Snapshot{
		CPU: hostinventory.CPU{Cores: 32, Utilization: &hostinventory.CPUUtilization{Busy: busy, IOWait: iowait, WindowMS: 250, Source: "test"}},
		Load: hostinventory.Load{
			Load1: 90, NormalizedLoad1: 90.0 / 32,
		},
	}
}

func TestDecideGrantsCPUWhenLoadIsHighButUtilizationIsLow(t *testing.T) {
	snapshot := thrashingHost(0.20, 0.67)
	v := Decide(cpuRequest(250), snapshot, nil, DefaultPolicy(), hostAwareNow())
	if v.Kind != VerdictGrant {
		t.Fatalf("verdict = %q (%s); 80%% of the CPUs are idle, so a 250m job must be admitted", v.Kind, v.Reason)
	}
	used, warn := observedCPUMillis(snapshot)
	if used != 6400 || warn != "" {
		t.Fatalf("used = %d warn = %q, want 6400 (20%% of 32000) and no warning", used, warn)
	}
}

func TestDecideStillDeniesCPUWhenUtilizationIsTrulySaturated(t *testing.T) {
	v := Decide(cpuRequest(250), thrashingHost(0.999, 0), nil, DefaultPolicy(), hostAwareNow())
	if v.Kind == VerdictGrant {
		t.Fatalf("a host measured 99.9%% busy admitted 250m: %q (%s)", v.Kind, v.Reason)
	}
}

// IO wait is its own signal: a slow disk must not turn into "CPU used".
func TestIOWaitIsNeverCountedAsUsedCPU(t *testing.T) {
	snapshot := thrashingHost(0.05, 0.90)
	used, _ := observedCPUMillis(snapshot)
	if want := int64(0.05 * 32000); used != want {
		t.Fatalf("used = %d, want %d (busy only)", used, want)
	}
}

// Utilization wins over load whichever is larger, in both directions.
func TestMeasuredUtilizationOverridesLoadAverage(t *testing.T) {
	lowLoadHighBusy := hostinventory.Snapshot{
		CPU:  hostinventory.CPU{Cores: 8, Utilization: &hostinventory.CPUUtilization{Busy: 0.95}},
		Load: hostinventory.Load{NormalizedLoad1: 0.05},
	}
	if used, _ := observedCPUMillis(lowLoadHighBusy); used != 7600 {
		t.Fatalf("used = %d, want 7600: measured busy must not be hidden by a low load average", used)
	}
}

func TestLoadAverageFallbackIsLabelled(t *testing.T) {
	snapshot := hostinventory.Snapshot{
		CPU:  hostinventory.CPU{Cores: 4},
		Load: hostinventory.Load{NormalizedLoad1: 0.5},
	}
	used, warn := observedCPUMillis(snapshot)
	if used != 2000 {
		t.Fatalf("fallback used = %d, want 2000", used)
	}
	if !strings.Contains(warn, "not measured") {
		t.Fatalf("fallback warning = %q, want it to say utilization is not measured", warn)
	}
}
