// Package process — Starter seam.
//
// Starter is the canonical exec abstraction for workspace-sandbox. Every
// production code path that spawns an external process goes through this
// interface; tests inject a FakeStarter from internal/testutil/mocks.
//
// Why this exists (Round 4 Phase 7):
//   - Mid-mount failures, process-fails-to-start, hung Wait, and
//     OS-signal kill paths were impossible to exercise from `go test`
//     because the driver layer constructed `*os/exec.Cmd` directly. The
//     2026-04-28 SSE flusher bug shipped because no test could simulate
//     a fast-failing process; the symptom only surfaced when a real
//     binary segfaulted in production.
//   - This seam unifies every exec call site (driver overlay mounts,
//     fuse-overlayfs subprocess, fusermount, mountpoint, modprobe, the
//     bwrap exec layer, the namespace probe, capability probes, and
//     diff/git/patch via diff.CommandRunner) under a single boundary
//     so the failure modes above can be injected deterministically.
//
// Contract:
//   - Start spawns a process and returns a Handle. Stdin/Stdout/Stderr
//     piping is wired before the process starts.
//   - Handle.Wait blocks until the process exits, returning a
//     ProcessExit with exit code, terminating signal (when applicable),
//     and OOM flag.
//   - Handle.Kill sends SIGKILL to the process; KillProcessGroup sends
//     SIGKILL to the entire process group (used by the agent runner to
//     reap shells with grandchildren).
//   - LookPath wraps os/exec.LookPath so capability probes are
//     injectable.
//   - Run / RunCombinedOutput are convenience helpers for synchronous
//     one-shots that capture output. They are implemented in terms of
//     Start+Wait so a fake Starter only needs to implement Start.
//
// See docs/SEAMS.md "Process Starter Seam (Round 4 Phase 7)".
package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"syscall"
)

// ErrBinaryNotFound is returned by Starter.LookPath when the named
// binary is not on PATH. Wraps the underlying os/exec error so callers
// using errors.Is keep working.
var ErrBinaryNotFound = errors.New("binary not found in PATH")

// StartOpts configures a process spawn. All fields are optional except
// Path. Any nil writer/reader defaults to os.DevNull-equivalent behavior
// (stdout/stderr discarded; stdin closed).
type StartOpts struct {
	// Path is the absolute path or PATH-resolvable name of the binary.
	Path string

	// Args is the argv excluding Path. The starter prepends Path so
	// callers can pass exactly the arguments they want to send.
	Args []string

	// Dir is the working directory for the spawned process. Empty means
	// inherit the parent's working directory.
	Dir string

	// Env is the full environment for the spawned process. nil means
	// inherit os.Environ(); an empty (non-nil) slice means an empty
	// environment.
	Env []string

	// Stdin, when non-nil, is wired to the process's stdin pipe.
	Stdin io.Reader

	// Stdout / Stderr, when non-nil, receive the process's stdout/stderr.
	// Nil writers discard the stream (io.Discard).
	Stdout io.Writer
	Stderr io.Writer

	// SysProcAttr forwards low-level process attributes (notably
	// Setpgid for process-group reaping). Nil leaves the kernel default.
	SysProcAttr *syscall.SysProcAttr
}

// ProcessExit describes how a process terminated. Mirrors what
// driver/exec previously returned via ExitInfoFromState; the rest of
// the system composes this into process.ExitInfo (which adds StoppedAt)
// when it needs a wall-clock timestamp.
type ProcessExit struct {
	// ExitCode is the process exit status. For signal-killed processes
	// ExitCode is -1 and Signal carries the signal number.
	ExitCode int

	// Signal is the terminating signal number, or 0 when the process
	// exited normally.
	Signal int

	// OOMKilled is true when the kernel reported the process was killed
	// by the OOM killer. Best-effort: set by the runtime impl when
	// observable from WaitStatus.
	OOMKilled bool
}

// Handle is the live reference to a running process. Implementations
// own the underlying *os/exec.Cmd (or test fixture) and guarantee Wait
// is safe to call exactly once; subsequent calls return the same exit.
type Handle interface {
	// PID returns the process ID. Available immediately after Start.
	PID() int

	// Wait blocks until the process exits (or ctx is canceled, in which
	// case the impl SIGKILLs the process and returns ctx.Err()). Safe
	// to call once.
	Wait(ctx context.Context) (ProcessExit, error)

	// Kill sends SIGKILL to the process. Idempotent: returns nil if the
	// process has already exited.
	Kill() error

	// KillProcessGroup sends SIGKILL to the process group rooted at the
	// process's PID. Used to reap shells that spawn grandchildren.
	// Idempotent.
	KillProcessGroup() error
}

