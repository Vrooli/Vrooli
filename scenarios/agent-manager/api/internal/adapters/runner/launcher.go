// Package runner — Launcher seam.
//
// Launcher abstracts the act of starting a long-running agent process so
// the runner code is decoupled from where the process actually executes.
// Two implementations:
//
//   - HostLauncher: launches via os/exec on the host. Preserves the
//     existing behavior (process group, idle timeout, grandchild cleanup).
//
//   - SandboxLauncher: launches through workspace-sandbox /processes APIs
//     for protected-mode runs. Enforces the workspace-sandbox guardrails
//     (git allowlist, network mode, bwrap isolation) on the agent process
//     tree itself.
//
// The seam was introduced by execute/protected-sandbox-agent-launch (the
// runner-fork) and is documented in
// scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md.
package runner

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// SandboxLauncherFactory builds a per-run sandbox launcher. The runner
// holds a factory at construction time and asks it for a launcher per
// Execute call when the request is for a protected-mode sandboxed run.
//
// Defining this in the runner package (rather than importing the sandbox
// package directly) avoids a sandbox→runner→sandbox import cycle, since
// the sandbox package's WorkspaceSandboxProvider already imports runner
// to satisfy this interface.
type SandboxLauncherFactory interface {
	// LauncherFor returns a Launcher bound to the given sandbox. Returns
	// nil if the factory cannot produce a launcher for this sandbox (e.g.,
	// the underlying provider is not the workspace-sandbox provider).
	LauncherFor(sandboxID uuid.UUID) Launcher
}

// Launcher launches long-running agent processes.
//
// Implementations must:
//   - Honor ctx cancellation by terminating the process tree.
//   - Provide stdout/stderr as io.Reader streams that close on process exit.
//   - Implement Kill as a synchronous "issue SIGKILL" that returns promptly;
//     callers use Wait to block on actual exit.
//   - Be safe for concurrent Wait/Kill/Signal calls (idempotent).
type Launcher interface {
	Launch(ctx context.Context, req LaunchRequest) (LaunchedProcess, error)
}

// LaunchRequest carries everything a runner needs to start its agent process.
type LaunchRequest struct {
	// Command is the executable to run. The runner is responsible for any
	// pre-resolution (exec.LookPath for host); the launcher passes Command
	// straight through. Use "env" with Args[0]="KEY=VALUE" to inject a
	// reconciler-visible tag into /proc/<pid>/cmdline.
	Command string

	// Args are the command arguments (NOT including Command itself).
	Args []string

	// Env is the full environment for the process in os.Environ() form
	// ("KEY=VALUE"). Replaces the parent process's environment.
	Env []string

	// WorkingDir is the directory in which to start the process. For
	// SandboxLauncher this must be inside the sandbox merged dir.
	WorkingDir string

	// Stdin, when non-nil, is io.Copy'd to the process's stdin and then
	// closed. Pass strings.NewReader(prompt) for runners that pipe a
	// prompt to stdin (claude_code, codex, opencode).
	Stdin io.Reader

	// IdleTimeout, when > 0, kills the process if it produces no stdout
	// for the given duration. Reset by ResetIdleTimer on each successful
	// stdout read. Zero disables (the runner default — agents legitimately
	// produce no stdout for long stretches during heavy builds/tests).
	IdleTimeout time.Duration
}

// LaunchedProcess is the live process handle returned by a Launcher.
type LaunchedProcess interface {
	// Stdout returns a reader for the process's standard output. Closes
	// (returns io.EOF) when the process exits or Kill is called.
	Stdout() io.Reader

	// Stderr returns a reader for the process's standard error.
	Stderr() io.Reader

	// ResetIdleTimer resets the idle-timeout safety net (no-op when
	// LaunchRequest.IdleTimeout was zero). Call from the stdout scanner
	// loop on each successful read.
	ResetIdleTimer()

	// TimedOut reports whether the idle-timeout fired during this run.
	TimedOut() bool

	// Kill issues a synchronous SIGKILL (or equivalent) to the process
	// and any descendants. Idempotent. Use Wait to block on actual exit.
	Kill()

	// Signal initiates graceful termination (SIGTERM-equivalent). After
	// the grace duration, escalates to Kill in a background goroutine.
	// Idempotent. Used by Runner.Stop() for graceful agent shutdown.
	Signal(grace time.Duration)

	// Wait blocks until the process has exited and resources are released.
	// Returns the exit error (typically *exec.ExitError for host launches,
	// or a typed remote-exit error for sandbox launches). Safe to call
	// multiple times — subsequent calls return the same value.
	Wait() error

	// PID returns the local-host process ID. For sandbox launches this is
	// the PID inside the sandbox namespace, useful for diagnostics and
	// for the workspace-sandbox kill endpoint.
	PID() int
}
