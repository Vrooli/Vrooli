//go:build windows

package process

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func processIsAlive(process *os.Process) bool {
	if process == nil {
		return false
	}
	return pidIsAliveWindows(process.Pid)
}

func pidIsAliveWindows(pid int) bool {
	if pid <= 0 {
		return false
	}
	// SYNCHRONIZE is required for WaitForSingleObject below; without it the
	// wait fails with access denied on every handle and all live processes
	// would read as dead — the false-dead failure mode this primitive exists
	// to avoid.
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		// ACCESS_DENIED means the process exists but belongs to another
		// user/session — alive (the windows mirror of EPERM on unix).
		// INVALID_PARAMETER means no such PID.
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	defer windows.CloseHandle(handle)
	// WaitForSingleObject with a zero timeout: WAIT_TIMEOUT means the process
	// has not signaled (still running); WAIT_OBJECT_0 means it has exited.
	// GetExitCodeProcess is deliberately avoided: a process that exited with
	// code 259 (STILL_ACTIVE) would be indistinguishable from a live one.
	event, err := windows.WaitForSingleObject(handle, 0)
	if err != nil {
		// We could open the process but could not query it. Fail safe: an
		// unverifiable process must read as alive, never dead (false-dead
		// evidence expires valid registry claims downstream).
		return true
	}
	return event == uint32(windows.WAIT_TIMEOUT)
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	return nil, fmt.Errorf("process environment inspection is not supported on this platform")
}
