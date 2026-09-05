package execution

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) runScenarioRestartAndHealth(ctx context.Context, executionID string, scenarioName string, target string) error {
	// NOTE: The overall finalization phase (restarting) is set once before the
	// scenario loop in processFinalization. Per-scenario restart/health updates
	// only update the scenario-level state to avoid phase flickering when
	// processing multiple scenarios.
	//
	// scenarioName keys the per-scenario finalization state; target is the
	// lifecycle address actually restarted/health-checked. They differ when the
	// scenario is shadow-engaged: target = "<scenario>@shadow" routes validation
	// to the instance running the just-merged candidate (plan P-b.5), while state
	// stays keyed by the bare scenario name.
	if s.scenarioLifecycle == nil || s.scenarioHealth == nil {
		return s.failFinalization(executionID, scenarioName, "scenario restart/health seams are not configured")
	}
	if strings.TrimSpace(target) == "" {
		target = scenarioName
	}

	for attempt := 1; attempt <= s.finalizationCfg.MaxRestartAttempts; attempt++ {
		restartStartedAt := nowRFC3339()
		if err := s.updateScenarioRestartState(executionID, scenarioName, RestartResult{
			Status:    FinalizationStatusRunning,
			Attempts:  attempt,
			StartedAt: restartStartedAt,
		}); err != nil {
			return err
		}

		restartErr := s.scenarioLifecycle.Restart(ctx, target)
		restartFinishedAt := nowRFC3339()
		if restartErr != nil {
			restartResult := RestartResult{
				Status:     FinalizationStatusFailed,
				Attempts:   attempt,
				LastError:  restartErr.Error(),
				StartedAt:  restartStartedAt,
				FinishedAt: restartFinishedAt,
			}
			if err := s.updateScenarioRestartState(executionID, scenarioName, restartResult); err != nil {
				return err
			}
			if attempt < s.finalizationCfg.MaxRestartAttempts {
				if err := s.appendFinalizationWarning(executionID, newFinalizationWarning(
					finalizationWarningRestartRetry,
					scenarioName,
					fmt.Sprintf("restart attempt %d failed: %v; retrying once", attempt, restartErr),
					true,
				)); err != nil {
					return err
				}
				continue
			}
			if err := s.updateScenarioHealthState(executionID, scenarioName, HealthCheckResult{
				Status:      FinalizationStatusFailed,
				SchemaValid: false,
				Details:     "restart command failed; health check skipped",
				CheckedAt:   restartFinishedAt,
			}); err != nil {
				return err
			}
			return nil
		}

		if err := s.updateScenarioRestartState(executionID, scenarioName, RestartResult{
			Status:     FinalizationStatusCompleted,
			Attempts:   attempt,
			StartedAt:  restartStartedAt,
			FinishedAt: restartFinishedAt,
		}); err != nil {
			return err
		}

		healthSnapshot, healthErr := s.waitForScenarioHealth(ctx, target)
		if healthErr == nil {
			return s.updateScenarioHealthState(executionID, scenarioName, HealthCheckResult{
				Status:         FinalizationStatusCompleted,
				ScenarioStatus: healthSnapshot.ScenarioStatus,
				HealthStatus:   healthSnapshot.HealthStatus,
				SchemaValid:    healthSnapshot.SchemaValid,
				Details:        healthSnapshot.Details,
				CheckedAt:      healthSnapshot.CheckedAt,
			})
		}

		if err := s.updateScenarioHealthState(executionID, scenarioName, HealthCheckResult{
			Status:         FinalizationStatusFailed,
			ScenarioStatus: healthSnapshot.ScenarioStatus,
			HealthStatus:   healthSnapshot.HealthStatus,
			SchemaValid:    healthSnapshot.SchemaValid,
			Details:        healthSnapshot.Details,
			CheckedAt:      healthSnapshot.CheckedAt,
		}); err != nil {
			return err
		}

		// SchemaValid is false only when the scenario status probe itself
		// errored (the CLI command failed), so it stands in for "health could
		// not be assessed". A typed probe can no longer report a parse/shape
		// mismatch, so the former "health checks missing" code is gone.
		warningCode := finalizationWarningHealthRetry
		if !healthSnapshot.SchemaValid {
			warningCode = finalizationWarningHealthSchemaInvalid
		}
		if attempt < s.finalizationCfg.MaxRestartAttempts {
			if err := s.appendFinalizationWarning(executionID, newFinalizationWarning(
				warningCode,
				scenarioName,
				fmt.Sprintf("health check failed after restart attempt %d: %s; retrying once", attempt, healthSnapshot.Details),
				true,
			)); err != nil {
				return err
			}
			continue
		}
		return nil
	}

	return nil
}

func (s *Service) waitForScenarioHealth(ctx context.Context, scenarioName string) (ScenarioHealthSnapshot, error) {
	var last ScenarioHealthSnapshot
	deadline := time.Now().Add(s.finalizationCfg.HealthPollTimeout)
	for {
		snapshot, err := s.scenarioHealth.Check(ctx, scenarioName)
		if err == nil {
			last = snapshot
			if snapshot.Healthy {
				return snapshot, nil
			}
		} else {
			last = ScenarioHealthSnapshot{
				SchemaValid: false,
				Healthy:     false,
				Details:     err.Error(),
				CheckedAt:   nowRFC3339(),
			}
		}

		if time.Now().After(deadline) {
			if strings.TrimSpace(last.Details) == "" {
				last.Details = "scenario did not become healthy before timeout"
				last.CheckedAt = nowRFC3339()
			}
			return last, fmt.Errorf("%s", last.Details)
		}

		select {
		case <-ctx.Done():
			last.Details = ctx.Err().Error()
			last.CheckedAt = nowRFC3339()
			return last, ctx.Err()
		case <-time.After(s.finalizationCfg.HealthPollInterval):
		}
	}
}
