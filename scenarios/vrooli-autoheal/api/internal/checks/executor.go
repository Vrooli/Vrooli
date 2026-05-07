// Package checks provides health check infrastructure
// [REQ:TEST-SEAM-001] Testing seams for external dependencies
package checks

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vrooli-autoheal/internal/reporoot"
)

// CommandExecutor abstracts command execution for testability.
// This interface allows health checks to be unit tested without
// actually executing shell commands.
type CommandExecutor interface {
	// Output runs the command and returns its stdout.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	// CombinedOutput runs the command and returns combined stdout/stderr.
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
	// Run executes the command without capturing output.
	Run(ctx context.Context, name string, args ...string) error
}

// RealExecutor is the production implementation of CommandExecutor.
// It delegates to os/exec for actual command execution.
type RealExecutor struct{}

// vrooliInvocation prepends --no-stale-check to args when invoking the vrooli
// CLI from autoheal. Autoheal runs as a child of the user's working tree, so
// the embedded fingerprint frequently differs from current sources; without
// this flag every check would enter buildinfo.RebuildAndReexec and contend on
// .vrooli/build/vrooli with sibling autoheal subprocesses, tripping the
// rebuild-loop guard. Idempotent.
func vrooliInvocation(args []string) []string {
	if len(args) > 0 && args[0] == "--no-stale-check" {
		return args
	}
	out := make([]string, 0, len(args)+1)
	out = append(out, "--no-stale-check")
	out = append(out, args...)
	return out
}

func resolveCommandArgs(name string, args []string) []string {
	if strings.TrimSpace(name) == "vrooli" {
		return vrooliInvocation(args)
	}
	return args
}

func resolveCommandPath(name string) string {
	if strings.TrimSpace(name) != "vrooli" {
		return name
	}
	if override := strings.TrimSpace(os.Getenv("VROOLI_CMD_PATH")); override != "" {
		return override
	}
	root := reporoot.ResolveFromOS()
	if root == "" {
		return name
	}
	for _, candidate := range []string{
		filepath.Join(root, ".vrooli", "build", "vrooli"),
		filepath.Join(root, ".vrooli", "build", "vrooli.exe"),
		filepath.Join(root, "vrooli"),
		filepath.Join(root, "vrooli.exe"),
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return name
}

// Output runs the command and returns stdout.
func (e *RealExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, resolveCommandPath(name), resolveCommandArgs(name, args)...).Output()
}

// CombinedOutput runs the command and returns combined stdout/stderr.
func (e *RealExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, resolveCommandPath(name), resolveCommandArgs(name, args)...).CombinedOutput()
}

// Run executes the command without capturing output.
func (e *RealExecutor) Run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, resolveCommandPath(name), resolveCommandArgs(name, args)...).Run()
}

// DefaultExecutor is the global executor instance used when none is injected.
var DefaultExecutor CommandExecutor = &RealExecutor{}

// HTTPDoer abstracts HTTP request execution for testability.
// This interface allows health checks to be unit tested without
// actually making HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient is the default HTTP client used when none is injected.
var DefaultHTTPClient HTTPDoer = &http.Client{Timeout: 10 * time.Second}

// NetworkDialer abstracts network dialing for testability.
// This interface allows health checks to be unit tested without
// actually making network connections.
type NetworkDialer interface {
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
}

// RealDialer is the production implementation of NetworkDialer.
type RealDialer struct{}

// DialTimeout establishes a network connection with a timeout.
func (d *RealDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, address, timeout)
}

// DefaultDialer is the global dialer instance used when none is injected.
var DefaultDialer NetworkDialer = &RealDialer{}
