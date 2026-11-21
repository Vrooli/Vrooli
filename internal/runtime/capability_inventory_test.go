package runtime

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

func TestCapabilityFactsExcludeLegacyUnifiedMemoryVRAM(t *testing.T) {
	for _, source := range []string{hostinventory.SourceDarwinUnifiedMemory, hostinventory.SourceNvidiaSMI, hostinventory.SourceROCmSMI, hostinventory.SourceSystemProfiler} {
		t.Run(source, func(t *testing.T) {
			snap := hostinventory.Snapshot{GPUs: []hostinventory.GPU{{Name: "GPU", Source: source, VRAMBytes: 16 << 30}}}
			got := capabilityFactsFromSnapshot(snap)
			want := float64(16)
			if source == hostinventory.SourceDarwinUnifiedMemory {
				want = 0
			}
			if got.HasGPU != (source != hostinventory.SourceDarwinUnifiedMemory) || got.MaxVRAMGb != want {
				t.Fatalf("capability facts = %+v, want GPU presence and %v GiB dedicated VRAM", got, want)
			}
		})
	}
}

type unavailableCapabilityFiles struct{}

func (unavailableCapabilityFiles) ReadFile(string) ([]byte, error) { return nil, os.ErrNotExist }

func TestNativeCollectorCapabilityFactsPreservePresenceWithoutInventingVRAM(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			collector := hostinventory.Collector{
				GOOS: platform, GOARCH: "arm64", Commands: &shelltest.Fake{}, Files: unavailableCapabilityFiles{},
				NativeDevices: func(context.Context, hostinventory.CommandRunner) hostinventory.NativeDeviceObservation {
					return hostinventory.NativeDeviceObservation{Mechanism: "fixture-native", ProbeStatus: map[string]hostinventory.IntegrityProbeState{"devices": hostinventory.IntegrityProbeOK}, Devices: []hostinventory.DeviceInfo{{ID: "native:gpu", BusType: "soc", Class: "display controller", DeviceName: "Native GPU"}}}
				},
			}
			snap, err := collector.Collect(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			got := capabilityFactsFromSnapshot(snap)
			if !got.HasGPU || got.MaxVRAMGb != 0 {
				t.Fatalf("native collector capability = %+v; inventory = %+v", got, snap.GPUs)
			}
			verdict := capacity.Decide(capacity.CapacityRequest{ResourceKind: capacity.ResourceKindVRAM, PreferredBytes: 1 << 30, FloorBytes: 1 << 30}, snap, nil, capacity.DefaultPolicy(), time.Now())
			if verdict.Kind != capacity.VerdictDeny || verdict.GrantedBytes != 0 {
				t.Fatalf("identity-only GPU funded capacity: %+v", verdict)
			}
		})
	}
}
