package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// blockingStreamingRunner starts, signals on started, then blocks until release
// is closed (or ctx is cancelled). It lets a test observe a precommit run while
// it is in flight.
type blockingStreamingRunner struct {
	started chan struct{}
	release chan struct{}
}

func (r *blockingStreamingRunner) Run(ctx context.Context, req CommandRunRequest) (CommandRunResult, error) {
	return r.RunStream(ctx, req, nil)
}

func (r *blockingStreamingRunner) RunStream(ctx context.Context, _ CommandRunRequest, onLine func(stream, line string)) (CommandRunResult, error) {
	if onLine != nil {
		onLine("stdout", "running")
	}
	select {
	case r.started <- struct{}{}:
	default:
	}
	select {
	case <-r.release:
		return CommandRunResult{Stdout: "done\n"}, nil
	case <-ctx.Done():
		return CommandRunResult{}, ctx.Err()
	}
}

// TestPrecommitRunDoesNotHoldWriteLock is the Part A regression guard:
// while a typed precommit run is running, the per-repo write lock must remain free
// so a concurrent (e.g. "Commit Anyway") commit can proceed immediately. Before
// the fix the handler used RepoWrite and held the lock for the whole run.
func TestPrecommitStreamHandlerDoesNotHoldWriteLock(t *testing.T) {
	rl := NewRepoLock()
	git := NewFakeGitRunner()
	runner := &blockingStreamingRunner{started: make(chan struct{}, 1), release: make(chan struct{})}
	svc := newTestPrecommitServiceWithRunner(t, runner)
	if _, err := svc.Save(authorizedHumanContext(), git.RepoRoot, PrecommitConfig{
		Enabled: true, Command: "noop", WorkingDirectory: git.RepoRoot,
		TimeoutSeconds: 30, RunBeforeCommit: true, AllowOverride: true,
	}); err != nil {
		t.Fatalf("save precommit config: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan struct{})
	go func() {
		_, _ = svc.Run(ctx, git.RepoRoot, PrecommitRunRequest{})
		close(runDone)
	}()

	// Wait until the stream is genuinely mid-run.
	select {
	case <-runner.started:
	case <-time.After(5 * time.Second):
		t.Fatal("precommit stream never started")
	}

	// The write lock must be immediately acquirable despite the running stream.
	acqCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	unlock, err := rl.Acquire(acqCtx, git.RepoRoot)
	if err != nil {
		t.Fatalf("write lock was held by the running precommit stream: %v", err)
	}
	unlock()

	// Let the typed run finish and return cleanly.
	close(runner.release)
	select {
	case <-runDone:
	case <-time.After(5 * time.Second):
		t.Fatal("precommit run did not return after release")
	}
}

// TestShellCommandRunStreamKillsProcessGroupOnCancel is the Part B regression
// guard: cancelling the context must terminate the entire process tree, not
// just the bash parent. A grandchild that outlives bash (and keeps the output
// pipes open) previously blocked RunStream from returning and kept burning CPU.
func TestShellCommandRunStreamKillsProcessGroupOnCancel(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "survived")
	// A backgrounded subshell sleeps well past the test, then would touch the
	// marker. It inherits bash's stdout pipe, so if only bash is killed the pipe
	// stays open and RunStream hangs. `wait` keeps bash alive meanwhile.
	command := "( sleep 30; touch " + marker + " ) & echo CHILD=$!; wait"

	ctx, cancel := context.WithCancel(context.Background())
	runner := ShellCommandRunner{}
	pidCh := make(chan int, 1)
	done := make(chan error, 1)
	go func() {
		_, err := runner.RunStream(ctx, CommandRunRequest{Command: command, WorkingDirectory: dir}, func(_, line string) {
			if strings.HasPrefix(line, "CHILD=") {
				if pid, convErr := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "CHILD="))); convErr == nil {
					select {
					case pidCh <- pid:
					default:
					}
				}
			}
		})
		done <- err
	}()

	var childPid int
	select {
	case childPid = <-pidCh:
	case <-time.After(5 * time.Second):
		t.Fatal("never observed the backgrounded grandchild pid")
	}

	cancel()

	// RunStream must return promptly — far under the 30s grandchild sleep.
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("RunStream did not return after cancel — a surviving grandchild kept the pipe open")
	}

	// The grandchild must have been killed with its group.
	gone := false
	for i := 0; i < 100; i++ {
		if err := syscall.Kill(childPid, 0); err == syscall.ESRCH {
			gone = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !gone {
		t.Fatalf("grandchild pid %d survived cancellation — process group was not killed", childPid)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("grandchild completed its work despite cancellation")
	}
}

func TestShellCommandRunStreamWaitDelayBoundsEscapedPipeHolder(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid not available")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pidCh := make(chan int, 1)
	done := make(chan error, 1)
	command := "setsid sh -c 'sleep 30' & echo ESCAPED=$!; wait"
	go func() {
		_, err := (ShellCommandRunner{}).RunStream(ctx, CommandRunRequest{Command: command, WorkingDirectory: t.TempDir()}, func(_, line string) {
			if strings.HasPrefix(line, "ESCAPED=") {
				if pid, err := strconv.Atoi(strings.TrimPrefix(line, "ESCAPED=")); err == nil {
					pidCh <- pid
				}
			}
		})
		done <- err
	}()
	var pid int
	select {
	case pid = <-pidCh:
	case <-time.After(5 * time.Second):
		t.Fatal("never observed escaped child pid")
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	parentSession := processSession(t, os.Getpid())
	deadline := time.Now().Add(2 * time.Second)
	for {
		session := processSessionOrZero(pid)
		if session != 0 && session != parentSession {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("child %d did not escape into a separate session", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// The child is outside the command process group, so cancellation cannot
	// close its inherited stdout. os/exec's WaitDelay must bound the copy wait.
	cancelAt := time.Now()
	cancel()
	select {
	case <-done:
	case <-time.After(shellCommandWaitDelay + 3*time.Second):
		t.Fatal("RunStream remained blocked on an escaped descendant's output")
	}
	if elapsed := time.Since(cancelAt); elapsed < shellCommandWaitDelay-250*time.Millisecond {
		t.Fatalf("RunStream returned in %s; escaped pipe holder did not exercise WaitDelay", elapsed)
	}
}

func processSession(t *testing.T, pid int) int {
	t.Helper()
	if session := processSessionOrZero(pid); session != 0 {
		return session
	}
	t.Fatalf("cannot read session for process %d", pid)
	return 0
}

func processSessionOrZero(pid int) int {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	// comm is parenthesized and may contain spaces; fields after its final ')'
	// begin with state, ppid, pgrp, session.
	end := strings.LastIndexByte(string(data), ')')
	if end < 0 {
		return 0
	}
	fields := strings.Fields(string(data[end+1:]))
	if len(fields) < 4 {
		return 0
	}
	session, _ := strconv.Atoi(fields[3])
	return session
}
