//go:build unix

package modelstorage

import (
	"os"
	"syscall"
)

func allocatedBytes(fi os.FileInfo) int64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return int64(st.Blocks) * 512 //nolint:unconvert // Blocks width differs by platform
	}
	return fi.Size()
}

func inodeKey(fi os.FileInfo) ([2]uint64, bool) {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return [2]uint64{uint64(st.Dev), uint64(st.Ino)}, true //nolint:unconvert // Dev width differs by platform
	}
	return [2]uint64{}, false
}
