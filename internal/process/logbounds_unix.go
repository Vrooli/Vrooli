//go:build !windows

package process

import (
	"io/fs"
	"syscall"
)

// allocatedBytes is the disk space a file occupies, which is smaller than its
// apparent size when the file is sparse.
func allocatedBytes(info fs.FileInfo) int64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return int64(stat.Blocks) * 512
	}
	return info.Size()
}
