//go:build windows

package cliutil

import "os"

// Windows does not expose POSIX signal 0. FindProcess is the portable process
// existence seam; the lifecycle owner PID check remains authoritative when the
// peer record is read on Windows.
func isPIDRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	_, err := os.FindProcess(pid)
	return err == nil
}
