//go:build !linux

package programs

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestTerminateKernelProcessKillsDirectChild(t *testing.T) {
	const childEnv = "PROGRAM_RUNTIME_TERMINATION_TEST_CHILD"
	if os.Getenv(childEnv) == "1" {
		time.Sleep(time.Hour)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestTerminateKernelProcessKillsDirectChild$")
	cmd.Env = append(os.Environ(), childEnv+"=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	terminateKernelProcess(cmd.Process.Pid)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("child exited successfully; expected it to be killed")
		}
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("Wait() error = %T %v, want killed child exit", err, err)
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("direct child remained alive after termination")
	}
}
