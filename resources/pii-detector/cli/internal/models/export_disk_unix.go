//go:build unix

package models

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

type exportFileIdentity struct{ dev, ino uint64 }

// allocatedDirectoryBytes measures allocated filesystem blocks at an owner
// path. It never follows symlinks, counts hard links once and does not cross
// onto another filesystem. The capacity job samples are therefore physical
// allocation observations, not logical model/tag byte totals.
func allocatedDirectoryBytes(root string) (int64, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return 0, errors.New("physical allocation path must be a real directory, not a symlink")
	}
	rootStat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok || rootStat.Blocks < 0 {
		return 0, errors.New("filesystem does not expose allocated directory blocks")
	}
	var total int64
	seen := map[exportFileIdentity]struct{}{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) && path != root {
				return nil
			}
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path != root {
				return nil
			}
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Blocks < 0 {
			return errors.New("filesystem does not expose allocated file blocks")
		}
		if uint64(stat.Dev) != uint64(rootStat.Dev) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode().IsRegular() {
			key := exportFileIdentity{dev: uint64(stat.Dev), ino: stat.Ino}
			if _, duplicate := seen[key]; duplicate {
				return nil
			}
			seen[key] = struct{}{}
		}
		blocks := int64(stat.Blocks) * 512
		if blocks < 0 || total > int64(^uint64(0)>>1)-blocks {
			return errors.New("allocated directory byte count overflow")
		}
		total += blocks
		return nil
	})
	return total, err
}
