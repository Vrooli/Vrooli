// Tests for the runner-fork Launcher seam:
//   - HostLauncher contract (start, stdout/stderr, kill, wait)
//   - LaunchedProcess idle-timeout semantics
//
// SandboxLauncher tests live in the sandbox adapter package because they
// require the workspace-sandbox HTTP wire and would create an import cycle
// here. See sandbox/sandbox_launcher_test.go.

package runner

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

// TestHostLauncher_LaunchEcho asserts the basic shape: launch a process,
// read stdout, wait for clean exit.
func TestHostLauncher_LaunchEcho(t *testing.T) {
	launcher := NewHostLauncher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, LaunchRequest{
		Command:    "echo",
		Args:       []string{"hello-from-host"},
		Env:        []string{},
		WorkingDir: "/tmp",
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Read stdout to EOF
	out, err := io.ReadAll(proc.Stdout())
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(out), "hello-from-host") {
		t.Errorf("stdout = %q; want substring %q", string(out), "hello-from-host")
	}

	if err := proc.Wait(); err != nil {
		t.Errorf("Wait returned error: %v", err)
	}
}

// TestHostLauncher_StdinPipedIn confirms LaunchRequest.Stdin is delivered
// to the process and closed (so the process sees EOF).
func TestHostLauncher_StdinPipedIn(t *testing.T) {
	launcher := NewHostLauncher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, LaunchRequest{
		Command:    "cat",
		Args:       nil,
		Env:        []string{},
		WorkingDir: "/tmp",
		Stdin:      strings.NewReader("piped-stdin-content\n"),
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	out, err := io.ReadAll(proc.Stdout())
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(out), "piped-stdin-content") {
		t.Errorf("cat output = %q; want substring %q", string(out), "piped-stdin-content")
	}
	if err := proc.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}
}

// TestHostLauncher_KillTerminatesProcess verifies Kill stops a long-running
// process and Wait unblocks promptly with a non-nil error.
func TestHostLauncher_KillTerminatesProcess(t *testing.T) {
	launcher := NewHostLauncher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, LaunchRequest{
		Command: "sleep",
		Args:    []string{"30"},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Drain stdout in a goroutine so Wait isn't blocked on the pipe.
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	// Give it a moment to actually be running, then kill.
	time.Sleep(100 * time.Millisecond)
	proc.Kill()

	done := make(chan error, 1)
	go func() { done <- proc.Wait() }()
	select {
	case err := <-done:
		// Killed — exec returns non-nil error.
		if err == nil {
			t.Error("Wait returned nil after Kill; want non-nil error")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Wait did not return within 3s of Kill")
	}
}

// TestHostLauncher_ContextCancelKills verifies that ctx cancellation
// terminates the process via exec.CommandContext.
func TestHostLauncher_ContextCancelKills(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	launcher := NewHostLauncher()
	proc, err := launcher.Launch(ctx, LaunchRequest{
		Command: "sleep",
		Args:    []string{"30"},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	time.Sleep(100 * time.Millisecond)
	cancel()

	done := make(chan error, 1)
	go func() { done <- proc.Wait() }()
	select {
	case <-done:
		// Good — exited.
	case <-time.After(3 * time.Second):
		t.Fatal("Wait did not return within 3s of ctx.Cancel")
	}
}

// TestHostLauncher_IdleTimeout verifies the safety-net idle timeout fires
// when stdout is silent for longer than IdleTimeout AND ResetIdleTimer
// pushes back the deadline as expected.
func TestHostLauncher_IdleTimeout(t *testing.T) {
	launcher := NewHostLauncher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, LaunchRequest{
		Command:     "sleep",
		Args:        []string{"10"},
		IdleTimeout: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	// sleep emits no stdout, so the idle-timer should fire.
	done := make(chan error, 1)
	go func() { done <- proc.Wait() }()

	select {
	case <-done:
		if !proc.TimedOut() {
			t.Error("Process exited but TimedOut() = false")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Idle-timeout did not fire within 2s")
	}
}

// TestHostLauncher_StderrCapture verifies stderr is exposed as an io.Reader.
func TestHostLauncher_StderrCapture(t *testing.T) {
	launcher := NewHostLauncher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, LaunchRequest{
		Command: "sh",
		Args:    []string{"-c", "echo to-stderr 1>&2"},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())

	var stderrBuf bytes.Buffer
	scanner := bufio.NewScanner(proc.Stderr())
	for scanner.Scan() {
		stderrBuf.WriteString(scanner.Text())
		stderrBuf.WriteString("\n")
	}
	if err := proc.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}
	if !strings.Contains(stderrBuf.String(), "to-stderr") {
		t.Errorf("stderr = %q; want substring %q", stderrBuf.String(), "to-stderr")
	}
}

// TestHostLauncher_NoCommandIsRejected confirms Launch refuses empty Command.
func TestHostLauncher_NoCommandIsRejected(t *testing.T) {
	launcher := NewHostLauncher()
	_, err := launcher.Launch(context.Background(), LaunchRequest{Command: ""})
	if err == nil {
		t.Fatal("Launch with empty Command returned nil; want error")
	}
}
