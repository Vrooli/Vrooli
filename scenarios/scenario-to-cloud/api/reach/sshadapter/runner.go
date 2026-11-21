package sshadapter

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ConnectionConfig is the resolved SSH locator plus the operator-held key
// reference the credential binding names. KeyPath empty means the operator's
// ambient identity (agent, default identity files) authenticates.
type ConnectionConfig struct {
	Host           string
	Port           int
	User           string
	KeyPath        string
	KnownHostsFile string
}

// Connection defaults for a locator that does not state them.
const (
	DefaultPort = 22
	DefaultUser = "root"
)

// NewConfig applies the locator defaults and the scenario's known_hosts store.
func NewConfig(host string, port int, user, keyPath string) ConnectionConfig {
	if port == 0 {
		port = DefaultPort
	}
	if user == "" {
		user = DefaultUser
	}
	return ConnectionConfig{
		Host:           host,
		Port:           port,
		User:           user,
		KeyPath:        ExpandPath(strings.TrimSpace(keyPath)),
		KnownHostsFile: KnownHostsFile(),
	}
}

// Result is what one ssh invocation produced.
type Result struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// RunOptions bounds one ssh invocation.
type RunOptions struct {
	ConnectTimeout      time.Duration
	ServerAliveInterval time.Duration
	ServerAliveCountMax int
	StrictHostKey       bool
	IdentitiesOnly      bool
	// Stdin is fed to the remote command. A value that must reach a remote
	// process travels here and never inside the command string, which is
	// visible in both local and remote process listings.
	Stdin          []byte
	MaxOutputBytes int
	CommandTimeout time.Duration
	ControlMaster  bool
}

// DefaultRunOptions is the transport's standard bound.
func DefaultRunOptions() RunOptions {
	return RunOptions{
		ConnectTimeout:      5 * time.Second,
		ServerAliveInterval: 5 * time.Second,
		ServerAliveCountMax: 1,
		StrictHostKey:       true,
		MaxOutputBytes:      512 * 1024,
		ControlMaster:       runtime.GOOS != "windows",
	}
}

// SCPOptions bounds one scp transfer.
type SCPOptions struct {
	ConnectTimeout  time.Duration
	StrictHostKey   bool
	TransferTimeout time.Duration
	MaxOutputBytes  int
}

// DefaultSCPOptions is the transport's standard transfer bound.
func DefaultSCPOptions() SCPOptions {
	return SCPOptions{
		ConnectTimeout:  5 * time.Second,
		StrictHostKey:   true,
		TransferTimeout: 10 * time.Minute,
		MaxOutputBytes:  512 * 1024,
	}
}

// Runner executes one remote string over ssh. It is the adapter's only
// execution seam; tests substitute an in-memory runner.
type Runner interface {
	Run(ctx context.Context, cfg ConnectionConfig, command string, opts RunOptions) (Result, error)
}

// SCPRunner places one local file on the target.
type SCPRunner interface {
	Copy(ctx context.Context, cfg ConnectionConfig, localPath, remotePath string, opts SCPOptions) error
}

// ExecRunner runs the local ssh binary.
type ExecRunner struct{}

// ExecSCPRunner runs the local scp binary.
type ExecSCPRunner struct{}

var (
	_ Runner    = ExecRunner{}
	_ SCPRunner = ExecSCPRunner{}
)

// exitCode is the process exit code, or 255 (the ssh convention for a
// connection-level failure) when the process did not report one.
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

// Run executes the remote string. The error, when set, wraps the process
// failure; the caller classifies by Result.ExitCode, never by message text.
func (ExecRunner) Run(ctx context.Context, cfg ConnectionConfig, command string, opts RunOptions) (Result, error) {
	if opts.CommandTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.CommandTimeout)
		defer cancel()
	}
	args := buildArgs(cfg, opts, "-p")
	args = append(args, fmt.Sprintf("%s@%s", cfg.User, cfg.Host), "--", command)

	maxOut := opts.MaxOutputBytes
	if maxOut <= 0 {
		maxOut = 512 * 1024
	}
	cmd := exec.CommandContext(ctx, "ssh", args...)
	if len(opts.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(opts.Stdin)
	}
	stdout := newBoundedBuffer(maxOut)
	stderr := newBoundedBuffer(maxOut)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	start := time.Now()
	err := cmd.Run()
	result := Result{
		Stdout:   strings.TrimRight(stdout.String(), "\n"),
		Stderr:   strings.TrimRight(stderr.String(), "\n"),
		ExitCode: exitCode(err),
	}
	slog.Info("ssh.command_executed",
		"host", cfg.Host,
		"command", formatForLog(cfg, command),
		"exit_code", result.ExitCode,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	if stdout.truncated || stderr.truncated {
		slog.Warn("ssh.output_truncated", "host", cfg.Host, "bytes_limit", maxOut)
	}
	if err != nil {
		if ctx.Err() != nil {
			return result, fmt.Errorf("ssh %s: %w", cfg.Host, ctx.Err())
		}
		return result, fmt.Errorf("ssh %s exited %d: %w", cfg.Host, result.ExitCode, err)
	}
	return result, nil
}

