//go:build linux

package setupprivilege

import (
	"os"
	"syscall"
)

func isRootOwnedDirectory(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}
