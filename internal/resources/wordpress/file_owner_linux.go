//go:build linux

package wordpress

import (
	"fmt"
	"os"
	"syscall"
)

func directoryOwnerID(info os.FileInfo) (int, int, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("cannot read WordPress data directory ownership")
	}
	return int(stat.Uid), int(stat.Gid), nil
}
