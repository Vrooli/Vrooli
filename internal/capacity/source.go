package capacity

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"

	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/internal/setpoint"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// CapacitySource is the injectable seam over live host capacity sensing
// (internal/hostinventory). Production reads the real snapshot; unit tests
// provide a fake so Decide/Reconcile never shell out to nvidia-smi.
type CapacitySource interface {
	Snapshot(ctx context.Context) (hostinventory.Snapshot, error)
}

// HostInventorySource is the production CapacitySource backed by
// hostinventory.Collect.
type HostInventorySource struct{}

// Snapshot collects a live host inventory snapshot.
func (HostInventorySource) Snapshot(ctx context.Context) (hostinventory.Snapshot, error) {
	return hostinventory.Collect(ctx)
}

// StaticSource is a CapacitySource returning a fixed snapshot (tests, and the
// CLI when a snapshot has already been fetched).
type StaticSource struct {
	Inventory hostinventory.Snapshot
	Err       error
}

// Snapshot returns the fixed snapshot.
func (s StaticSource) Snapshot(context.Context) (hostinventory.Snapshot, error) {
	return s.Inventory, s.Err
}

// ReadHostMemoryPressure reads fresh current memory-stall evidence. An unread
// sensor is not a measured zero and retains the legacy swap-occupancy fallback.
func ReadHostMemoryPressure() hostpressure.Reading {
	if runtime.GOOS != "linux" {
		return hostpressure.NewUnread("host-psi", "memory PSI is unavailable on this host")
	}
	const path = "/proc/pressure/memory"
	raw, err := os.ReadFile(path)
	if err != nil {
		return hostpressure.NewUnread("linux:"+path, err.Error())
	}
	value, ok := hostpressure.ParsePSISomeAvg10(string(raw))
	if !ok || !validPressurePercent(value) {
		return hostpressure.NewUnread("linux:"+path, "memory PSI some.avg10 is missing or invalid")
	}
	return hostpressure.NewRead(value, "linux:"+path)
}

// MemoryPressurePolicy applies the one canonical pressure classification before
// pure Decide. It is shared by the public broker, CLI claims and resource
// lifecycle admission. A nil reader leaves injected snapshots unchanged.
// A clear measured PSI suppresses only the legacy occupancy gate; RAM fit,
// ledger commitments and reclaim protections still run in Decide.
func MemoryPressurePolicy(policy Policy, reader func() hostpressure.Reading) (Policy, error) {
	if reader == nil {
		return policy, nil
	}
	cwd, _ := os.Getwd()
	bars, err := setpoint.Resolve(os.Environ(), cwd)
	if err != nil {
		return policy, fmt.Errorf("memory pressure threshold unavailable: %w", err)
	}
	bar, ok := bars.Bar(setpoint.CellMemoryPSI)
	if !ok || bar.Max == nil || !validPressurePercent(*bar.Max) {
		return policy, fmt.Errorf("memory pressure threshold unavailable: setpoint %s has no valid maximum", setpoint.CellMemoryPSI)
	}
	reading := reader()
	if reading.State != hostpressure.Read || !validPressurePercent(reading.Value) {
		return policy, nil
	}
	if reading.Value >= *bar.Max {
		return policy, fmt.Errorf("host memory PSI some.avg10 %.2f%% is at or above setpoint %s maximum %.2f%%", reading.Value, setpoint.CellMemoryPSI, *bar.Max)
	}
	policy.SwapPressureThresholdPct = 0
	return policy, nil
}

func validPressurePercent(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 100
}
