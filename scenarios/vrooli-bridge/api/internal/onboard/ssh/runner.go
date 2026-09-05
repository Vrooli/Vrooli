package ssh

import (
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

// Runner executes SSH commands on a remote host.
type Runner interface {
	Run(ctx context.Context, cfg Config, command string, opts RunOptions) (Result, error)
}

// SCPRunner transfers files to a remote host via SCP.
type SCPRunner interface {
	Copy(ctx context.Context, cfg Config, localPath, remotePath string, opts SCPOptions) error
}

// exitCode extracts the exit code from an exec error (0 for nil, the process
// code for *exec.ExitError, or 255 for connection-level failures).
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 255
}

// ExecRunner implements Runner using os/exec.
type ExecRunner struct{}

// runSSH executes an SSH command via os/exec with bounded output capture.
func runSSH(ctx context.Context, cfg Config, command string, opts RunOptions) (Result, error) {
	if opts.CommandTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.CommandTimeout)
		defer cancel()
	}

	args := buildSSHArgs(cfg, opts)
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	args = append(args, target, "--", command)

	maxOut := opts.maxOutput()
	cmd := exec.CommandContext(ctx, "ssh", args...)
	stdout := newBoundedBuffer(maxOut)
	stderr := newBoundedBuffer(maxOut)
	cmd.Stdout = io.MultiWriter(stdout)
	cmd.Stderr = io.MultiWriter(stderr)

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := Result{
		Stdout:   strings.TrimRight(stdout.String(), "\n"),
		Stderr:   strings.TrimRight(stderr.String(), "\n"),
		ExitCode: exitCode(err),
	}

	slog.Info("ssh.command_executed",
		"host", cfg.Host,
		"command", FormatCommandForLog(cfg, command),
		"exit_code", result.ExitCode,
		"duration_ms", duration.Milliseconds(),
	)

	if stdout.truncated || stderr.truncated {
		slog.Warn("ssh.output_truncated", "host", cfg.Host, "bytes_limit", maxOut)
	}

	if err != nil {
		return result, newCommandError(err, result, cfg.Host)
	}
	return result, nil
}

// Run executes an SSH command and returns the result.
func (ExecRunner) Run(ctx context.Context, cfg Config, command string, opts RunOptions) (Result, error) {
	return runSSH(ctx, cfg, command, opts)
}

// ExecSCPRunner implements SCPRunner using os/exec.
type ExecSCPRunner struct{}

// Copy transfers a local file to a remote path via SCP.
func (ExecSCPRunner) Copy(ctx context.Context, cfg Config, localPath, remotePath string, opts SCPOptions) error {
	timeout := opts.TransferTimeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	copyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := buildSCPArgs(cfg, opts)
	host := cfg.Host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	target := fmt.Sprintf("%s@%s:%s", cfg.User, host, remotePath)
	args = append(args, localPath, target)

	maxOut := opts.MaxOutputBytes
	if maxOut == 0 {
		maxOut = 512 * 1024
	}

	cmd := exec.CommandContext(copyCtx, "scp", args...)
	stdout := newBoundedBuffer(maxOut)
	stderr := newBoundedBuffer(maxOut)
	cmd.Stdout = io.MultiWriter(stdout)
	cmd.Stderr = io.MultiWriter(stderr)

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	slog.Info("ssh.scp_transfer",
		"host", cfg.Host,
		"remote_path", remotePath,
		"duration_ms", duration.Milliseconds(),
	)

	if err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		combined := stderrStr + " " + err.Error()
		classified := ClassifyError(combined, cfg.Host, stderrStr)
		exitC := exitCode(err)

		if !errors.Is(classified, ErrCommand) {
			classified.ExitCode = exitC
			return classified
		}

		return &SSHError{
			Category:  ErrCommand,
			Message:   fmt.Sprintf("scp failed: %v", err),
			Hint:      stderrStr,
			Retryable: false,
			ExitCode:  exitC,
			Host:      cfg.Host,
		}
	}
	return nil
}

type boundedBuffer struct {
	buf       bytes.Buffer
	remaining int
	truncated bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	return &boundedBuffer{remaining: limit}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > b.remaining {
		b.truncated = true
		p = p[:b.remaining]
	}
	_, _ = b.buf.Write(p)
	b.remaining -= len(p)
	return len(p), nil
}

func (b *boundedBuffer) String() string {
	if !b.truncated {
		return b.buf.String()
	}
	if b.buf.Len() == 0 {
		return "[output truncated]"
	}
	return b.buf.String() + "\n[output truncated]"
}
