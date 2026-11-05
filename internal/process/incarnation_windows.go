//go:build windows

package process

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func processIncarnation(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid process ID %d", pid)
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &created, &exited, &kernel, &user); err != nil {
		return "", err
	}
	return fmt.Sprintf("windows:%d:%d", pid, created.Nanoseconds()), nil
}
