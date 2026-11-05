//go:build windows

package process

import (
	"os"
	"testing"
)

func TestIsPIDRunningOwnPID(t *testing.T) {
	if !IsPIDRunning(os.Getpid()) {
		t.Fatal("own PID must always be reported running")
	}
}
