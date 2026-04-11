//go:build !windows

package maintenance

import (
	"os"
	"syscall"
)

func killProcess(pid int, force bool) error {
	if pid <= 0 {
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if force {
		return process.Kill()
	}
	return process.Signal(syscall.SIGTERM)
}
