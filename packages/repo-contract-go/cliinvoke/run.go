package cliinvoke

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const (
	// DefaultTimeout bounds any single vrooli invocation. Lifecycle commands
	// are the slowest thing a supervisor runs (a start does setup, develop,
	// and a health wait), so this is generous: a backstop against hanging
	// forever, not a latency target.
	DefaultTimeout = 10 * time.Minute
	// DefaultWaitDelay is how long Wait keeps reading a finished command's
	// output before giving up on descendants that inherited the pipe.
	DefaultWaitDelay = 5 * time.Second
	// stderrCaptureLimit bounds the stderr tail kept for classification when
	// the caller supplied its own writer.
	stderrCaptureLimit = 64 * 1024
)

// Invocation describes one vrooli subprocess.
type Invocation struct {
	// Binary is the resolved executable path (see Resolve). Required.
	Binary string
	// Args is the argv after the binary. It must not start with a global
	// flag: supervisors pass behavior switches as VROOLI_* environment
	// variables so an older or newer CLI ignores them harmlessly.
	Args []string
	// Dir is the working directory; empty inherits the caller's.
	Dir string
	// Env, when non-nil, is the complete child environment. Callers that
	// need boundary-aware derivation compose it with envkit and pass the
	// result; nil inherits the process environment.
	Env []string
	// Timeout defaults to DefaultTimeout; WaitDelay to DefaultWaitDelay.
	Timeout   time.Duration
	WaitDelay time.Duration
	// Stdout and Stderr receive the streams as they are produced. When nil,
	// the stream is captured into the Result instead.
	Stdout io.Writer
	Stderr io.Writer
}

// Result is what one invocation produced.
type Result struct {
	Class    Class
	ExitCode int
	// Stdout holds the captured standard output when Invocation.Stdout was
	// nil. Stderr always holds (a bounded tail of) the error stream so the
	// class can be explained.
	Stdout []byte
	Stderr []byte
	// Err is the underlying error for non-OK classes; nil on OK.
	Err      error
	Duration time.Duration
	// Argv is the binary plus args, for logs and evidence.
	Argv []string
}

// Combined returns stdout followed by stderr, for callers that used to read
// exec.CombinedOutput.
func (r Result) Combined() []byte {
	if len(r.Stderr) == 0 {
		return r.Stdout
	}
	out := make([]byte, 0, len(r.Stdout)+len(r.Stderr))
	out = append(out, r.Stdout...)
	return append(out, r.Stderr...)
}

// Error renders the result as an error for callers that only need one.
func (r Result) Error() error {
	if r.Class == OK {
		return nil
	}
	summary := strings.TrimSpace(string(r.Stderr))
	if summary == "" {
		summary = strings.TrimSpace(string(r.Stdout))
	}
	if summary != "" {
		return fmt.Errorf("%s (%s): %w: %s", strings.Join(r.Argv, " "), r.Class, r.Err, summary)
	}
	return fmt.Errorf("%s (%s): %w", strings.Join(r.Argv, " "), r.Class, r.Err)
}

// Run executes the invocation with the inherited-pipe discipline.
//
// The plain exec.Cmd.CombinedOutput cannot be used by a supervisor, and the
// reason is the 2026-08-01 outage. CombinedOutput hands the child a PIPE and
// reads it until EOF. EOF arrives when every write end is closed, not when
// the child exits. `vrooli scenario start` spawns the long-lived runtime
// supervisor, which inherits that pipe as its stderr and holds it open for as
// long as it runs, which is forever. So the autoheal loop's very first action
// on boot blocked in Wait permanently: it never reached its tick loop and
// never started a single scenario. Observed directly: the loop parked in
// futex_do_wait holding the read end of pipe:[623670] while the supervisor
// held the write end as fd 2.
//
// Two things fix it, and both are here because either alone leaves a hole:
//
//   - WaitDelay bounds how long Wait keeps reading after the process itself
//     has exited, so an inherited pipe delays the result by seconds instead
//     of ending the supervisor's useful life.
//   - The context timeout bounds the command itself, for the different
//     failure where the CLI genuinely hangs rather than exiting and leaking
//     a pipe.
//
// A supervisor that can be blocked forever by the thing it supervises is not
// a supervisor.
func Run(ctx context.Context, inv Invocation) Result {
	started := time.Now()
	argv := append([]string{inv.Binary}, inv.Args...)
	result := Result{Argv: argv}
	if strings.TrimSpace(inv.Binary) == "" {
		result.Class, result.Err = BinaryMissing, &BinaryMissingError{Candidates: []string{"(empty binary path)"}}
		return result
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := inv.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	waitDelay := inv.WaitDelay
	if waitDelay <= 0 {
		waitDelay = DefaultWaitDelay
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The binary is invoked directly on every platform. A shell or
	// powershell wrapper would re-split the argv and turn a quoted path with
	// a space into a different command.
	cmd := exec.CommandContext(ctx, inv.Binary, inv.Args...)
	cmd.Dir = inv.Dir
	if inv.Env != nil {
		cmd.Env = inv.Env
	}
	cmd.WaitDelay = waitDelay

	var stdout, stderr bytes.Buffer
	if inv.Stdout != nil {
		cmd.Stdout = inv.Stdout
	} else {
		cmd.Stdout = &stdout
	}
	tail := &boundedBuffer{limit: stderrCaptureLimit, buf: &stderr}
	if inv.Stderr != nil {
		cmd.Stderr = io.MultiWriter(inv.Stderr, tail)
	} else {
		cmd.Stderr = tail
	}

	err := cmd.Run()
	// A WaitDelay expiry means the command finished but something it
	// spawned still holds the output pipe. The command's own result is what
	// was asked for, and it is complete; the leftover descendant is expected
	// and is not this call's failure.
	if errors.Is(err, exec.ErrWaitDelay) {
		err = nil
	}
	result.Duration = time.Since(started)
	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err == nil {
		result.Class = OK
		return result
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = fmt.Errorf("%s timed out after %v: %w", strings.Join(argv, " "), timeout, context.DeadlineExceeded)
	}
	result.Err = err
	result.Class = Classify(ctx, err, result.Stderr)
	return result
}

// boundedBuffer keeps the last `limit` bytes written to it.
type boundedBuffer struct {
	limit int
	buf   *bytes.Buffer
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.buf.Write(p)
	if b.buf.Len() > b.limit {
		excess := b.buf.Len() - b.limit
		b.buf.Next(excess)
	}
	return len(p), nil
}
