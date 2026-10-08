package capacity

import (
	"context"

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
