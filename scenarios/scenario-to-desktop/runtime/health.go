package bundleruntime

import (
	"context"

	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/manifest"
)

// waitForReadiness waits for a service to become ready.
// Delegates to the injected HealthChecker.
func (s *Supervisor) waitForReadiness(ctx context.Context, svc manifest.Service) error {
	return s.healthChecker.WaitForReadiness(ctx, svc.ID)
}

// waitForDependencies waits for all service dependencies to be ready.
// This is kept on Supervisor as it accesses service status directly.
func (s *Supervisor) waitForDependencies(ctx context.Context, svc *manifest.Service) error {
	if len(svc.Dependencies) == 0 {
		return nil
	}

	// Use the HealthChecker's implementation if it's a health.Monitor.
	if hm, ok := s.healthChecker.(*health.Monitor); ok {
		return hm.WaitForDependencies(ctx, svc)
	}

	// Fallback implementation for custom HealthCheckers.
	// This shouldn't normally be reached with the default setup.
	return s.healthChecker.WaitForReadiness(ctx, svc.ID)
}
