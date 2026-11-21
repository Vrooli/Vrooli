//go:build !windows

package setup

import (
	"os"

	"github.com/vrooli/vrooli/internal/process"
)

func processIdentityAlive(pid int, host string) bool {
	localHost, err := os.Hostname()
	if err != nil || host == "" || host != localHost || pid <= 0 {
		return false
	}
	return process.IsPIDRunning(pid)
}
