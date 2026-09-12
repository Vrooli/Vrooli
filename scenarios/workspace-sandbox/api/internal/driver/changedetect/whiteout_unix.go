//go:build !windows

package changedetect

import (
	"os"
	"syscall"
)

// isCharDevWhiteout reports whether the entry at path is an overlayfs
// whiteout (a character device with rdev=0, i.e. major=0/minor=0). The
// rdev check needs the platform stat struct, which is why this lives
// behind a build tag; it works identically on Linux and Darwin.
func isCharDevWhiteout(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return stat.Rdev == 0
}
