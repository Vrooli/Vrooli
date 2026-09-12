//go:build !windows

package hostfs

import (
	"io/fs"
	"os"
	"syscall"
)

// currentUID is resolved once. os.Getuid is a cheap syscall, but the ownership
// check runs for every entry in a walk that can exceed twenty thousand of them,
// and the value cannot change during the process lifetime.
var currentUID = os.Getuid()

// ownedByCurrentUser reports whether the running user owns the entry.
//
// This is the guard that makes walking a world-writable /tmp safe. The sticky
// bit on /tmp already prevents removing another user's entries at the kernel
// level, so without this check the provider would happily list them, estimate
// their bytes as reclaimable, and then fail at apply time — reporting space it
// was never able to free.
//
// A file whose ownership cannot be determined is treated as foreign. Failing
// closed is the only defensible default for a check that gates deletion.
func ownedByCurrentUser(info fs.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return int(stat.Uid) == currentUID
}
