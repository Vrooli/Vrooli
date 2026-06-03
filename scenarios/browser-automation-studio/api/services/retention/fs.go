package retention

import (
	"io/fs"
	"os"
	"path/filepath"
)

// OSFileSystem is the production FileSystem backed by the local disk.
type OSFileSystem struct{}

var _ FileSystem = OSFileSystem{}

// DirSize walks dir and sums the size of regular files. A missing directory
// returns (0, false, nil).
func (OSFileSystem) DirSize(dir string) (int64, bool, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if !info.IsDir() {
		// A non-directory at this path: report its size, treat as present.
		return info.Size(), true, nil
	}

	var total int64
	walkErr := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, statErr := d.Info()
		if statErr != nil {
			return statErr
		}
		total += fi.Size()
		return nil
	})
	if walkErr != nil {
		return total, true, walkErr
	}
	return total, true, nil
}

// RemoveAll removes dir and its contents. Removing a missing path is a no-op.
func (OSFileSystem) RemoveAll(dir string) error {
	return os.RemoveAll(dir)
}
