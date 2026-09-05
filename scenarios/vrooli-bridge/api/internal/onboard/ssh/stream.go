package ssh

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// KnownHostsPath returns the bridge-owned known_hosts file the Service pins all
// its ssh/scp invocations to. The onboard orchestrator threads it into the
// Config it builds for the post-first-touch SCP + streaming bootstrap exec so
// they share the same TOFU host-key store the first touch populated.
func (s *Service) KnownHostsPath() string { return s.knownHostsPath() }

// StreamOptions configures a streaming remote exec (RunStreaming).
type StreamOptions struct {
	// Run carries the underlying ssh connection/timeout options.
	Run RunOptions

	// Stdin is written to the remote command's standard input and then closed.
	// It is the ONE channel a secret may ride into the remote process without
	// appearing in argv (local or remote `ps`) or in any logged command string —
	// RunStreaming never logs Stdin. The onboard orchestrator uses it to hand the
	// single-use pairing code to the bootstrap script.
	Stdin []byte

	// StdinReader streams the remote command's standard input from an io.Reader
	// instead of a fixed Stdin buffer, for payloads too large to materialise in
	// memory (the working-tree tar the orchestrator pipes to a remote `tar -xf -`).
	// When set it takes precedence over Stdin. Like Stdin it is never logged.
	StdinReader io.Reader

	// OnStdoutLine is invoked for each stdout line as it arrives (newline
	// trimmed), so a long-running remote command's progress can be parsed and
	// persisted live. nil discards stdout.
	OnStdoutLine func(line string)
}

// RunStreaming executes command on the remote host described by cfg, writing
// opts.Stdin to the command's stdin (then closing it) and invoking
// opts.OnStdoutLine for each stdout line as it streams back. Stderr is captured
// (bounded) into the returned Result. The returned Result carries the remote
// exit code; a non-nil error is returned only for a connection/transport-level
// failure, NOT for a non-zero remote exit (the caller inspects Result.ExitCode).
//
// The command string is logged with the key path redacted (FormatCommandForLog);
// opts.Stdin is never logged, so a secret injected over stdin stays out of logs.
func (s *Service) RunStreaming(ctx context.Context, cfg Config, command string, opts StreamOptions) (Result, error) {
	if opts.Run.CommandTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Run.CommandTimeout)
		defer cancel()
	}

	args := buildSSHArgs(cfg, opts.Run)
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	args = append(args, target, "--", command)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	switch {
	case opts.StdinReader != nil:
		// Streamed stdin (e.g. a working-tree tar) — no in-memory materialisation.
		cmd.Stdin = opts.StdinReader
	case len(opts.Stdin) > 0:
		// bytes.NewReader (not strings.NewReader(string(...))) so no immutable,
		// unzeroable copy of a secret stdin is ever materialised — the caller's
		// []byte remains the only copy and can be wiped after this returns.
		cmd.Stdin = bytes.NewReader(opts.Stdin)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("ssh stdout pipe: %w", err)
	}
	stderr := newBoundedBuffer(opts.Run.maxOutput())
	cmd.Stderr = stderr

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return Result{}, newCommandError(err, Result{}, cfg.Host)
	}

	// Scan stdout line-by-line on this goroutine; the bootstrap stream is small
	// and strictly ordered, so a single consumer keeps the persisted event order
	// identical to the emitted order.
	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if opts.OnStdoutLine != nil {
			opts.OnStdoutLine(scanner.Text())
		}
	}
	scanErr := scanner.Err()

	waitErr := cmd.Wait()
	duration := time.Since(start)

	result := Result{
		Stderr:   strings.TrimRight(stderr.String(), "\n"),
		ExitCode: exitCode(waitErr),
	}

	slog.Info("ssh.stream_executed",
		"host", cfg.Host,
		"command", FormatCommandForLog(cfg, command),
		"exit_code", result.ExitCode,
		"duration_ms", duration.Milliseconds(),
	)

	// A stdout read error (not EOF) is a transport problem worth surfacing; a
	// non-zero remote exit is reported via Result.ExitCode, not an error.
	if scanErr != nil && scanErr != io.EOF {
		return result, fmt.Errorf("ssh stream read: %w", scanErr)
	}
	// A real remote non-zero exit (*exec.ExitError) is not an error here — the
	// caller reads Result.ExitCode. Any other Wait error is a transport failure.
	if waitErr != nil {
		var ee *exec.ExitError
		if !errors.As(waitErr, &ee) {
			return result, newCommandError(waitErr, result, cfg.Host)
		}
	}
	return result, nil
}
