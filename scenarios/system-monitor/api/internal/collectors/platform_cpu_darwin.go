//go:build darwin

package collectors

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"golang.org/x/sys/unix"
)

func collectPlatformCPU(_ context.Context, c *CPUCollector, snapshot hostinventory.Snapshot) platformCPUReading {
	raw, err := unix.SysctlRaw("kern.cp_time")
	if err != nil {
		return platformCPUReading{status: "failed", reason: fmt.Sprintf("kern.cp_time: %v", err), provenance: "darwin sysctl kern.cp_time"}
	}
	counters, err := darwinCPUCounters(raw)
	if err != nil {
		return platformCPUReading{status: "failed", reason: err.Error(), provenance: "darwin sysctl kern.cp_time"}
	}
	now := time.Now()
	usage, modes, measured := deltaCPUValues(&c.platformState, counters, now, []string{"user", "nice", "system", "interrupt", "idle"}, 4)
	if !measured {
		return platformCPUReading{status: "not_yet_sampled", reason: "CPU counter delta requires a previous valid sample", provenance: "darwin sysctl kern.cp_time", values: map[string]interface{}{"mode_breakdown_status": "not_yet_sampled"}}
	}
	values := map[string]interface{}{"mode_breakdown": modes, "load_average_status": "unsupported", "load_average_reason": "Darwin load is collected by host inventory", "mode_breakdown_status": "measured"}
	if perCoreRaw, perCoreErr := unix.SysctlRaw("kern.cp_times"); perCoreErr == nil {
		if perCoreCounters, parseErr := darwinPerCoreCounters(perCoreRaw); parseErr == nil {
			perCore, imbalance, ok := darwinPerCoreUsage(&c.platformState, perCoreCounters)
			if ok {
				values["per_core_utilization"] = perCore
				values["core_imbalance_index"] = imbalance
				values["per_core_utilization_status"] = "measured"
				values["core_imbalance_index_status"] = "measured"
			} else {
				values["per_core_utilization_status"] = "not_yet_sampled"
				values["core_imbalance_index_status"] = "not_yet_sampled"
			}
		} else {
			values["per_core_utilization_status"] = "failed"
			values["per_core_utilization_reason"] = parseErr.Error()
			values["core_imbalance_index_status"] = "failed"
			values["core_imbalance_index_reason"] = parseErr.Error()
		}
	} else {
		values["per_core_utilization_status"] = "unsupported"
		values["per_core_utilization_reason"] = "kern.cp_times is unavailable"
		values["core_imbalance_index_status"] = "unsupported"
		values["core_imbalance_index_reason"] = "kern.cp_times is unavailable"
	}
	if snapshot.ProbeStatuses != nil && snapshot.ProbeStatuses["load"] == "ok" {
		values["load_average_status"] = "measured"
		values["load_average"] = []float64{snapshot.Load.Load1, snapshot.Load.Load5, snapshot.Load.Load15}
		values["normalized_load_1"] = snapshot.Load.NormalizedLoad1
		values["normalized_load_5"] = snapshot.Load.NormalizedLoad5
		values["run_queue_depth"] = snapshot.Load.RunningProcs
	}
	return platformCPUReading{usage: usage, status: "measured", provenance: "darwin sysctl kern.cp_time", values: values}
}

func darwinPerCoreCounters(raw []byte) ([][]uint64, error) {
	if len(raw) == 0 || len(raw)%40 != 0 && len(raw)%20 != 0 {
		return nil, fmt.Errorf("kern.cp_times returned %d bytes; expected five counters per core", len(raw))
	}
	wordSize := 8
	if len(raw)%40 != 0 {
		wordSize = 4
	}
	words := len(raw) / wordSize
	cores := make([][]uint64, 0, words/5)
	for offset := 0; offset < words; offset += 5 {
		values := make([]uint64, 5)
		for i := range values {
			if wordSize == 8 {
				values[i] = binary.LittleEndian.Uint64(raw[(offset+i)*wordSize:])
			} else {
				values[i] = uint64(binary.LittleEndian.Uint32(raw[(offset+i)*wordSize:]))
			}
		}
		cores = append(cores, values)
	}
	return cores, nil
}

func darwinPerCoreUsage(state *platformCPUState, counters [][]uint64) (map[string]float64, float64, bool) {
	if state.perCore == nil {
		state.perCore = map[string][]uint64{}
	}
	values := map[string]float64{}
	max, min := -1.0, 101.0
	for i, current := range counters {
		name := fmt.Sprintf("cpu%d", i)
		prior := state.perCore[name]
		state.perCore[name] = append([]uint64(nil), current...)
		if len(prior) != len(current) {
			continue
		}
		var total, idle uint64
		for index, value := range current {
			if value < prior[index] {
				total = 0
				break
			}
			delta := value - prior[index]
			total += delta
			if index == 4 {
				idle += delta
			}
		}
		if total == 0 {
			continue
		}
		usage := float64(total-idle) * 100 / float64(total)
		values[name] = usage
		if usage > max {
			max = usage
		}
		if usage < min {
			min = usage
		}
	}
	if len(values) == 0 {
		return nil, 0, false
	}
	return values, max - min, true
}

func darwinCPUCounters(raw []byte) ([]uint64, error) {
	if len(raw) >= 5*8 {
		counters := make([]uint64, len(raw)/8)
		for i := range counters {
			counters[i] = binary.LittleEndian.Uint64(raw[i*8:])
		}
		return counters, nil
	}
	if len(raw) >= 5*4 {
		counters := make([]uint64, len(raw)/4)
		for i := range counters {
			counters[i] = uint64(binary.LittleEndian.Uint32(raw[i*4:]))
		}
		return counters, nil
	}
	return nil, fmt.Errorf("kern.cp_time returned %d bytes; expected at least 20", len(raw))
}
