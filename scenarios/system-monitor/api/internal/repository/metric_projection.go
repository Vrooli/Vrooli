package repository

import (
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// HydrateMetricObservation projects one collector payload into the stable API
// response. Both storage engines use this so persistence format cannot change
// metric semantics.
func HydrateMetricObservation(response *models.MetricsResponse, cycleID string, observedAt time.Time, collector string, values map[string]interface{}) {
	state := storedMetricState(cycleID, observedAt, collector, values)
	switch collector {
	case "cpu":
		response.CPUState = state
		if state.Status == "measured" {
			response.CPUUsage = numericValue(values["usage_percent"])
		}
		response.CPUContextSwitchesPerSecond = signalMetricState(cycleID, observedAt, collector, values, "context_switches_per_second")
		response.CPUInterruptsPerSecond = signalMetricState(cycleID, observedAt, collector, values, "interrupts_per_second")
		response.CPUNormalizedLoad1 = signalMetricState(cycleID, observedAt, collector, values, "normalized_load_1")
		response.CPUNormalizedLoad5 = signalMetricState(cycleID, observedAt, collector, values, "normalized_load_5")
		response.CPURunQueueDepth = signalMetricState(cycleID, observedAt, collector, values, "run_queue_depth")
		response.CPUCoreImbalanceIndex = signalMetricState(cycleID, observedAt, collector, values, "core_imbalance_index")
		response.CPUModeIowait = cpuModeMetricState(cycleID, observedAt, values, "iowait")
		response.CPUModeSteal = cpuModeMetricState(cycleID, observedAt, values, "steal")
	case "memory":
		response.KernelMemory = KernelMemoryFromValues(values, cycleID, observedAt)
		response.MemoryState = state
		if state.Status == "measured" {
			response.MemoryUsage = numericValue(values["usage_percent"])
		}
		if state.Status == "measured" {
			if swap, ok := values["swap"].(map[string]interface{}); ok {
				if percent, measured := numericLookup(swap, "percent"); measured {
					response.SwapUsage = &percent
					swapState := state
					swapState.Value = percent
					swapState.Provenance = "system-monitor/memory.swap"
					swapState.Units = "percent"
					response.SwapState = swapState
				}
			}
		}
	case "network":
		response.ConnectionsState = state
		response.NetworkEstablishedRate = signalMetricState(cycleID, observedAt, collector, values, "established_rate_per_second")
		response.NetworkTimeWaitRate = signalMetricState(cycleID, observedAt, collector, values, "time_wait_rate_per_second")
		response.NetworkCloseWaitRate = signalMetricState(cycleID, observedAt, collector, values, "close_wait_rate_per_second")
		if state.Status == "measured" {
			response.TCPConnections = int(numericValue(values["tcp_connections"]))
		}
	case "gpu":
		response.GPUState = state
		response.GPUProcessesState = gpuProcessesMetricState(cycleID, observedAt, values)
		if usage, measured := numericLookup(values, "total_usage_percent"); measured && state.Status == "measured" {
			response.GPUUsage = &usage
		}
	case "disk":
		response.DiskState = state
		if usage, ok := values["usage"].(map[string]interface{}); ok && state.Status == "measured" {
			response.DiskUsage = numericValue(usage["percent"])
		}
	case "disk_io":
		response.DiskReadRateState = diskRateMetricState(cycleID, observedAt, values, "read_rate")
		response.DiskWriteRateState = diskRateMetricState(cycleID, observedAt, values, "write_rate")
	case "pressure":
		response.CPUStallSomeAvg10 = pressureMetricState(cycleID, observedAt, collector, values, "cpu_psi_some_avg10", "cpu_psi_status", "cpu_psi_reason")
		response.CPUStallFullAvg10 = pressureMetricState(cycleID, observedAt, collector, values, "cpu_psi_full_avg10", "cpu_psi_status", "cpu_psi_reason")
		response.SwapTrafficState = pressureMetricState(cycleID, observedAt, collector, values, "swap_traffic_pages_per_second", "swap_traffic_rate_status", "swap_traffic_rate_reason")
		response.MajorFaultsState = pressureMetricState(cycleID, observedAt, collector, values, "pgmajfault_per_second", "pgmajfault_rate_status", "pgmajfault_rate_reason")
		response.FragmentationIndexState = pressureMetricState(cycleID, observedAt, collector, values, "fragmentation_max_free_order", "fragmentation_status", "fragmentation_reason")
	}
}

func gpuProcessesMetricState(cycleID string, cycleTime time.Time, values map[string]interface{}) models.MetricState {
	state := models.MetricState{Status: "unsupported", Reason: "GPU process inventory was not reported", Provenance: "system-monitor/gpu", Units: "processes", CycleID: cycleID, ObservedAt: cycleTime}
	if status, ok := values["processes_status"].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := values["processes_reason"].(string); ok && reason != "" {
		state.Reason = reason
	}
	if provenance, ok := values["processes_provenance"].(string); ok && provenance != "" {
		state.Provenance = provenance
	}
	if raw, ok := values["processes_observed_at"].(string); ok && raw != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			state.ObservedAt = parsed
		}
	}
	if state.Status == "ok" {
		state.Status, state.Reason = "measured", ""
	} else if state.Reason == "" {
		state.Reason = "GPU process inventory probe status: " + state.Status
	}
	return state
}

