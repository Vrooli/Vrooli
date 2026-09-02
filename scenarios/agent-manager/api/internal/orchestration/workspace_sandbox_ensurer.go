// This file ensures workspaces satisfy sandbox prerequisites before a run starts.
package orchestration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"github.com/vrooli/envkit-go"
)

type sandboxAvailabilityChecker interface {
	IsAvailable(ctx context.Context) (bool, string)
}

type ensureCall struct {
	done chan struct{}
	err  error
}

// CommandWorkspaceSandboxEnsurer makes workspace-sandbox available at run
// time by delegating startup to Vrooli lifecycle. It deliberately does not
// start workspace-sandbox's API binary directly; lifecycle owns process
// naming, port claims, logs, and cross-process scenario locks.
type CommandWorkspaceSandboxEnsurer struct {
	provider sandboxAvailabilityChecker
	levers   agentconfig.SandboxLevers
	command  []string

	mu       sync.Mutex
	inFlight *ensureCall
}

// NewCommandWorkspaceSandboxEnsurer builds the production run-time ensurer.
func NewCommandWorkspaceSandboxEnsurer(provider sandboxAvailabilityChecker, levers agentconfig.SandboxLevers) *CommandWorkspaceSandboxEnsurer {
	if levers.EnsureStartTimeout <= 0 {
		levers = agentconfig.DefaultLevers().Sandbox
	}
	return &CommandWorkspaceSandboxEnsurer{
		provider: provider,
		levers:   levers,
		command:  []string{"vrooli", "--no-stale-check", "scenario", "start", "workspace-sandbox"},
	}
}

// EnsureAvailable returns when workspace-sandbox reports healthy or when the
// bounded lifecycle start/poll sequence fails.
func (e *CommandWorkspaceSandboxEnsurer) EnsureAvailable(ctx context.Context) error {
	if e == nil || e.provider == nil {
		return domain.NewConfigMissingError("workspaceSandboxEnsurer", "provider not configured", nil)
	}
	if ok, _ := e.check(ctx); ok {
		return nil
	}

	call, owner := e.joinOrStart()
	if owner {
		e.runEnsure(ctx, call)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-call.done:
		return call.err
	}
}

func (e *CommandWorkspaceSandboxEnsurer) joinOrStart() (*ensureCall, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.inFlight != nil {
		return e.inFlight, false
	}
	call := &ensureCall{done: make(chan struct{})}
	e.inFlight = call
	return call, true
}

func (e *CommandWorkspaceSandboxEnsurer) runEnsure(ctx context.Context, call *ensureCall) {
	go func() {
		defer func() {
			e.mu.Lock()
			if e.inFlight == call {
				e.inFlight = nil
			}
			e.mu.Unlock()
			close(call.done)
		}()
		defer obs.RecoverToFailure("workspace sandbox ensure", func(failure obs.PanicFailure) {
			call.err = failure
		})

		timeout := e.levers.EnsureStartTimeout
		if timeout <= 0 {
			timeout = agentconfig.DefaultLevers().Sandbox.EnsureStartTimeout
		}
		ensureCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if err := e.startLifecycle(ensureCtx); err != nil {
			call.err = err
			return
		}
		call.err = e.waitHealthy(ensureCtx)
	}()
}

func (e *CommandWorkspaceSandboxEnsurer) startLifecycle(ctx context.Context) error {
	if len(e.command) == 0 {
		return domain.NewConfigMissingError("workspaceSandboxEnsurer.command", "lifecycle command not configured", nil)
	}
	cmd := exec.CommandContext(ctx, e.command[0], e.command[1:]...)
	cmd.Env = []string(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.ForeignScenario, nil))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return &domain.SandboxError{
			Operation:   "workspace_sandbox_ensure",
			Cause:       fmt.Errorf("lifecycle start failed: %w: %s", err, trimCommandOutput(out.String())),
			IsTransient: true,
			CanRetry:    true,
		}
	}
	return nil
}

func (e *CommandWorkspaceSandboxEnsurer) waitHealthy(ctx context.Context) error {
	ticker := time.NewTicker(e.pollInterval())
	defer ticker.Stop()
	for {
		ok, reason := e.check(ctx)
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return &domain.SandboxError{
				Operation:   "workspace_sandbox_ensure",
				Cause:       fmt.Errorf("workspace-sandbox did not become healthy: %s", reason),
				IsTransient: true,
				CanRetry:    true,
			}
		case <-ticker.C:
		}
	}
}

func (e *CommandWorkspaceSandboxEnsurer) check(ctx context.Context) (bool, string) {
	timeout := e.levers.AvailabilityCheckTimeout
	if timeout <= 0 {
		timeout = agentconfig.DefaultLevers().Sandbox.AvailabilityCheckTimeout
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return e.provider.IsAvailable(checkCtx)
}

func (e *CommandWorkspaceSandboxEnsurer) pollInterval() time.Duration {
	if e.levers.EnsurePollInterval > 0 {
		return e.levers.EnsurePollInterval
	}
	return agentconfig.DefaultLevers().Sandbox.EnsurePollInterval
}

func trimCommandOutput(output string) string {
	output = strings.TrimSpace(output)
	if len(output) <= 500 {
		return output
	}
	return output[:500] + "...(truncated)"
}
