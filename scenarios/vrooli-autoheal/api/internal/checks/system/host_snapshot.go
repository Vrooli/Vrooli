package system

import (
	"context"

	sharedhost "github.com/vrooli/vrooli/internal/hostinventory"
)

type hostSnapshotCollector interface {
	Collect(ctx context.Context) (sharedhost.Snapshot, error)
}

type defaultHostSnapshotCollector struct{}

func (defaultHostSnapshotCollector) Collect(ctx context.Context) (sharedhost.Snapshot, error) {
	return sharedhost.Collect(ctx)
}
