// Package checks provides health check infrastructure
// [REQ:TEST-SEAM-001] Testing seams for external dependencies
package checks

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/elevation"
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

// RunPrivilegedService delegates service recovery to the typed elevation
// broker. The caller receives the broker outcome even when the command was
// refused, allowing the API to show the exact operator command.
func RunPrivilegedService(ctx context.Context, executor CommandExecutor, action elevation.Action, unit string) ([]byte, elevation.Outcome, error) {
	outcome, output, err := elevation.Run(ctx, executor, action, unit)
	if err == nil && outcome.State != elevation.Granted {
		err = fmt.Errorf("service action refused: %s", outcome.Reason)
	}
	return output, outcome, err
}

// RunAuthorizedService is the compact form for checks whose existing action
// contract only needs output and error. The elevation decision is still typed
// and centralized; callers that expose action metadata should use
// RunPrivilegedService to retain the full outcome.
func RunAuthorizedService(ctx context.Context, executor CommandExecutor, verb, unit string) ([]byte, error) {
	output, _, err := RunAuthorizedServiceWithOutcome(ctx, executor, verb, unit)
	return output, err
}

// RunAuthorizedServiceWithOutcome is the action-facing form. It preserves the
// typed elevation state so API callers can distinguish granted recovery from a
// setup-required or unsupported action.
func RunAuthorizedServiceWithOutcome(ctx context.Context, executor CommandExecutor, verb, unit string) ([]byte, elevation.Outcome, error) {
	var action elevation.Action
	switch verb {
	case "start":
		action = elevation.ServiceStart
	case "restart":
		action = elevation.ServiceRestart
	case "stop":
		action = elevation.ServiceStop
	default:
		return nil, elevation.Outcome{State: elevation.Refused, Reason: fmt.Sprintf("service action %q requires an operator-supported native backend", verb)}, fmt.Errorf("service action %q requires an operator-supported native backend", verb)
	}
	return RunPrivilegedService(ctx, executor, action, unit)
}

// RealExecutor is the production implementation of CommandExecutor.
// It delegates to os/exec for actual command execution.
type RealExecutor struct{}

// vrooliBinaryOverrideEnv is a retired override kept as the explicit path
// for callers that already ship it. Retired 2026-09-02 with the move onto
// cli-invoke-go; remove after 2026-12-01. New callers set VROOLI_BIN, which
// every invoker honors.
const vrooliBinaryOverrideEnv = "VROOLI_CMD_PATH"

// resolveVrooliBinary finds the control-plane CLI through the one invoker
// seam. There is no newest-mtime selection between repo candidates: freshness
// is the lifecycle engine's job, not the caller's.
func resolveVrooliBinary() string {
	home, _ := os.UserHomeDir()
	path, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{
		Explicit:    strings.TrimSpace(os.Getenv(vrooliBinaryOverrideEnv)),
		RuntimeHome: home,
	})
	if err != nil {
		return "vrooli"
	}
	return path
}

func isVrooli(name string) bool { return strings.TrimSpace(name) == "vrooli" }

// Output runs the command and returns stdout.
func (e *RealExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if isVrooli(name) {
		res := cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: resolveVrooliBinary(), Args: args})
		return res.Stdout, res.Error()
	}
	return exec.CommandContext(ctx, name, args...).Output()
}

// CombinedOutput runs the command and returns combined stdout/stderr.
func (e *RealExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	if isVrooli(name) {
		var combined bytes.Buffer
		res := cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: resolveVrooliBinary(), Args: args, Stdout: &combined, Stderr: &combined})
		return combined.Bytes(), res.Error()
	}
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Run executes the command without capturing output.
func (e *RealExecutor) Run(ctx context.Context, name string, args ...string) error {
	if isVrooli(name) {
		return cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: resolveVrooliBinary(), Args: args}).Error()
	}
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
