// Package checks provides health check infrastructure
// [REQ:TEST-SEAM-001] Testing seams for external dependencies
package checks

import (
	"context"
	"net"
	"net/http"
	"os/exec"
	"time"
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

// Output runs the command and returns stdout.
func (e *RealExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// CombinedOutput runs the command and returns combined stdout/stderr.
func (e *RealExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Run executes the command without capturing output.
func (e *RealExecutor) Run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
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
