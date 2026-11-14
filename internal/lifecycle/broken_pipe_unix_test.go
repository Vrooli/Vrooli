//go:build !windows

package lifecycle

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

// brokenPipeHelperEnv names the directory the helper child works in; it is
// set only when TestBrokenPipeHelperProcess runs as that child.
const brokenPipeHelperEnv = "VROOLI_LIFECYCLE_BROKEN_PIPE_HELPER"

// TestBrokenPipeHelperProcess is not a test on its own: it is the child that
// TestLifecycleMutationFinishesAfterItsOutputReaderGoes runs. Holding a
// scenario lock, it keeps writing to stdout after its reader has gone, and
// records how far it got in marker files.
func TestBrokenPipeHelperProcess(t *testing.T) {
	dir := os.Getenv(brokenPipeHelperEnv)
	if dir == "" {
		return
	}
	r := &Runner{Home: dir, Out: io.Discard, Err: io.Discard, Verbosity: VerbosityQuiet}
	release, err := r.acquireScenarioLock("broken-pipe-probe")
	if err != nil {
		os.Exit(3)
	}
	fmt.Println("ready")
	// The parent closes stdin once it has closed its end of stdout.
	_, _ = io.Copy(io.Discard, os.Stdin)
	for range 20 {
		fmt.Println("output during the mutation")
	}
	_ = os.WriteFile(filepath.Join(dir, "mutation-finished"), nil, 0o600)
	release()
	for range 20 {
		fmt.Println("output after the mutation")
	}
	_ = os.WriteFile(filepath.Join(dir, "wrote-after-release"), nil, 0o600)
	os.Exit(0)
}

// A lifecycle mutation must finish once begun, even when whoever reads the
// CLI's output goes away mid-operation (a `| head` that has its lines, a
// closed terminal). Once no mutation is in flight, a broken pipe ends the CLI
// as it always has, so `vrooli ... | head` still returns promptly.
func TestLifecycleMutationFinishesAfterItsOutputReaderGoes(t *testing.T) {
	dir := t.TempDir()
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestBrokenPipeHelperProcess$")
	cmd.Env = append(os.Environ(), brokenPipeHelperEnv+"="+dir)
	cmd.Stdin = stdinReader
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stdoutWriter
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = stdinReader.Close()
	_ = stdoutWriter.Close()

	line, err := bufio.NewReader(stdoutReader).ReadString('\n')
	if err != nil || line != "ready\n" {
		t.Fatalf("helper did not report ready: %q, %v", line, err)
	}
	// The reader goes away, then the helper is told to carry on writing.
	_ = stdoutReader.Close()
	_ = stdinWriter.Close()
	waitErr := cmd.Wait()

	if _, err := os.Stat(filepath.Join(dir, "mutation-finished")); err != nil {
		t.Fatalf("the process died mid-mutation once its output reader went away (wait: %v)", waitErr)
	}
	if _, err := os.Stat(filepath.Join(dir, "wrote-after-release")); err == nil {
		t.Fatal("a broken pipe no longer ends the process once the mutation is over")
	}
	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		t.Fatalf("helper exit: %v, want death by SIGPIPE after the mutation", waitErr)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGPIPE {
		t.Fatalf("helper exit: %v, want death by SIGPIPE after the mutation", waitErr)
	}
}
