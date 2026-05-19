package phases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
)

func ensureWorkspaceSandboxAvailable(ctx context.Context, deps Deps, run *domain.Run, provider sandbox.Provider) error {
	if provider == nil {
		return domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}

	ok, reason := checkSandboxAvailable(ctx, deps.Levers.Sandbox, provider)
	if ok {
		return nil
	}

	if run != nil {
		EmitSystemEvent(ctx, deps, run.ID, "warn",
			fmt.Sprintf("workspace-sandbox unavailable; attempting lifecycle start: %s", reason))
	}
	if deps.WorkspaceSandbox == nil {
		return workspaceSandboxUnavailableError("create", reason, nil)
	}

	if err := deps.WorkspaceSandbox.EnsureAvailable(ctx); err != nil {
		return workspaceSandboxUnavailableError("create", reason, err)
	}

	ok, reason = checkSandboxAvailable(ctx, deps.Levers.Sandbox, provider)
	if !ok {
		return workspaceSandboxUnavailableError("create", reason, nil)
	}
	if run != nil {
		EmitSystemEvent(ctx, deps, run.ID, "info", "workspace-sandbox is available after lifecycle ensure")
	}
	return nil
}

func checkSandboxAvailable(ctx context.Context, levers config.SandboxLevers, provider sandbox.Provider) (bool, string) {
	timeout := levers.AvailabilityCheckTimeout
	if timeout <= 0 {
		timeout = config.DefaultLevers().Sandbox.AvailabilityCheckTimeout
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return provider.IsAvailable(checkCtx)
}

func workspaceSandboxUnavailableError(operation, reason string, cause error) *domain.SandboxError {
	if reason == "" {
		reason = "workspace-sandbox did not become healthy"
	}
	if cause != nil {
		reason = fmt.Sprintf("%s; ensure failed: %v", reason, cause)
	}
	return &domain.SandboxError{
		Operation:   operation,
		Cause:       fmt.Errorf("workspace-sandbox unavailable: %s", reason),
		IsTransient: true,
		CanRetry:    true,
	}
}

func retryableSandboxError(err error) bool {
	var sandboxErr *domain.SandboxError
	return errors.As(err, &sandboxErr) && sandboxErr.Retryable()
}

func retryBackoff(levers config.SandboxLevers, attempt int) time.Duration {
	delay := levers.OperationInitialBackoff
	if delay <= 0 {
		delay = config.DefaultLevers().Sandbox.OperationInitialBackoff
	}
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	maxDelay := levers.OperationMaxBackoff
	if maxDelay <= 0 {
		maxDelay = config.DefaultLevers().Sandbox.OperationMaxBackoff
	}
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

func waitForSandboxRetry(ctx context.Context, levers config.SandboxLevers, attempt int) error {
	timer := time.NewTimer(retryBackoff(levers, attempt))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
