//go:build unix

package runs

import (
	"fmt"
	"os"
	"syscall"
)

// runtimeDeviceKey returns a key that identifies the mounted filesystem backing
// path. Two paths on the same mounted filesystem share the key; two paths on
// different devices do not. It is a *current* proof used for the degraded-write
// redundancy check, not a durable destination identity.
//
// It exists because a legacy destination may have no persisted volume
// identity. Refusing to use a real second copy just because its catalog row
// predates identity binding would keep a single-destination deployment blocked
// forever.
func runtimeDeviceKey(path string) string {
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
