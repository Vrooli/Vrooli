//go:build linux

package collectors

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func collectPlatformCPU(_ context.Context, c *CPUCollector, snapshot hostinventory.Snapshot) platformCPUReading {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return platformCPUReading{status: "failed", reason: fmt.Sprintf("read /proc/stat: %v", err), provenance: "linux /proc/stat"}
	}
	var counters []uint64
	var ctxt, intr uint64
	coreCounters := map[string][]uint64{}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch {
		case fields[0] == "cpu":
			counters = parseCPUFields(fields[1:])
			if len(counters) > 8 {
				counters = counters[:8]
			}
		case strings.HasPrefix(fields[0], "cpu") && len(fields[0]) > 3:
			if parsed := parseCPUFields(fields[1:]); len(parsed) >= 8 {
				coreCounters[fields[0]] = parsed[:8]
			}
		case fields[0] == "ctxt":
			ctxt, _ = strconv.ParseUint(fields[1], 10, 64)
		case fields[0] == "intr":
			intr, _ = strconv.ParseUint(fields[1], 10, 64)
		}
	}
	now := time.Now()
	names := []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq", "steal"}
	usage, modes, measured := deltaCPUValues(&c.platformState, counters, now, names, 3, 4)
	values := map[string]interface{}{"mode_breakdown": modes, "load_average_status": loadStatus(snapshot), "load_average_provenance": loadProvenance(snapshot)}
	if c.platformState.perCore == nil {
		c.platformState.perCore = map[string][]uint64{}
	}
	coreSetChanged := len(c.platformState.perCore) > 0 && len(c.platformState.perCore) != len(coreCounters)
	if !coreSetChanged {
		for name := range c.platformState.perCore {
			if _, exists := coreCounters[name]; !exists {
				coreSetChanged = true
				break
			}
		}
	}
	perCore := map[string]float64{}
	busiest, least := -1.0, 101.0
	for name, current := range coreCounters {
		prior := c.platformState.perCore[name]
		c.platformState.perCore[name] = append([]uint64(nil), current...)
		if coreSetChanged || len(prior) != len(current) {
			continue
		}
		var total, idle uint64
		reset := false
		for i, v := range current {
			if v < prior[i] {
				reset = true
				break
			}
			d := v - prior[i]
			total += d
			if i == 3 || i == 4 {
				idle += d
			}
		}
		if reset || total == 0 {
			continue
		}
		u := float64(total-idle) * 100 / float64(total)
		perCore[name] = u
		if u > busiest {
			busiest = u
		}
		if u < least {
			least = u
		}
	}
	if coreSetChanged {
		values["per_core_utilization_status"] = "not_yet_sampled"
		values["per_core_utilization_reason"] = "CPU core set changed between samples"
		values["core_imbalance_index_status"] = "not_yet_sampled"
		values["core_imbalance_index_reason"] = "CPU core set changed between samples"
	} else if len(perCore) > 0 {
		values["per_core_utilization"] = perCore
		values["core_imbalance_index"] = busiest - least
		values["per_core_utilization_status"] = "measured"
		values["core_imbalance_index_status"] = "measured"
	} else {
		values["per_core_utilization_status"] = "not_yet_sampled"
		values["core_imbalance_index_status"] = "not_yet_sampled"
	}
	quotaValues, quotaStatus, quotaReason := linuxQuotaEvidence(c, now)
	for key, value := range quotaValues {
		values[key] = value
	}
	values["quota_throttling_status"], values["quota_throttling_reason"] = quotaStatus, quotaReason
	frequencyValue, frequencyStatus, frequencyReason := linuxFrequencyEvidence()
	if frequencyStatus == "measured" {
		values["frequency_derate_ratio"] = frequencyValue
	}
	values["frequency_derate_ratio_status"], values["frequency_derate_ratio_reason"] = frequencyStatus, frequencyReason
	values["thermal_throttle_evidence_status"] = "unsupported"
	values["thermal_throttle_evidence_reason"] = "thermal-to-CPU attribution backend is not enabled"
	if loadStatus(snapshot) == "measured" {
		values["load_average"] = []float64{snapshot.Load.Load1, snapshot.Load.Load5, snapshot.Load.Load15}
		values["normalized_load_1"] = snapshot.Load.NormalizedLoad1
		values["normalized_load_5"] = snapshot.Load.NormalizedLoad5
		values["run_queue_depth"] = snapshot.Load.RunningProcs
		values["total_processes"] = snapshot.Load.TotalProcs
	}
	for key, total := range map[string]uint64{"context_switches": ctxt, "interrupts": intr} {
		for k, v := range counterRateValues(c.rates, key, total, now) {
			values[k] = v
		}
	}
	for _, rate := range []string{"context_switches", "interrupts"} {
		status, _ := values[rate+"_rate_status"].(string)
		values[rate+"_per_second_status"] = status
		values[rate+"_per_second_provenance"] = "linux /proc/stat"
	}
	if !measured {
		return platformCPUReading{status: "not_yet_sampled", reason: "CPU counter delta requires a previous valid sample", provenance: "linux /proc/stat", values: values}
	}
	return platformCPUReading{usage: usage, status: "measured", provenance: "linux /proc/stat", values: values}
}