func diskRateMetricState(cycleID string, cycleTime time.Time, values map[string]interface{}, key string) models.MetricState {
	state := models.MetricState{
		Status:     "not_yet_sampled",
		Reason:     "disk throughput has not been sampled",
		Provenance: "system-monitor/disk_io",
		Units:      "MB/s",
		CycleID:    cycleID,
		ObservedAt: cycleTime,
	}
	if status, ok := values[key+"_status"].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := values[key+"_reason"].(string); ok && reason != "" {
		state.Reason = reason
	}
	if source, ok := values[key+"_provenance"].(string); ok && source != "" {
		state.Provenance = source
	}
	if units, ok := values[key+"_units"].(string); ok && units != "" {
		state.Units = units
	}
	if rawTime, ok := values[key+"_observed_at"].(string); ok && rawTime != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, rawTime); err == nil {
			state.ObservedAt = parsed
		}
	}
	if state.Status == "measured" {
		if value, measured := numericLookup(values, key); measured {
			state.Value, state.Reason = value, ""
		} else {
			state.Status = "failed"
			state.Reason = "disk throughput was marked measured without a value"
		}
	}
	return state
}

func storedMetricState(cycleID string, observedAt time.Time, collector string, values map[string]interface{}) models.MetricState {
	state := models.MetricState{
		Status:     "failed",
		Reason:     "collector did not return a measurement",
		Provenance: "system-monitor/" + collector,
		Units:      metricUnits(collector),
		CycleID:    cycleID,
		ObservedAt: observedAt,
	}
	if status, ok := values["status"].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := values["reason"].(string); ok && reason != "" {
		state.Reason = reason
	}
	if source, ok := values["source"].(string); ok && source != "" {
		state.Provenance = source
	}
	if _, explicitStatus := values["status"].(string); !explicitStatus && hasStoredMeasurement(collector, values) {
		state.Status = "measured"
	}
	if state.Status == "" {
		state.Status = "failed"
	}
	if state.Status == "measured" {
		state.Reason = ""
		state.Value = storedMeasurementValue(collector, values)
	}
	return state
}

func hasStoredMeasurement(collector string, values map[string]interface{}) bool {
	switch collector {
	case "cpu", "memory":
		_, ok := numericLookup(values, "usage_percent")
		return ok
	case "network":
		_, ok := numericLookup(values, "tcp_connections")
		return ok
	case "gpu":
		_, ok := numericLookup(values, "total_usage_percent")
		return ok
	case "disk":
		usage, ok := values["usage"].(map[string]interface{})
		if !ok {
			return false
		}
		_, measured := numericLookup(usage, "percent")
		return measured
	default:
		return false
	}
}

func storedMeasurementValue(collector string, values map[string]interface{}) float64 {
	switch collector {
	case "cpu", "memory":
		return numericValue(values["usage_percent"])
	case "network":
		return numericValue(values["tcp_connections"])
	case "gpu":
		return numericValue(values["total_usage_percent"])
	case "disk":
		usage, _ := values["usage"].(map[string]interface{})
		return numericValue(usage["percent"])
	default:
		return 0
	}
}

func signalMetricState(cycleID string, observedAt time.Time, collector string, values map[string]interface{}, key string) models.MetricState {
	state := models.MetricState{
		Status:     "not_yet_sampled",
		Reason:     "rate has not been sampled",
		Provenance: "system-monitor/" + collector,
		CycleID:    cycleID,
		ObservedAt: observedAt,
	}
	if status, ok := values[key+"_status"].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := values[key+"_reason"].(string); ok && reason != "" {
		state.Reason = reason
	}
	if value, measured := numericLookup(values, key); measured {
		state.Status, state.Value, state.Reason = "measured", value, ""
	}
	return state
}

func cpuModeMetricState(cycleID string, observedAt time.Time, values map[string]interface{}, mode string) models.MetricState {
	state := models.MetricState{
		Status:     "not_yet_sampled",
		Reason:     "CPU mode breakdown has not been sampled",
		Provenance: "system-monitor/cpu",
		Units:      "percent",
		CycleID:    cycleID,
		ObservedAt: observedAt,
	}
	switch breakdown := values["mode_breakdown"].(type) {
	case map[string]interface{}:
		if value, ok := numericLookup(breakdown, mode); ok {
			state.Status, state.Value, state.Reason = "measured", value, ""
		}
	case map[string]float64:
		if value, ok := breakdown[mode]; ok {
			state.Status, state.Value, state.Reason = "measured", value, ""
		}
	}
	return state
}

func pressureMetricState(cycleID string, observedAt time.Time, collector string, values map[string]interface{}, valueKey, statusKey, reasonKey string) models.MetricState {
	if _, hasSignalStatus := values[valueKey+"_status"]; hasSignalStatus {
		statusKey, reasonKey = valueKey+"_status", valueKey+"_reason"
	}
	state := models.MetricState{
		Status:     "not_yet_sampled",
		Provenance: "system-monitor/" + collector,
		CycleID:    cycleID,
		ObservedAt: observedAt,
	}
	if status, ok := values[statusKey].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := values[reasonKey].(string); ok {
		state.Reason = reason
	}
	if value, measured := numericLookup(values, valueKey); measured {
		state.Status, state.Value, state.Reason = "measured", value, ""
	}
	if state.Status == "unsupported" && state.Reason == "" {
		state.Reason = "metric unsupported on this platform"
	}
	if state.Status == "not_yet_sampled" && state.Reason == "" {
		state.Reason = "rate has not been sampled"
	}
	return state
}

func metricUnits(collector string) string {
	if collector == "network" {
		return "count"
	}
	return "percent"
}

func numericLookup(values map[string]interface{}, key string) (float64, bool) {
	value, ok := values[key]
	if !ok {
		return 0, false
	}
	return numeric(value)
}

func numeric(value interface{}) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint:
		return float64(number), true
	case uint32:
		return float64(number), true
	case uint64:
		return float64(number), true
	default:
		return 0, false
	}
}

func numericValue(value interface{}) float64 {
	number, _ := numeric(value)
	return number
}
