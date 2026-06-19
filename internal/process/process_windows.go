//go:build windows

package process

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

// Syscall seams so the open/wait decision mapping below can be pinned by
// tests without real PIDs (see process_windows_test.go).
var (
	openProcessFn = func(pid uint32) (windows.Handle, error) {
		// SYNCHRONIZE is required for WaitForSingleObject below; without it
		// the wait fails with access denied on every handle and all live
		// processes would read as dead — the false-dead failure mode this
		// primitive exists to avoid.
		return windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, pid)
	}
	waitForSingleObjectFn = func(handle windows.Handle) (uint32, error) {
		return windows.WaitForSingleObject(handle, 0)
	}
	closeHandleFn = windows.CloseHandle
)

func pidIsAlive(pid int) bool {
	handle, err := openProcessFn(uint32(pid))
	if err != nil {
		// ACCESS_DENIED means the process exists but belongs to another
		// user/session — alive (the windows mirror of EPERM on unix).
		// INVALID_PARAMETER means no such PID.
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	defer closeHandleFn(handle)
	// WaitForSingleObject with a zero timeout: WAIT_TIMEOUT means the process
	// has not signaled (still running); WAIT_OBJECT_0 means it has exited.
	// GetExitCodeProcess is deliberately avoided: a process that exited with
	// code 259 (STILL_ACTIVE) would be indistinguishable from a live one.
	event, err := waitForSingleObjectFn(handle)
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
