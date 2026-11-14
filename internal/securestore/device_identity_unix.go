//go:build !windows

package securestore

import (
	"fmt"
	"os"
	"syscall"
)

func pathDeviceIdentity(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("dev:%d", stat.Dev), true
}
