package services

import (
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/maputil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// Re-export maputil functions as package-level aliases for backward compatibility
// within the services package. These delegate to the shared maputil package.
var (
	getFloat64Value = maputil.GetFloat64
	getIntValue     = maputil.GetInt
	getInt64Value   = maputil.GetInt64
	getFloat64Slice = maputil.GetFloat64Slice
	getStringValue  = maputil.GetString
	getBoolValue    = maputil.GetBool
)

func metricStateNotYetSampled(reason string, observedAt time.Time) models.MetricState {
	return models.MetricState{Status: "not_yet_sampled", Reason: reason, ObservedAt: observedAt}
}

func notYetSampledState(reason, units string, observedAt time.Time) models.MetricState {
	state := metricStateNotYetSampled(reason, observedAt)
	state.Units = units
	return state
}

func metricState(data *collectors.MetricData, key, unavailableReason string) models.MetricState {
	now := time.Now()
	if data != nil && !data.Timestamp.IsZero() {
		now = data.Timestamp
	}
	state := models.MetricState{Status: "not_yet_sampled", Reason: unavailableReason, ObservedAt: now}
	state.Units = metricUnits(data, key)
	if data == nil {
		return state
	}
	if data.Tags != nil {
		state.Provenance = data.Tags["source"]
		state.CycleID = data.Tags["cycle_id"]
	}
	if state.Provenance == "" {
		state.Provenance = "system-monitor/" + data.CollectorName
	}
	// CPU and other multi-signal collectors carry independent states beside
	// each value. Prefer that envelope over the collector-wide status so an
	// unavailable signal cannot be mistaken for an unmeasured generic value.
	if status, ok := data.Values[key+"_status"].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := data.Values[key+"_reason"].(string); ok && reason != "" {
		state.Reason = reason
	}
	if provenance, ok := data.Values[key+"_provenance"].(string); ok && provenance != "" {
		state.Provenance = provenance
	}
	if status, ok := data.Values["status"].(string); ok && status != "" {
		if _, hasSignalStatus := data.Values[key+"_status"]; !hasSignalStatus {
			state.Status = status
		}
	}
	if reason, ok := data.Values["reason"].(string); ok && reason != "" {
		if _, hasSignalReason := data.Values[key+"_reason"]; !hasSignalReason {
			state.Reason = reason
		}
	}
	if errText, ok := data.Values["error"].(string); ok && errText != "" {
		state.Reason = errText
	}
	// An explicit unsupported/failed status is authoritative. Collectors may
	// retain a zero-valued compatibility field alongside the state, but that
	// field must never turn an unavailable measurement into a false reading.
	if state.Status == "unsupported" || state.Status == "failed" {
		if state.Reason == "" {
			state.Reason = fmt.Sprintf("collector did not measure %s", key)
		}
		return state
	}
	if value, ok := data.Values[key].(float64); ok {
		state.Status = "measured"
		state.Value = value
		state.Reason = ""
		return state
	}
	if value, ok := data.Values[key].(int); ok {
		state.Status = "measured"
		state.Value = float64(value)
		state.Reason = ""
		return state
	}
	if state.Status == "measured" {
		state.Status = "failed"
	}
	if state.Reason == "" {
		state.Reason = fmt.Sprintf("collector did not return %s", key)
	}
	return state
}

func metricUnits(data *collectors.MetricData, key string) string {
	if data != nil && data.Tags != nil && data.Tags["units"] != "" {
		return data.Tags["units"]
	}
	if key == "tcp_connections" {
		return "count"
	}
	switch key {
	case "context_switches_per_second", "interrupts_per_second", "fork_rate":
		return "per second"
	case "run_queue_depth":
		return "processes"
	case "normalized_load_1", "normalized_load_5", "load_average":
		return "load/core"
	case "quota_throttling":
		return "per second, percent"
	case "frequency_derate_ratio":
		return "ratio"
	case "thermal_throttle_evidence", "thermal_trip_point_celsius":
		return "celsius"
	}
	return "percent"
}

func diskMetricState(data *collectors.MetricData) models.MetricState {
	state := metricState(data, "usage", "disk collector did not return usage")
	if data == nil {
		return state
	}
	usage, ok := data.Values["usage"].(map[string]interface{})
	if !ok {
		return state
	}
	if status, _ := usage["status"].(string); status == "unsupported" || status == "failed" {
		state.Status = status
		if reason, _ := usage["reason"].(string); reason != "" {
			state.Reason = reason
		}
		return state
	}
	percent, ok := usage["percent"].(float64)
	if !ok {
		return models.MetricState{Status: "failed", Reason: "disk usage percent was not measured", Provenance: state.Provenance, ObservedAt: state.ObservedAt}
	}
	state.Status, state.Value, state.Reason = "measured", percent, ""
	return state
}