func parseCPUFields(fields []string) []uint64 {
	if len(fields) < 4 {
		return nil
	}
	out := make([]uint64, len(fields))
	for i, f := range fields {
		v, e := strconv.ParseUint(f, 10, 64)
		if e != nil {
			return nil
		}
		out[i] = v
	}
	return out
}

func loadStatus(s hostinventory.Snapshot) string {
	if s.ProbeStatuses == nil {
		return "not_yet_sampled"
	}
	if s.ProbeStatuses["load"] == "ok" {
		return "measured"
	}
	if s.ProbeStatuses["load"] == "unsupported" {
		return "unsupported"
	}
	if s.ProbeStatuses["load"] == "failed" {
		return "failed"
	}
	return "not_yet_sampled"
}

func loadProvenance(s hostinventory.Snapshot) string {
	if p, ok := s.FieldProvenance["load"]; ok {
		return p.Source
	}
	return "host inventory load probe"
}

func linuxQuotaEvidence(c *CPUCollector, now time.Time) (map[string]interface{}, string, string) {
	raw, err := os.ReadFile("/sys/fs/cgroup/cpu.stat")
	if err != nil {
		return nil, "unsupported", "cgroup v2 cpu.stat is unavailable"
	}
	stats := map[string]uint64{}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 {
			if value, parseErr := strconv.ParseUint(fields[1], 10, 64); parseErr == nil {
				stats[fields[0]] = value
			}
		}
	}
	throttled, ok := stats["nr_throttled"]
	if !ok {
		return nil, "failed", "cgroup cpu.stat has no nr_throttled counter"
	}
	values := map[string]interface{}{"quota_throttled_periods": float64(throttled)}
	for key, value := range counterRateValues(c.rates, "quota_throttled_periods", throttled, now) {
		values[key] = value
	}
	if throttledUsec, exists := stats["throttled_usec"]; exists {
		for key, value := range counterRateValues(c.rates, "quota_throttled_usec", throttledUsec, now) {
			values[key] = value
		}
	}
	if periods := stats["nr_periods"]; periods > 0 {
		share := float64(throttled) * 100 / float64(periods)
		values["quota_throttled_share"] = share
		values["quota_throttling"] = share
	}
	if throttledUsec, exists := stats["throttled_usec"]; exists {
		values["quota_throttled_seconds"] = float64(throttledUsec) / 1e6
	}
	max, readErr := os.ReadFile("/sys/fs/cgroup/cpu.max")
	if readErr != nil {
		return nil, "unsupported", "no cgroup v2 cpu.max; no cgroup CPU limit applies"
	}
	fields := strings.Fields(string(max))
	if len(fields) != 2 || fields[0] == "max" {
		return nil, "unsupported", "cgroup cpu.max reports no CPU limit"
	}
	quota, quotaErr := strconv.ParseFloat(fields[0], 64)
	period, periodErr := strconv.ParseFloat(fields[1], 64)
	if quotaErr != nil || periodErr != nil || quota <= 0 || period <= 0 {
		return nil, "failed", "cgroup cpu.max contains an invalid quota"
	}
	values["cpu_quota_cores"] = quota / period
	return values, "measured", "cgroup v2 cpu.stat"
}

func linuxFrequencyEvidence() (float64, string, string) {
	curRaw, curErr := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq")
	maxRaw, maxErr := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq")
	if curErr != nil || maxErr != nil {
		return 0, "unsupported", "cpufreq current or maximum frequency is unavailable"
	}
	cur, curErr := strconv.ParseFloat(strings.TrimSpace(string(curRaw)), 64)
	max, maxErr := strconv.ParseFloat(strings.TrimSpace(string(maxRaw)), 64)
	if curErr != nil || maxErr != nil || max <= 0 {
		return 0, "failed", "cpufreq frequency values are not numeric"
	}
	return cur / max, "measured", "sysfs cpufreq scaling_cur_freq and cpuinfo_max_freq"
}