// Copy transfers one file with scp.
func (ExecSCPRunner) Copy(ctx context.Context, cfg ConnectionConfig, localPath, remotePath string, opts SCPOptions) error {
	timeout := opts.TransferTimeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	copyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := buildArgs(cfg, RunOptions{ConnectTimeout: opts.ConnectTimeout, StrictHostKey: opts.StrictHostKey}, "-P")
	host := cfg.Host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	args = append(args, localPath, fmt.Sprintf("%s@%s:%s", cfg.User, host, remotePath))

	maxOut := opts.MaxOutputBytes
	if maxOut <= 0 {
		maxOut = 512 * 1024
	}
	cmd := exec.CommandContext(copyCtx, "scp", args...)
	stdout := newBoundedBuffer(maxOut)
	stderr := newBoundedBuffer(maxOut)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	start := time.Now()
	err := cmd.Run()
	slog.Info("ssh.scp_transfer", "host", cfg.Host, "remote_path", remotePath, "duration_ms", time.Since(start).Milliseconds())
	if err != nil {
		return fmt.Errorf("scp to %s exited %d: %s: %w", cfg.Host, exitCode(err), strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

// buildArgs assembles the option flags shared by ssh (-p) and scp (-P).
// BatchMode is always on: the transport never prompts.
func buildArgs(cfg ConnectionConfig, opts RunOptions, portFlag string) []string {
	timeout := int(opts.ConnectTimeout.Seconds())
	if timeout == 0 {
		timeout = 5
	}
	out := []string{"-o", "BatchMode=yes", "-o", fmt.Sprintf("ConnectTimeout=%d", timeout)}
	if opts.ServerAliveInterval > 0 {
		out = append(out,
			"-o", fmt.Sprintf("ServerAliveInterval=%d", int(opts.ServerAliveInterval.Seconds())),
			"-o", fmt.Sprintf("ServerAliveCountMax=%d", opts.ServerAliveCountMax),
		)
	}
	if opts.StrictHostKey {
		out = append(out, "-o", "StrictHostKeyChecking=accept-new")
	}
	if cfg.KnownHostsFile != "" {
		out = append(out, "-o", "UserKnownHostsFile="+cfg.KnownHostsFile, "-o", "GlobalKnownHostsFile=/dev/null")
	}
	if opts.IdentitiesOnly {
		out = append(out, "-o", "IdentitiesOnly=yes")
	}
	if opts.ControlMaster {
		out = append(out, "-o", "ControlMaster=auto", "-o", "ControlPath="+controlPath(cfg), "-o", "ControlPersist=60")
	}
	port := cfg.Port
	if port == 0 {
		port = DefaultPort
	}
	out = append(out, portFlag, strconv.Itoa(port))
	if cfg.KeyPath != "" {
		out = append(out, "-i", cfg.KeyPath)
	}
	return out
}

// controlPath is a short, stable multiplexing socket path per locator.
func controlPath(cfg ConnectionConfig) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s@%s:%d", cfg.User, cfg.Host, cfg.Port)))
	dir := os.TempDir()
	if runtime.GOOS != "windows" {
		if info, err := os.Stat("/tmp"); err == nil && info.IsDir() {
			dir = "/tmp"
		}
	}
	return filepath.ToSlash(filepath.Join(dir, "vrooli-ssh-"+hex.EncodeToString(sum[:8])))
}

// formatForLog renders the invocation for the structured log with the key
// path redacted.
func formatForLog(cfg ConnectionConfig, command string) string {
	redacted := cfg
	redacted.KeyPath = ""
	parts := append([]string{"ssh"}, buildArgs(redacted, DefaultRunOptions(), "-p")...)
	if cfg.KeyPath != "" {
		parts = append(parts, "-i", "<redacted>")
	}
	parts = append(parts, fmt.Sprintf("%s@%s", cfg.User, cfg.Host), "--", command)
	return strings.Join(parts, " ")
}

// ExpandPath expands a leading ~/ to the operator's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

type boundedBuffer struct {
	buf       bytes.Buffer
	remaining int
	truncated bool
}

func newBoundedBuffer(limit int) *boundedBuffer { return &boundedBuffer{remaining: limit} }

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

var _ io.Writer = (*boundedBuffer)(nil)
