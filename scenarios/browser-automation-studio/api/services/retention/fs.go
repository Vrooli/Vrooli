package retention

import (
	"context"
	"os"

	coreRetention "github.com/vrooli/api-core/retention"
)

// OSFileSystem is the production FileSystem backed by the local disk.
type OSFileSystem struct{}

var _ FileSystem = OSFileSystem{}

// DirSize walks dir and sums the size of regular files. A missing directory
// returns (0, false, nil).
func (OSFileSystem) DirSize(dir string) (int64, bool, error) {
	return coreRetention.MeasureDirectory(context.Background(), dir)
}

func (OSFileSystem) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// RemoveAll removes dir and its contents. Removing a missing path is a no-op.
func (OSFileSystem) RemoveAll(dir string) error {
	return coreRetention.RemovePath(context.Background(), dir)
}

// DeleteContained routes domain-selected artifact deletion through the shared
// engine while preserving the retention service's database-driven selection.
func (OSFileSystem) DeleteContained(ctx context.Context, root, target string) error {
	return coreRetention.DeleteContained(ctx, root, target, nil)
}
