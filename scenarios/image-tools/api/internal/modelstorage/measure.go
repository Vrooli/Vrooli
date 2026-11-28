package modelstorage

import (
	"io/fs"
	"os"
	"path/filepath"
)

// MeasureAllocated returns the bytes a model directory occupies on its
// filesystem: allocated blocks of regular files, each inode counted once,
// without following symlinks. A missing path measures zero.
func MeasureAllocated(root string) (int64, error) {
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return allocatedBytes(info), nil
	}
	seen := map[[2]uint64]bool{}
	var total int64
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.Type().IsRegular() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		if key, ok := inodeKey(fi); ok {
			if seen[key] {
				return nil
			}
			seen[key] = true
		}
		total += allocatedBytes(fi)
		return nil
	})
	return total, err
}
