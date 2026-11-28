//go:build !windows

package providers

import (
	"fmt"
	"strconv"
	"syscall"
)

// MeasureFilesystemSpace reports the space available to unprivileged writers
// on the filesystem holding path, keyed by device so stores on one
// filesystem share one band.
func MeasureFilesystemSpace(path string) (FilesystemSpace, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return FilesystemSpace{}, fmt.Errorf("statfs %s: %w", path, err)
	}
	var info syscall.Stat_t
	if err := syscall.Stat(path, &info); err != nil {
		return FilesystemSpace{}, fmt.Errorf("stat %s: %w", path, err)
	}
	return FilesystemSpace{
		ID:         "dev:" + strconv.FormatUint(uint64(info.Dev), 10),
		Path:       path,
		FreeBytes:  int64(uint64(stat.Bavail) * uint64(stat.Bsize)),
		TotalBytes: int64(uint64(stat.Blocks) * uint64(stat.Bsize)),
	}, nil
}
