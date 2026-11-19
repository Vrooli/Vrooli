//go:build unix

package sysmounts

import (
	"fmt"
	"os"
	"syscall"
)

// DeviceKey identifies the mounted filesystem backing path. Two paths on the
// same mounted filesystem share the key; two paths on different devices do
// not. It is a current proof used for redundancy decisions, not a durable
// catalog identity.
func DeviceKey(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ""
	}
	return fmt.Sprintf("stdev:%d", stat.Dev)
}