// Starter is the canonical exec seam. Production code wires
// OSExecStarter; tests wire FakeStarter from internal/testutil/mocks.
type Starter interface {
	// Start spawns a process and returns a Handle.
	Start(ctx context.Context, opts StartOpts) (Handle, error)

	// LookPath resolves a binary name to an absolute path. Returns
	// ErrBinaryNotFound when the binary isn't on PATH.
	LookPath(name string) (string, error)
}

// RunResult is returned by Run and RunCombinedOutput.
type RunResult struct {
	// PID is the spawned process's PID (captured at Start time so it
	// remains observable after Wait).
	PID int

	// Exit is the terminal state of the process.
	Exit ProcessExit

	// Stdout / Stderr are the captured outputs. For RunCombinedOutput,
	// the combined stream is in Stdout and Stderr is empty.
	Stdout []byte
	Stderr []byte
}

// Run starts a process, captures stdout and stderr separately, waits
// for it to exit, and returns the result. opts.Stdout / opts.Stderr
// are overridden — pass them via Start directly if you need streaming.
//
// Run propagates Wait's error (e.g. context cancellation) so callers
// can distinguish "ran and exited non-zero" (err==nil, Exit.ExitCode!=0)
// from "didn't run / was canceled" (err!=nil).
func Run(ctx context.Context, s Starter, opts StartOpts) (RunResult, error) {
	var stdout, stderr bytes.Buffer
	opts.Stdout = &stdout
	opts.Stderr = &stderr
	h, err := s.Start(ctx, opts)
	if err != nil {
		return RunResult{}, err
	}
	exit, waitErr := h.Wait(ctx)
	return RunResult{
		PID:    h.PID(),
		Exit:   exit,
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}, waitErr
}

// RunCombinedOutput is Run but with stdout and stderr merged into a
// single buffer (mirroring *exec.Cmd.CombinedOutput). The combined
// stream lands in result.Stdout; result.Stderr is always empty.
func RunCombinedOutput(ctx context.Context, s Starter, opts StartOpts) (RunResult, error) {
	var combined bytes.Buffer
	opts.Stdout = &combined
	opts.Stderr = &combined
	h, err := s.Start(ctx, opts)
	if err != nil {
		return RunResult{}, err
	}
	exit, waitErr := h.Wait(ctx)
	return RunResult{
		PID:    h.PID(),
		Exit:   exit,
		Stdout: combined.Bytes(),
	}, waitErr
}

// CommandExists is a tiny convenience wrapper: returns true when
// LookPath succeeds. Several capability probes need exactly this.
func CommandExists(s Starter, name string) bool {
	_, err := s.LookPath(name)
	return err == nil
}

// =============================================================================
// OSExecStarter — production implementation backed by os/exec
// =============================================================================

// OSExecStarter runs real subprocesses via os/exec. The zero value is
// usable; NewOSExecStarter is provided for symmetry with other seams.
type OSExecStarter struct{}

// NewOSExecStarter returns the production Starter. Equivalent to the
// zero value; provided so DI sites read uniformly.
func NewOSExecStarter() *OSExecStarter { return &OSExecStarter{} }

// LookPath delegates to os/exec.LookPath, normalizing not-found errors
// to ErrBinaryNotFound for callers using errors.Is.
func (OSExecStarter) LookPath(name string) (string, error) {
	path, err := osexec.LookPath(name)
	if err != nil {
		if errors.Is(err, osexec.ErrNotFound) {
			return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, name)
		}
		return "", err
	}
	return path, nil
}

// Start wraps os/exec to spawn a process per StartOpts.
func (OSExecStarter) Start(ctx context.Context, opts StartOpts) (Handle, error) {
	if opts.Path == "" {
		return nil, errors.New("process.Start: opts.Path is required")
	}
	cmd := osexec.CommandContext(ctx, opts.Path, opts.Args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if opts.Env != nil {
		cmd.Env = opts.Env
	}
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else {
		cmd.Stdout = io.Discard
	}
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stderr = io.Discard
	}
	if opts.SysProcAttr != nil {
		cmd.SysProcAttr = opts.SysProcAttr
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", opts.Path, err)
	}
	return &osExecHandle{cmd: cmd, pid: cmd.Process.Pid}, nil
}

