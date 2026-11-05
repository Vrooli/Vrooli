package repository

import (
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// KernelMemoryFromValues is shared by live, SQLite and in-memory projections.
// Old rows have no kernel fields: retain that absence instead of inventing zero.
func KernelMemoryFromValues(values map[string]interface{}, cycleID string, observedAt time.Time) *models.KernelMemoryMetrics {
	switch sourceTime := values["kernel_observed_at"].(type) {
	case time.Time:
		if !sourceTime.IsZero() {
			observedAt = sourceTime
		}
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, sourceTime); err == nil {
			observedAt = parsed
		}
	}
	state := func(key string) models.MetricState {
		result := models.MetricState{Status: "not_yet_sampled", Reason: "kernel memory accounting was not collected in this sample", Provenance: "linux /proc/meminfo", Units: "bytes", CycleID: cycleID, ObservedAt: observedAt}
		if status, ok := values[key+"_status"].(string); ok && status != "" {
			result.Status, result.Reason = status, ""
		}
		if reason, ok := values[key+"_reason"].(string); ok {
			result.Reason = reason
		}
		if result.Status == "measured" {
			if value, ok := values[key].(float64); ok {
				result.Value = value
			} else {
				result.Status, result.Reason = "failed", "measured kernel memory field has no numeric value"
			}
		}
		return result
	}
	return &models.KernelMemoryMetrics{
		SlabBytes: state("kernel_slab_bytes"), ReclaimableBytes: state("kernel_reclaimable_bytes"),
		UnreclaimableBytes: state("kernel_unreclaimable_bytes"), HostTotalBytes: state("kernel_host_total_bytes"),
	}
}
