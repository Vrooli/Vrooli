//go:build linux

package process

import (
	"os/exec"
	"testing"
	"time"
)

func TestIsPIDRunningTreatsZombieProcessAsStopped(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = cmd.Wait() }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !IsPIDRunning(cmd.Process.Pid) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("process never became non-running before timeout")
}
