package collectors

import (
	"context"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"testing"
	"time"
)

func TestGPUIdentityNeverBecomesMeasuredTelemetry(t *testing.T) {
	for _, source := range []string{hostinventory.SourceNativeDevice, hostinventory.SourceDarwinUnifiedMemory, hostinventory.SourceSystemProfiler, hostinventory.SourceWMIC} {
		t.Run(source, func(t *testing.T) {
			snapshot := hostinventory.Snapshot{GPUs: []hostinventory.GPU{{Name: "observed or historical display", Source: source, VRAMBytes: 16 << 30}}}
			collector := &GPUCollector{BaseCollector: NewBaseCollector("gpu", time.Second), snapshots: &CachedSnapshotProvider{ttl: time.Second, now: time.Now, collect: func(context.Context) (hostinventory.Snapshot, error) { return snapshot, nil }}}
			data, err := collector.Collect(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if data.Values["status"] != "unsupported" {
				t.Fatalf("identity became telemetry: %#v", data.Values)
			}
			if _, ok := data.Values["total_usage_percent"]; ok {
				t.Fatalf("invented zero utilization: %#v", data.Values)
			}
		})
	}
}

func TestGPUIdentityDoesNotDiluteMeasuredTelemetry(t *testing.T) {
	snapshot := hostinventory.Snapshot{GPUs: []hostinventory.GPU{
		{Name: "native AMD", Source: hostinventory.SourceNativeDevice},
		{Name: "measured NVIDIA", Source: hostinventory.SourceNvidiaSMI, VRAMBytes: 8 << 30, UtilizationPercent: 60},
	}}
	devices, summary, _, _ := adaptGPUInventory(snapshot)
	if len(devices) != 1 || summary.DeviceCount != 1 || summary.AverageUtilizationPercent != 60 || summary.TotalMemoryMB != 8192 {
		t.Fatalf("mixed identity diluted measurement: %+v / %+v", devices, summary)
	}
}
