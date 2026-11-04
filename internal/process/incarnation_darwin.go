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
	return fmt.Sprintf("darwin:%d:%d:%d", pid, proc.Proc.P_starttime.Sec, proc.Proc.P_starttime.Usec), nil
}
