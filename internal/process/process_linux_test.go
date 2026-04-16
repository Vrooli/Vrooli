//go:build linux

package process

import (
	"os/exec"
	"testing"
	"time"
)

func TestParseProcessStateParsesProcStatLines(t *testing.T) {
	state, ok := parseProcessState([]byte("12345 (test process) Z 1 2 3 4\n"))
	if !ok {
		t.Fatal("expected stat line to parse")
	}
	if state != 'Z' {
		t.Fatalf("state = %q, want Z", state)
	}
}

func TestIsPIDRunningTreatsZombieProcessAsStopped(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		_ = cmd.Wait()
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state, ok := readProcessState(cmd.Process.Pid)
		if ok && state == 'Z' {
			if IsPIDRunning(cmd.Process.Pid) {
				t.Fatal("expected zombie process to be treated as stopped")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("process never reached zombie state before timeout")
}
