//go:build linux

package process

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestProcessHandleHelper(t *testing.T) {
	if os.Getenv("VROOLI_PROCESS_HANDLE_HELPER") != "1" {
		return
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}

func startHandleFixture(t *testing.T) (*exec.Cmd, Handle) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessHandleHelper$")
	cmd.Env = append(os.Environ(), "VROOLI_PROCESS_HANDLE_HELPER=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	handle, err := OpenHandle(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = handle.Close() })
	return cmd, handle
}

func TestProcessHandleRemainsBoundAfterExit(t *testing.T) {
	for _, force := range []bool{false, true} {
		name := "graceful"
		if force {
			name = "forced"
		}
		t.Run(name, func(t *testing.T) {
			cmd, handle := startHandleFixture(t)
			if alive, err := handle.Alive(); err != nil || !alive {
				t.Fatalf("live handle: alive=%v err=%v", alive, err)
			}
			if err := handle.Signal(force); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(3 * time.Second)
			for {
				alive, err := handle.Alive()
				if err != nil {
					t.Fatal(err)
				}
				if !alive {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("signaled process did not exit")
				}
				time.Sleep(time.Millisecond)
			}
			_ = cmd.Wait() // Reap it; the handle must remain tied to the dead incarnation.
			_, other := startHandleFixture(t)
			if err := handle.Signal(true); !errors.Is(err, unix.ESRCH) {
				t.Fatalf("signal reaped handle = %v, want ESRCH", err)
			}
			if alive, err := other.Alive(); err != nil || !alive {
				t.Fatalf("unrelated process affected: alive=%v err=%v", alive, err)
			}
			if err := handle.Close(); err != nil {
				t.Fatal(err)
			}
			if err := handle.Close(); err != nil {
				t.Fatalf("repeated close: %v", err)
			}
			if _, err := handle.Alive(); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("closed handle liveness = %v", err)
			}
			if err := handle.Signal(true); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("closed handle signal = %v", err)
			}
		})
	}
}

func TestProcessHandleRejectsInvalidPID(t *testing.T) {
	for _, pid := range []int{-1, 0} {
		if handle, err := OpenHandle(pid); err == nil || handle != nil {
			t.Fatalf("OpenHandle(%d) = %v, %v", pid, handle, err)
		}
	}
}
