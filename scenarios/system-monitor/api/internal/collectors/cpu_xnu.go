package collectors

import (
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

const darwinCPUProvenance = "Darwin Mach host_processor_info PROCESSOR_CPU_LOAD_INFO"

// darwinCPUReading consumes four unsigned tick counters per core in XNU order:
// user, system, idle, nice. Load availability is independent of CPU deltas.
func darwinCPUReading(state *platformCPUState, snapshot hostinventory.Snapshot, cores [][]uint64, status string, readErr error, now time.Time) platformCPUReading {
	values := map[string]interface{}{
		"run_queue_depth_status": "unsupported",
		"run_queue_depth_reason": "vm.loadavg has no runnable-process field",
		"load_average_status":    "unsupported",
		"load_average_reason":    "host inventory did not provide vm.loadavg",
	}
	loadStatus := snapshot.ProbeStatuses["load"]
	if loadStatus == "ok" {
		values["load_average_status"] = "measured"
		delete(values, "load_average_reason")
		values["load_average"] = []float64{snapshot.Load.Load1, snapshot.Load.Load5, snapshot.Load.Load15}
		values["normalized_load_1"] = snapshot.Load.NormalizedLoad1
		values["normalized_load_5"] = snapshot.Load.NormalizedLoad5
	} else if loadStatus == "failed" {
		values["load_average_status"] = "failed"
		values["load_average_reason"] = "host inventory vm.loadavg probe failed"
	}
	for _, key := range []string{"load_average", "normalized_load_1", "normalized_load_5"} {
		values[key+"_status"] = values["load_average_status"]
		values[key+"_provenance"] = "Darwin sysctl vm.loadavg"
		if reason, exists := values["load_average_reason"]; exists {
			values[key+"_reason"] = reason
		}
	}
	reading := platformCPUReading{status: status, provenance: darwinCPUProvenance, values: values}
	if readErr == nil && len(cores) == 0 {
		readErr = fmt.Errorf("host_processor_info returned no CPU counters")
		reading.status = "failed"
	}
	for _, core := range cores {
		if len(core) != 4 {
			readErr = fmt.Errorf("host_processor_info expected four CPU states")
			reading.status = "failed"
			break
		}
	}
	if readErr != nil {
		reading.reason = readErr.Error()
		for _, key := range []string{"mode_breakdown", "per_core_utilization", "core_imbalance_index"} {
			values[key+"_status"], values[key+"_reason"] = reading.status, reading.reason
		}
		// A failed probe must not compare a later reading with an old baseline.
		*state = platformCPUState{}
		return reading
	}
	if len(state.perCore) != len(cores) {
		*state = platformCPUState{}
	}
	total := make([]uint64, 4)
	for _, core := range cores {
		for i, ticks := range core {
			total[i] += ticks
		}
	}
	usage, modes, measured := deltaCPUValues(state, total, now, []string{"user", "system", "idle", "nice"}, 2)
	if state.perCore == nil {
		state.perCore = map[string][]uint64{}
	}
	perCore := map[string]float64{}
	min, max := 101.0, -1.0
	for i, core := range cores {
		name := fmt.Sprintf("cpu%d", i)
		prior := state.perCore[name]
		state.perCore[name] = append([]uint64(nil), core...)
		if len(prior) != 4 {
			continue
		}
		var sum, idle uint64
		valid := true
		for mode, tick := range core {
			if tick < prior[mode] {
				valid = false
				break
			}
			delta := tick - prior[mode]
			sum += delta
			if mode == 2 {
				idle = delta
			}
		}
		if !valid || sum == 0 {
			continue
		}
		value := float64(sum-idle) * 100 / float64(sum)
		perCore[name] = value
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	values["per_core_utilization_status"], values["core_imbalance_index_status"] = "not_yet_sampled", "not_yet_sampled"
	// Report imbalance only from a complete comparable processor set.
	if len(perCore) == len(cores) {
		values["per_core_utilization"], values["core_imbalance_index"] = perCore, max-min
		values["per_core_utilization_status"], values["core_imbalance_index_status"] = "measured", "measured"
	} else {
		measured = false
	}
	if !measured {
		reading.status, reading.reason = "not_yet_sampled", "CPU counters require two comparable samples; first sample, reset, wrap or processor-set change"
		values["mode_breakdown_status"] = "not_yet_sampled"
		return reading
	}
	reading.status, reading.usage = "measured", usage
	values["mode_breakdown"], values["mode_breakdown_status"] = modes, "measured"
	return reading
}
