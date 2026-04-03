package execution

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) runScenarioRestartAndHealth(ctx context.Context, executionID string, scenarioName string) error {
	if err := s.markFinalizationPhase(executionID, FinalizationPhaseRestarting); err != nil {
		return err
	}

	if s.scenarioLifecycle == nil || s.scenarioHealth == nil {
		return s.failFinalization(executionID, scenarioName, "scenario restart/health seams are not configured")
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

		restartErr := s.scenarioLifecycle.Restart(ctx, scenarioName)
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

		if err := s.markFinalizationPhase(executionID, FinalizationPhaseHealthCheck); err != nil {
			return err
		}
		healthSnapshot, healthErr := s.waitForScenarioHealth(ctx, scenarioName)
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

		warningCode := finalizationWarningHealthRetry
		if !healthSnapshot.SchemaValid {
			warningCode = finalizationWarningHealthSchemaInvalid
		} else if strings.Contains(strings.ToLower(healthSnapshot.Details), "no health checks") {
			warningCode = finalizationWarningHealthChecksMissing
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
