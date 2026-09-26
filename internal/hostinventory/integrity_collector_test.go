package hostinventory

import (
	"context"
	"testing"
	"time"
)

type fixedIntegrityCollector struct {
	value HostInventory
}

func (f fixedIntegrityCollector) Collect(context.Context) (HostInventory, error) { return f.value, nil }

func TestIntegrityFingerprintIgnoresObservationMetadata(t *testing.T) {
	base := HostInventory{CollectedAt: time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC), Platform: "linux", OS: "linux", Arch: "amd64", BootID: "boot-1", Kernel: KernelInfo{Release: "6.1", LoadedModules: []string{"zeta", "alpha"}}}
	changedTime := base
	changedTime.CollectedAt = base.CollectedAt.Add(time.Minute)
	if Fingerprint(base) != Fingerprint(changedTime) {
		t.Fatal("observation time changed the host fingerprint")
	}
}

func TestParseIntegrityPCIExtractsDriverAndModules(t *testing.T) {
	devices := parseIntegrityPCI("01:00.0 VGA compatible controller: Example GPU [1234:5678]\n\tKernel driver in use: example\n\tKernel modules: example, fallback\n")
	if len(devices) != 1 || devices[0].BoundDriver != "example" || len(devices[0].AvailableModules) != 2 {
		t.Fatalf("devices = %#v", devices)
	}
}

func TestCachedIntegrityCollectorReusesFreshInventory(t *testing.T) {
	collector := NewCachedIntegrityCollector(fixedIntegrityCollector{value: HostInventory{CollectedAt: time.Now().UTC()}}, time.Hour)
	first, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !first.CollectedAt.Equal(second.CollectedAt) {
		t.Fatalf("cache returned different observations: %v and %v", first.CollectedAt, second.CollectedAt)
	}
}
