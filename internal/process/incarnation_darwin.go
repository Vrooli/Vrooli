//go:build darwin

package process

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func processIncarnation(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid process ID %d", pid)
	}
	proc, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return "", err
	}
	if int(proc.Proc.P_pid) != pid || proc.Proc.P_starttime.Sec == 0 {
		return "", fmt.Errorf("process identity unavailable for pid %d", pid)
	}
	return formatIncarnation(pid, int64(proc.Proc.P_starttime.Sec), int64(proc.Proc.P_starttime.Usec)), nil
}

// formatIncarnation is the stored incarnation format.
func formatIncarnation(pid int, sec, usec int64) string {
	return fmt.Sprintf("darwin:%d:%d:%d", pid, sec, usec)
}

func incarnationFormatSample() string { return formatIncarnation(1, 1, 1) }
