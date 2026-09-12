package retention

import (
	"os"
	"path/filepath"
)

// FreeSpaceFunc reports the bytes available on the filesystem holding a path.
// It is a seam so the free-space guard can be driven to its failure case in a
// test without filling a disk.
type FreeSpaceFunc func(path string) (int64, error)

// FreeSpace reports the bytes available on the filesystem holding path.
//
// When path does not exist yet — a database about to be created, a directory not
// yet populated — the nearest existing ancestor is measured instead, because the
// filesystem is the same one either way and failing here would disable the guard
// exactly when it is most needed.
func FreeSpace(path string) (int64, error) {
	dir := path
	for {
		if _, err := os.Stat(dir); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return freeSpace(dir)
}
