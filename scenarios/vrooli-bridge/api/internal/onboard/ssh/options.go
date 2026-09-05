package ssh

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// RunOptions configures SSH command execution.
type RunOptions struct {
	ConnectTimeout      time.Duration // Default: 5s
	ServerAliveInterval time.Duration // Default: 5s
	ServerAliveCountMax int           // Default: 1
	StrictHostKey       bool          // Default: true
	IdentitiesOnly      bool          // Default: false

	MaxOutputBytes    int // Default: 512 * 1024
	ErrorContextLines int // Default: 50

	CommandTimeout time.Duration // 0 = inherit from ctx

	ControlMaster bool // SSH connection multiplexing
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
		ControlMaster:       runtime.GOOS != "windows",
	}
}

// TestConnectionOptions returns options for connection testing (longer timeout,
// IdentitiesOnly so only the supplied key is offered).
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

// connectTimeoutSecs returns the connect timeout as whole seconds, floor 5.
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
func buildSSHArgs(cfg Config, opts RunOptions) []string {
	return buildArgs(cfg, opts, "-p")
}

// buildSCPArgs assembles the option flags for an scp invocation.
func buildSCPArgs(cfg Config, opts SCPOptions) []string {
	runOpts := RunOptions{ConnectTimeout: opts.ConnectTimeout, StrictHostKey: opts.StrictHostKey}
	return buildArgs(cfg, runOpts, "-P")
}

// buildArgs assembles common SSH/SCP option flags. portFlag is "-p" for ssh and
// "-P" for scp. When cfg.KnownHostsFile is set the invocation is pinned to the
// bridge-owned known_hosts (and the system-wide file is ignored) so system ssh
// shares TOFU state with the x/crypto key-copy path.
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
	if cfg.KnownHostsFile != "" {
		out = append(out,
			"-o", "UserKnownHostsFile="+cfg.KnownHostsFile,
			"-o", "GlobalKnownHostsFile=/dev/null",
		)
	}
	if opts.IdentitiesOnly {
		out = append(out, "-o", "IdentitiesOnly=yes")
	}
	if opts.ControlMaster {
		controlPath := buildControlPath(cfg)
		out = append(out,
			"-o", "ControlMaster=auto",
			"-o", fmt.Sprintf("ControlPath=%s", controlPath),
			"-o", "ControlPersist=60",
		)
	}
	out = append(out, portFlag, strconv.Itoa(cfg.Port))
	if cfg.KeyPath != "" {
		out = append(out, "-i", cfg.KeyPath)
	}
	return out
}

// buildControlPath returns an OS-safe, short, stable control socket path.
func buildControlPath(cfg Config) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s@%s:%d", cfg.User, cfg.Host, cfg.Port)))
	name := "vrooli-bridge-ssh-" + hex.EncodeToString(sum[:8])
	return filepath.ToSlash(filepath.Join(controlPathDir(), name))
}

// controlPathDir picks a short temp directory for SSH control sockets.
func controlPathDir() string {
	if runtime.GOOS != "windows" {
		if info, err := os.Stat("/tmp"); err == nil && info.IsDir() {
			return "/tmp"
		}
	}
	return os.TempDir()
}
