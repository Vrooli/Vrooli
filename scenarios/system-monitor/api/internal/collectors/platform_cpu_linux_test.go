//go:build linux

package collectors

import "testing"

func TestLinuxPerCoreMetricsRequiresStableCounterBaselines(t *testing.T) {
	state := platformCPUState{}
	first := linuxPerCoreMetrics(&state, map[string][]uint64{"cpu0": {10, 0, 10, 80, 0, 0, 0, 0}})
	if first["per_core_utilization_status"] != "not_yet_sampled" {
		t.Fatalf("first sample status = %v", first["per_core_utilization_status"])
	}
	second := linuxPerCoreMetrics(&state, map[string][]uint64{"cpu0": {20, 0, 20, 100, 0, 0, 0, 0}})
	perCore, ok := second["per_core_utilization"].(map[string]float64)
	if !ok || perCore["cpu0"] != 50 || second["core_imbalance_index"] != 0.0 {
		t.Fatalf("second sample = %#v", second)
	}
	reset := linuxPerCoreMetrics(&state, map[string][]uint64{"cpu0": {1, 0, 1, 1, 0, 0, 0, 0}})
	if reset["per_core_utilization_status"] != "not_yet_sampled" {
		t.Fatalf("counter reset status = %v", reset["per_core_utilization_status"])
	}
}

func TestLinuxPerCoreMetricsSuppressesChangedCoreSet(t *testing.T) {
	state := platformCPUState{perCore: map[string][]uint64{
		"cpu0": {10, 0, 10, 80, 0, 0, 0, 0},
	}}
	metrics := linuxPerCoreMetrics(&state, map[string][]uint64{
		"cpu1": {10, 0, 10, 80, 0, 0, 0, 0},
	})
	if metrics["per_core_utilization_status"] != "not_yet_sampled" {
		t.Fatalf("changed core set status = %v", metrics["per_core_utilization_status"])
	}
	if _, exists := metrics["per_core_utilization"]; exists {
		t.Fatalf("changed core set must not report usage: %#v", metrics)
	}
}
