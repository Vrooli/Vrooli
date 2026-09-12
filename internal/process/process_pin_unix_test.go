//go:build unix

package process

import (
	"os"
	"testing"
)

// TestIsPIDRunningTreatsOtherUserProcessAsAlive pins the EPERM mapping: PID 1
// is always alive and, for an unprivileged test run, not signalable — kill(2)
// with signal 0 returns EPERM. Reporting it dead would feed false-dead
// process evidence into registry claim expiry (`vrooli cleanup locks`).
func TestIsPIDRunningTreatsOtherUserProcessAsAlive(t *testing.T) {
	if !IsPIDRunning(1) {
		t.Fatal("expected PID 1 (init) to be reported running; EPERM from kill(2) must map to alive")
	}
}

// TestIsPIDRunningNeverFalseForOwnPID is the cross-platform floor: whatever
// the GOOS-specific liveness primitive does, the current process must always
// read as running.
func TestIsPIDRunningNeverFalseForOwnPID(t *testing.T) {
	if !IsPIDRunning(os.Getpid()) {
		t.Fatal("own PID must always be reported running")
	}
}
