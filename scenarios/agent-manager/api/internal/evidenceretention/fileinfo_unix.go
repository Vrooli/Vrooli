//go:build unix

package evidenceretention

import (
	"io/fs"
	"syscall"
)

type fileKey struct{ dev, ino uint64 }

func keyOf(info fs.FileInfo) (fileKey, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileKey{}, false
	}
	return fileKey{dev: uint64(st.Dev), ino: uint64(st.Ino)}, true //nolint:unconvert // Dev width differs by platform
}
