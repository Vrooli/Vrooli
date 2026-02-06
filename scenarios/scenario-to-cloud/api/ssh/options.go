package ssh

import (
	"fmt"
	"strconv"
	"time"
)

// RunOptions configures SSH command execution.
type RunOptions struct {
	// Connection
	ConnectTimeout      time.Duration // Default: 5s
	ServerAliveInterval time.Duration // Default: 5s
	ServerAliveCountMax int           // Default: 1
	StrictHostKey       bool          // Default: true
	IdentitiesOnly      bool          // Default: false

	// Output limits
	MaxOutputBytes    int // Default: 512 * 1024
	ErrorContextLines int // Default: 50

	// Execution (0 = inherit from ctx)
	CommandTimeout time.Duration
}

// DefaultRunOptions returns the standard options for command execution.
func DefaultRunOptions() RunOptions {
	return RunOptions{
		ConnectTimeout:      5 * time.Second,
		ServerAliveInterval: 5 * time.Second,
		ServerAliveCountMax: 1,
		StrictHostKey:       true,
		MaxOutputBytes:      512 * 1024,
		ErrorContextLines:   50,
	}
}

// TestConnectionOptions returns options for connection testing (longer timeout, IdentitiesOnly).
func TestConnectionOptions() RunOptions {
	return RunOptions{
		ConnectTimeout:    10 * time.Second,
		StrictHostKey:     true,
		IdentitiesOnly:    true,
		MaxOutputBytes:    512 * 1024,
		ErrorContextLines: 50,
	}
}

// SCPOptions configures SCP file transfers.
type SCPOptions struct {
	ConnectTimeout  time.Duration // Default: 5s
	StrictHostKey   bool          // Default: true
	TransferTimeout time.Duration // Default: 10min
	MaxOutputBytes  int           // Default: 512 * 1024
}

// DefaultSCPOptions returns the standard options for SCP transfers.
func DefaultSCPOptions() SCPOptions {
	return SCPOptions{
		ConnectTimeout:  5 * time.Second,
		StrictHostKey:   true,
		TransferTimeout: 10 * time.Minute,
		MaxOutputBytes:  512 * 1024,
	}
}

// HandlerOptions configures HTTP handler timeouts.
type HandlerOptions struct {
	TestConnectionTimeout time.Duration // Default: 30s
	CopyKeyTimeout        time.Duration // Default: 30s
}

// DefaultHandlerOptions returns production handler timeouts.
func DefaultHandlerOptions() HandlerOptions {
	return HandlerOptions{
		TestConnectionTimeout: 30 * time.Second,
		CopyKeyTimeout:        30 * time.Second,
	}
}

// connectTimeoutSecs returns the connect timeout as whole seconds, with a floor of 5.
func (o RunOptions) connectTimeoutSecs() int {
	secs := int(o.ConnectTimeout.Seconds())
	if secs == 0 {
		secs = 5
	}
	return secs
}

// maxOutput returns MaxOutputBytes or the default (512KB).
func (o RunOptions) maxOutput() int {
	if o.MaxOutputBytes > 0 {
		return o.MaxOutputBytes
	}
	return 512 * 1024
}

// buildSSHArgs assembles the option flags for an ssh invocation.
// Always includes BatchMode=yes.
func buildSSHArgs(cfg Config, opts RunOptions) []string {
	return buildArgs(cfg, opts, "-p")
}

// buildSCPArgs assembles the option flags for an scp invocation.
// Same as buildSSHArgs but uses -P (uppercase) for port.
func buildSCPArgs(cfg Config, opts SCPOptions) []string {
	runOpts := RunOptions{
		ConnectTimeout: opts.ConnectTimeout,
		StrictHostKey:  opts.StrictHostKey,
	}
	return buildArgs(cfg, runOpts, "-P")
}

// buildArgs assembles common SSH/SCP option flags.
// portFlag is "-p" for ssh and "-P" for scp.
func buildArgs(cfg Config, opts RunOptions, portFlag string) []string {
	timeout := opts.connectTimeoutSecs()

	out := []string{
		"-o", "BatchMode=yes",
		"-o", fmt.Sprintf("ConnectTimeout=%d", timeout),
	}
	if opts.ServerAliveInterval > 0 {
		out = append(out,
			"-o", fmt.Sprintf("ServerAliveInterval=%d", int(opts.ServerAliveInterval.Seconds())),
			"-o", fmt.Sprintf("ServerAliveCountMax=%d", opts.ServerAliveCountMax),
		)
	}
	if opts.StrictHostKey {
		out = append(out, "-o", "StrictHostKeyChecking=accept-new")
	}
	if opts.IdentitiesOnly {
		out = append(out, "-o", "IdentitiesOnly=yes")
	}
	out = append(out, portFlag, strconv.Itoa(cfg.Port))
	if cfg.KeyPath != "" {
		out = append(out, "-i", cfg.KeyPath)
	}
	return out
}

// ---- Legacy adapters for format.go / connect.go display functions ----

// BuildSSHArgs assembles the option flags for an ssh invocation (used by display/format functions).
func BuildSSHArgs(cfg Config, opts RunOptions) []string {
	return buildSSHArgs(cfg, opts)
}

// BuildSCPArgs assembles the option flags for an scp invocation (used by display/format functions).
func BuildSCPArgs(cfg Config, opts SCPOptions) []string {
	return buildSCPArgs(cfg, opts)
}
