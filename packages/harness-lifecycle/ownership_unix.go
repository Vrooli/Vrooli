//go:build !windows

package agentinstall

import (
	"os"
	"syscall"
)

func ownedByRoot(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}

// writableByCaller asks the kernel, so group membership and ACLs count. A mode
// comparison would misjudge both.
func writableByCaller(path string) bool {
	const writeAccess = 2
	return syscall.Access(path, writeAccess) == nil
}
