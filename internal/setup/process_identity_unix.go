//go:build !windows

package setup

import (
	"os"
	"syscall"
)

func processIdentityAlive(pid int, host string) bool {
	localHost, err := os.Hostname()
	if err != nil || host == "" || host != localHost || pid <= 0 {
		return false
	}
	err = syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
