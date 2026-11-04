package hostinventory

import (
	"context"
	"maps"

	"github.com/vrooli/vrooli/internal/tuning"
)

// RefreshNvidiaGPUProcesses reuses stable inventory but observes current process
// occupancy. It does not write the shared hardware cache or run unrelated probes.
// NVIDIA is currently the collector that populates GPUProcesses; other backends
// retain their existing observation contract when the NVIDIA tool is absent.
func (c Collector) RefreshNvidiaGPUProcesses(ctx context.Context, snapshot Snapshot) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if !snapshot.HasVendorTool(ToolNvidiaSMI) {
		return snapshot, nil
	}
	c = c.withDefaults()
	snapshot.ProbeStatuses = maps.Clone(snapshot.ProbeStatuses)
	if snapshot.ProbeStatuses == nil {
		snapshot.ProbeStatuses = map[string]string{}
	}
	snapshot.FieldProvenance = maps.Clone(snapshot.FieldProvenance)
	if snapshot.FieldProvenance == nil {
		snapshot.FieldProvenance = map[string]Provenance{}
	}
	snapshot.Warnings = append([]string(nil), snapshot.Warnings...)
	probeCtx, cancel := context.WithTimeout(ctx, tuning.ServiceHealthTimeout())
	defer cancel()
	c.collectNvidiaGPUProcesses(probeCtx, &snapshot, c.Clock.Now())
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	// Probe failure/incomplete rows stay explicit in the snapshot; placement
	// reports unknown rather than turning missing data into zero occupancy.
	return snapshot, nil
}