// osExecHandle is the production Handle implementation.
type osExecHandle struct {
	cmd     *osexec.Cmd
	pid     int
	waitErr error
	exit    ProcessExit
	waited  bool
}

func (h *osExecHandle) PID() int { return h.pid }

func (h *osExecHandle) Wait(ctx context.Context) (ProcessExit, error) {
	if h.waited {
		return h.exit, h.waitErr
	}
	// We respect ctx by killing the process when ctx is canceled before
	// the process exits. This mirrors *os/exec.CommandContext semantics
	// (which already ties cmd to ctx) and ensures hung processes are
	// reaped when the caller cancels.
	done := make(chan error, 1)
	go func() { done <- h.cmd.Wait() }()
	select {
	case err := <-done:
		h.waitErr = err
		h.exit = exitFromState(h.cmd.ProcessState, err)
	case <-ctx.Done():
		_ = h.killUnlocked()
		<-done // drain
		h.waitErr = ctx.Err()
		h.exit = ProcessExit{ExitCode: -1, Signal: int(syscall.SIGKILL)}
	}
	h.waited = true
	return h.exit, h.waitErr
}

func (h *osExecHandle) Kill() error {
	if h.cmd == nil || h.cmd.Process == nil {
		return nil
	}
	return h.killUnlocked()
}

func (h *osExecHandle) killUnlocked() error {
	if err := h.cmd.Process.Kill(); err != nil {
		// Already-finished is not an error to surface.
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}
	return nil
}

func (h *osExecHandle) KillProcessGroup() error {
	if h.cmd == nil || h.cmd.Process == nil {
		return nil
	}
	return killProcessGroup(h.pid)
}

// killProcessGroup sends SIGKILL to the entire process group rooted at
// pid. Falls back to a single-process kill when getpgid fails (no
// dedicated group, e.g. caller forgot Setpgid).
func killProcessGroup(pid int) error {
	pgid, err := sysGetpgid(pid)
	if err != nil {
		if killErr := sysKill(pid, syscall.SIGKILL); killErr != nil && !isProcessGone(killErr) {
			return killErr
		}
		return nil
	}
	if killErr := sysKill(-pgid, syscall.SIGKILL); killErr != nil && !isProcessGone(killErr) {
		return killErr
	}
	return nil
}

// IsProcessRunning reports whether a process with the given PID is
// alive (signal 0 round-trip). Centralized here so callers don't have
// to re-implement the syscall dance; the FakeStarter doesn't need to
// satisfy this since tests rarely poke at host PIDs directly.
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// KillProcessGroupByPID is the package-level helper used by callers that
// only have a PID (not a Handle), e.g. the orphan-process reaper. In
// tests this can be replaced by overriding the package var; production
// always uses the syscall path.
var KillProcessGroupByPID = killProcessGroup

// =============================================================================
// ProcessExit / *os.ProcessState extraction (shared with driver/exec)
// =============================================================================

// exitFromState is the canonical translation from *os.ProcessState +
// wait error to a ProcessExit. Exported as ExitFromState so legacy
// callers (driver/exec/run.go) share the implementation while we
// migrate them onto Starter.
func exitFromState(state *os.ProcessState, waitErr error) ProcessExit {
	if state == nil {
		if waitErr != nil {
			return ProcessExit{ExitCode: -1}
		}
		return ProcessExit{ExitCode: 0}
	}
	if status, ok := state.Sys().(syscall.WaitStatus); ok {
		if status.Signaled() {
			return ProcessExit{ExitCode: -1, Signal: int(status.Signal())}
		}
		if status.Exited() {
			return ProcessExit{ExitCode: status.ExitStatus()}
		}
	}
	return ProcessExit{ExitCode: state.ExitCode()}
}

// ExitFromState is the public alias for exitFromState. Exported so the
// existing exec.ExitInfoFromState wrapper can delegate during the
// migration window inside this same Round 4 Phase 7 PR.
func ExitFromState(state *os.ProcessState, waitErr error) ProcessExit {
	return exitFromState(state, waitErr)
}
