//go:build !windows

package resources

import "syscall"

// detachSysProcAttr starts the companion in its own session so it survives the
// short-lived control process that launched it.
func detachSysProcAttr() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }

// terminateCompanion asks the companion to exit (SIGTERM; it shuts down its
// listener gracefully).
func terminateCompanion(pid int) error {
	if pid <= 0 {
		return nil
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}
