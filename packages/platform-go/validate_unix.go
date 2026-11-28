//go:build unix

package platform

import (
	"os"
	"syscall"
)

// ownRuntimeDir reports whether dir is an existing directory owned by the
// calling user, the only runtime directory a user-scope systemd manager accepts.
func ownRuntimeDir(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Geteuid()
}
