//go:build windows

package process

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

// stubWindowsProbe replaces the OpenProcess/WaitForSingleObject seams for one
// test and restores them on cleanup.
func stubWindowsProbe(t *testing.T, openErr error, waitEvent uint32, waitErr error) {
	t.Helper()
	prevOpen, prevWait, prevClose := openProcessFn, waitForSingleObjectFn, closeHandleFn
	t.Cleanup(func() {
		openProcessFn, waitForSingleObjectFn, closeHandleFn = prevOpen, prevWait, prevClose
	})
	openProcessFn = func(pid uint32) (windows.Handle, error) {
		if openErr != nil {
			return 0, openErr
		}
		return windows.Handle(1), nil
	}
	waitForSingleObjectFn = func(handle windows.Handle) (uint32, error) {
		return waitEvent, waitErr
	}
	closeHandleFn = func(handle windows.Handle) error { return nil }
}

// TestIsPIDRunningWindowsAccessDeniedMeansAlive pins the windows mirror of the
// unix EPERM mapping: a PID we cannot open because it belongs to another
// user/session exists, so it must read as alive. False-dead evidence expires
// valid registry claims downstream (`vrooli cleanup locks`).
func TestIsPIDRunningWindowsAccessDeniedMeansAlive(t *testing.T) {
	stubWindowsProbe(t, windows.ERROR_ACCESS_DENIED, 0, nil)
	if !IsPIDRunning(4242) {
		t.Fatal("ERROR_ACCESS_DENIED from OpenProcess must map to alive")
	}
}

// TestIsPIDRunningWindowsInvalidParameterMeansDead pins the no-such-PID case.
func TestIsPIDRunningWindowsInvalidParameterMeansDead(t *testing.T) {
	stubWindowsProbe(t, windows.ERROR_INVALID_PARAMETER, 0, nil)
	if IsPIDRunning(4242) {
		t.Fatal("ERROR_INVALID_PARAMETER from OpenProcess means no such PID — must map to dead")
	}
}

// TestIsPIDRunningWindowsWaitTimeoutMeansAlive pins the live-process case:
// WaitForSingleObject(handle, 0) returns WAIT_TIMEOUT while the process has
// not exited.
func TestIsPIDRunningWindowsWaitTimeoutMeansAlive(t *testing.T) {
	stubWindowsProbe(t, nil, uint32(windows.WAIT_TIMEOUT), nil)
	if !IsPIDRunning(4242) {
		t.Fatal("WAIT_TIMEOUT must map to alive")
	}
}

// TestIsPIDRunningWindowsSignaledMeansDead pins the exited-process case:
// WAIT_OBJECT_0 means the process handle is signaled (the process exited),
// even though a handle to it could still be opened.
func TestIsPIDRunningWindowsSignaledMeansDead(t *testing.T) {
	stubWindowsProbe(t, nil, uint32(windows.WAIT_OBJECT_0), nil)
	if IsPIDRunning(4242) {
		t.Fatal("WAIT_OBJECT_0 (process signaled/exited) must map to dead")
	}
}

// TestIsPIDRunningWindowsWaitErrorFailsTowardAlive pins the fail direction for
// an openable but unqueryable process: unverifiable must read alive, never
// dead.
func TestIsPIDRunningWindowsWaitErrorFailsTowardAlive(t *testing.T) {
	stubWindowsProbe(t, nil, 0, windows.ERROR_ACCESS_DENIED)
	if !IsPIDRunning(4242) {
		t.Fatal("a wait failure on an openable process must fail toward alive")
	}
}

// TestIsPIDRunningWindowsOwnPID exercises the real syscall path end to end on
// a windows host: the current process must always read as running.
func TestIsPIDRunningWindowsOwnPID(t *testing.T) {
	if !IsPIDRunning(os.Getpid()) {
		t.Fatal("own PID must always be reported running")
	}
}
