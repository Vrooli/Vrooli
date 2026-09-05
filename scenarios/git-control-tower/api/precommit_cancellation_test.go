package main

import (
	"context"
	"net/http/httptest"
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

// TestPrecommitStreamHandlerDoesNotHoldWriteLock is the Part A regression guard:
// while a precommit stream is running, the per-repo write lock must remain free
// so a concurrent (e.g. "Commit Anyway") commit can proceed immediately. Before
// the fix the handler used RepoWrite and held the lock for the whole run.
func TestPrecommitStreamHandlerDoesNotHoldWriteLock(t *testing.T) {
	rl := NewRepoLock()
	git := NewFakeGitRunner()
	runner := &blockingStreamingRunner{started: make(chan struct{}, 1), release: make(chan struct{})}
	svc := newTestPrecommitServiceWithRunner(t, runner)
	if _, err := svc.Save(context.Background(), git.RepoRoot, PrecommitConfig{
		Enabled: true, Command: "noop", WorkingDirectory: git.RepoRoot,
		TimeoutSeconds: 30, RunBeforeCommit: true, AllowOverride: true,
	}); err != nil {
		t.Fatalf("save precommit config: %v", err)
	}
	srv := &Server{git: git, repoLock: rl, precommit: svc}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/repo/precommit/run/stream", nil)
	handlerDone := make(chan struct{})
	go func() {
		srv.handlePrecommitRunStream(w, r)
		close(handlerDone)
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

	// Let the stream finish and the handler return cleanly.
	close(runner.release)
	select {
	case <-handlerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not return after release")
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
