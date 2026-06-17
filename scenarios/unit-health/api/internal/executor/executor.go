// Package executor runs planned test commands under hard bounds: a per-command
// timeout, a no-output watchdog, process-group cleanup, captured output
// excerpts, and a failure classification. It is deliberately ignorant of Unit
// Health's domain types so the validation package can fake it in tests.
package executor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Failure classes Unit Health distinguishes. The validation package maps these
// onto maturity finding codes.
const (
	ClassNone              = ""
	ClassTestFailure       = "test_failure"
	ClassMissingDependency = "missing_dependency"
	ClassMisconfiguration  = "misconfiguration"
	ClassTimeoutHang       = "timeout_hang"
	ClassNoOutputStall     = "no_output_stall"
	ClassSystem            = "system"
)

// Status values for a command result.
const (
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusTimeout = "timeout"
	StatusError   = "error"
)

const (
	defaultNoOutputTimeout = 2 * time.Minute
	maxExcerptBytes        = 8 << 10 // 8 KiB tail per stream
)

// Command is a single planned command to run.
type Command struct {
	WorkspaceID    string
	Name           string
	Argv           []string
	Dir            string
	TimeoutSeconds int
	// Env is appended to the inherited environment.
	Env []string
}

// Result is the outcome of running one Command.
type Result struct {
	WorkspaceID   string
	Name          string
	Command       string
	Status        string
	ExitCode      int
	Stdout        string
	Stderr        string
	FailureClass  string
	FailureReason string
	DurationMS    int64
}

// Runner executes a single Command.
type Runner interface {
	Run(ctx context.Context, cmd Command) Result
}

// Bounded is the default Runner. Zero value is usable.
type Bounded struct {
	// NoOutputTimeout cancels a command that produces no output for this long.
	// Defaults to defaultNoOutputTimeout when zero.
	NoOutputTimeout time.Duration
}

// RunAll executes commands with bounded concurrency, preserving input order in
// the returned results.
func RunAll(ctx context.Context, runner Runner, commands []Command, maxConcurrency int) []Result {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	results := make([]Result, len(commands))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	for i, cmd := range commands {
		wg.Add(1)
		go func(i int, cmd Command) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = runner.Run(ctx, cmd)
		}(i, cmd)
	}
	wg.Wait()
	return results
}

// Run executes one command under the configured bounds.
func (b Bounded) Run(ctx context.Context, cmd Command) Result {
	res := Result{WorkspaceID: cmd.WorkspaceID, Name: cmd.Name, Command: strings.Join(cmd.Argv, " ")}
	if len(cmd.Argv) == 0 {
		res.Status = StatusError
		res.FailureClass = ClassMisconfiguration
		res.FailureReason = "empty command"
		return res
	}
	if path, err := exec.LookPath(cmd.Argv[0]); err != nil || path == "" {
		res.Status = StatusError
		res.FailureClass = ClassMissingDependency
		res.FailureReason = "required command not found: " + cmd.Argv[0]
		return res
	}

	noOutput := b.NoOutputTimeout
	if noOutput <= 0 {
		noOutput = defaultNoOutputTimeout
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if cmd.TimeoutSeconds > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(cmd.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	// Watchdog context layered on top so the no-output stall can cancel
	// independently of the hard timeout.
	watchCtx, watchCancel := context.WithCancel(runCtx)
	defer watchCancel()

	c := exec.CommandContext(watchCtx, cmd.Argv[0], cmd.Argv[1:]...)
	if cmd.Dir != "" {
		c.Dir = cmd.Dir
	}
	c.Env = append(os.Environ(), "GOWORK=off", "CI=1")
	c.Env = append(c.Env, cmd.Env...)
	setProcessGroup(c)
	// On cancel (hard timeout or no-output stall) kill the whole process group
	// so children (e.g. pnpm -> node) die too. WaitDelay bounds how long Wait
	// blocks on output pipes still held open by orphaned grandchildren.
	c.Cancel = func() error {
		killGroup(c)
		return os.ErrProcessDone
	}
	c.WaitDelay = 3 * time.Second

	stdout := newTailWriter(maxExcerptBytes)
	stderr := newTailWriter(maxExcerptBytes)
	c.Stdout = stdout
	c.Stderr = stderr

	start := time.Now()
	if err := c.Start(); err != nil {
		res.Status = StatusError
		res.FailureClass = ClassSystem
		res.FailureReason = "failed to start command: " + err.Error()
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}

	stalled := watchNoOutput(watchCtx, watchCancel, []*tailWriter{stdout, stderr}, noOutput)
	waitErr := c.Wait()
	// Ensure the whole process group is gone even on normal exit.
	killGroup(c)

	res.DurationMS = time.Since(start).Milliseconds()
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()
	res.ExitCode = c.ProcessState.ExitCode()

	switch {
	case stalled.Load():
		res.Status = StatusTimeout
		res.FailureClass = ClassNoOutputStall
		res.FailureReason = "no output for " + noOutput.String() + "; likely hung"
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		res.Status = StatusTimeout
		res.FailureClass = ClassTimeoutHang
		res.FailureReason = "exceeded timeout"
	case waitErr == nil:
		res.Status = StatusPassed
		res.FailureClass = ClassNone
	default:
		res.Status = StatusFailed
		res.FailureClass = classifyFailure(stdout.String(), stderr.String())
		res.FailureReason = waitErr.Error()
	}
	return res
}

// classifyFailure inspects output to refine a nonzero exit into a more specific
// class than a bare test failure.
func classifyFailure(stdout, stderr string) string {
	combined := strings.ToLower(stdout + "\n" + stderr)
	switch {
	case strings.Contains(combined, "command not found"),
		strings.Contains(combined, "executable file not found"),
		strings.Contains(combined, "is not recognized"),
		// Node "Cannot find module 'x'" / pnpm lockfile errors mean the
		// dependency install is missing or stale, not a test misconfiguration.
		// The quote after "module" distinguishes node from Go's unquoted
		// "go: cannot find module providing package …".
		strings.Contains(combined, "err_module_not_found"),
		strings.Contains(combined, "cannot find module '"),
		strings.Contains(combined, "err_pnpm_no_lockfile"),
		strings.Contains(combined, "frozen-lockfile"),
		strings.Contains(combined, "missing dependencies in the lockfile"):
		return ClassMissingDependency
	case strings.Contains(combined, "no such tool"),
		strings.Contains(combined, "missing go.sum"),
		strings.Contains(combined, "cannot find module"),
		strings.Contains(combined, "no test files"),
		strings.Contains(combined, "cannot find package"):
		return ClassMisconfiguration
	default:
		return ClassTestFailure
	}
}
