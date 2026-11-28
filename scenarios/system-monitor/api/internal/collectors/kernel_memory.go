package collectors

import "github.com/vrooli/vrooli/internal/hostinventory"

// kernelMemoryValues reuses the shared host snapshot; it does not add a proc
// walk or privileged slab-cache read. Fields keep individual availability.
func kernelMemoryValues(snapshot hostinventory.Snapshot, snapshotErr error, platform string) map[string]interface{} {
	values := map[string]interface{}{}
	if observedAt := snapshot.FieldProvenance["memory.total_bytes"].ObservedAt; !observedAt.IsZero() {
		// The snapshot provider can serve cached facts. Preserve their original
		// time instead of claiming the underlying meminfo was just read.
		values["kernel_observed_at"] = observedAt
	}
	fields := []struct {
		key, source string
		value       *uint64
	}{
		{"kernel_slab_bytes", "Slab", snapshot.Memory.SlabBytes},
		{"kernel_reclaimable_bytes", "SReclaimable", snapshot.Memory.SlabReclaimableBytes},
		{"kernel_unreclaimable_bytes", "SUnreclaim", snapshot.Memory.SlabUnreclaimableBytes},
		{"kernel_host_total_bytes", "MemTotal", &snapshot.Memory.TotalBytes},
	}
	for _, field := range fields {
		status, reason := "measured", ""
		switch {
		case platform != "linux":
			status, reason = "unsupported", "Linux slab accounting unavailable on "+platform
		case snapshotErr != nil:
			status, reason = "failed", "host memory snapshot: "+snapshotErr.Error()
		case snapshot.Memory.TotalBytes == 0:
			status, reason = "failed", "Linux memory snapshot was not measured"
		case field.value == nil:
			status, reason = "unsupported", field.source+" is absent from /proc/meminfo"
		}
		values[field.key+"_status"] = status
		if reason != "" {
			values[field.key+"_reason"] = reason
		}
		if status == "measured" {
			values[field.key] = float64(*field.value)
		}
	}
	return values
}
