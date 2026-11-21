package collectors

import (
	"fmt"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"testing"
	"time"
)

func TestDarwinCPUFourStateDeltasAndLoad(t *testing.T) {
	state := platformCPUState{}
	snapshot := hostinventory.Snapshot{ProbeStatuses: map[string]string{"load": "ok"}, Load: hostinventory.Load{Load1: 2, Load5: 1, Load15: 0.5, NormalizedLoad1: 1, NormalizedLoad5: 0.5}}
	first := darwinCPUReading(&state, snapshot, [][]uint64{{100, 100, 100, 100}, {100, 100, 100, 100}}, "measured", nil, time.Now())
	if first.status != "not_yet_sampled" || first.values["load_average_status"] != "measured" {
		t.Fatalf("first: %+v", first)
	}
	got := darwinCPUReading(&state, snapshot, [][]uint64{{120, 110, 160, 110}, {110, 120, 140, 130}}, "measured", nil, time.Now())
	if got.status != "measured" || got.usage != 50 {
		t.Fatalf("CPU reading: %+v", got)
	}
	modes := got.values["mode_breakdown"].(map[string]float64)
	if modes["idle"] != 50 || modes["nice"] != 20 || len(modes) != 4 {
		t.Fatalf("modes: %+v", modes)
	}
	perCore := got.values["per_core_utilization"].(map[string]float64)
	if perCore["cpu0"] != 40 || perCore["cpu1"] != 60 || got.values["core_imbalance_index"] != float64(20) {
		t.Fatalf("core values: %+v", got.values)
	}
	ensureCPUObservationStates(got.values, got)
	if _, exists := got.values["run_queue_depth"]; exists {
		t.Fatal("invented runnable-process count")
	}
	if got.values["run_queue_depth_status"] != "unsupported" {
		t.Fatal("run queue must be unsupported")
	}
}

func TestDarwinCPUUnavailableKeepsLoadAndStates(t *testing.T) {
	for _, status := range []string{"unsupported", "failed"} {
		state := platformCPUState{}
		got := darwinCPUReading(&state, hostinventory.Snapshot{ProbeStatuses: map[string]string{"load": "ok"}}, nil, status, fmt.Errorf("native backend unavailable"), time.Now())
		if got.status != status || got.values["mode_breakdown_status"] != status || got.values["per_core_utilization_status"] != status || got.values["load_average_status"] != "measured" {
			t.Fatalf("%s: %+v", status, got)
		}
	}
}

func TestDarwinCPURejectsPartialOrResetSamples(t *testing.T) {
	for _, next := range [][][]uint64{{{1, 2, 3}}, {{1, 2, 3, 4}}, {{110, 110, 110, 110}, {1, 1, 1, 1}}, {{100, 100, 100, 100}, {100, 100, 100, 100}}} {
		state := platformCPUState{}
		darwinCPUReading(&state, hostinventory.Snapshot{}, [][]uint64{{100, 100, 100, 100}, {100, 100, 100, 100}}, "measured", nil, time.Now())
		got := darwinCPUReading(&state, hostinventory.Snapshot{}, next, "measured", nil, time.Now())
		if got.status == "measured" {
			t.Fatalf("accepted invalid/noncomparable sample: %+v", next)
		}
	}
}

func TestCPUObservationStatesPreserveMeasuredRunQueue(t *testing.T) {
	for _, depth := range []int{0, 7} {
		values := map[string]interface{}{"run_queue_depth": depth}
		ensureCPUObservationStates(values, platformCPUReading{status: "measured", provenance: "native test backend"})
		if values["run_queue_depth_status"] != "measured" || values["run_queue_depth"] != depth {
			t.Fatalf("measured queue changed: %+v", values)
		}
	}
}
