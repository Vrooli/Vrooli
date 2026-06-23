//go:build windows

package resources

import (
	"os"
	"syscall"
)

// detachSysProcAttr is a no-op on Windows (no Setsid); the process is launched
// normally and released so it outlives the control process.
func detachSysProcAttr() *syscall.SysProcAttr { return nil }

// terminateCompanion stops the companion by killing the tracked process.
func terminateCompanion(pid int) error {
	if pid <= 0 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
