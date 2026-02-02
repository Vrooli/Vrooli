// DOC: docs/reference/smoke-test-pipeline.md
package pipeline

import (
	"context"
	"fmt"
	"time"

	"scenario-to-desktop-api/shared/errors"
	"scenario-to-desktop-api/smoketest"
)

// SmokeTestStage implements the smoke test stage of the pipeline.
// See docs/reference/smoke-test-pipeline.md for detailed execution flow.
type SmokeTestStage struct {
	service      smoketest.Service
	store        smoketest.Store
	timeProvider TimeProvider
}

// SmokeTestStageOption configures a SmokeTestStage.
type SmokeTestStageOption func(*SmokeTestStage)

// WithSmokeTestService sets the smoke test service.
func WithSmokeTestService(svc smoketest.Service) SmokeTestStageOption {
	return func(s *SmokeTestStage) {
		s.service = svc
	}
}

// WithSmokeTestStore sets the smoke test store for status polling.
func WithSmokeTestStore(store smoketest.Store) SmokeTestStageOption {
	return func(s *SmokeTestStage) {
		s.store = store
	}
}

// WithSmokeTestTimeProvider sets the time provider.
func WithSmokeTestTimeProvider(tp TimeProvider) SmokeTestStageOption {
	return func(s *SmokeTestStage) {
		s.timeProvider = tp
	}
}

// NewSmokeTestStage creates a new smoke test stage.
func NewSmokeTestStage(opts ...SmokeTestStageOption) *SmokeTestStage {
	s := &SmokeTestStage{
		timeProvider: NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the stage name.
func (s *SmokeTestStage) Name() string {
	return StageSmokeTest
}

// Dependencies returns stages that must complete before this one.
func (s *SmokeTestStage) Dependencies() []string {
	return []string{StageBuild}
}

// CanSkip returns whether this stage can be skipped.
func (s *SmokeTestStage) CanSkip(input *StageInput) bool {
	return ShouldSkipSmokeTest(input.Config)
}

// Execute runs the smoke test stage.
func (s *SmokeTestStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := newStageResult(s.Name(), s.timeProvider)

	if s.CanSkip(input) {
		skipStage(result, s.timeProvider, "Skipping smoke test: explicitly skipped via config")
		return result
	}

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}

	if s.service == nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeServiceStartError, "smoke test service not configured").
			WithRecovery(errors.RecoveryContactSupport, "Server configuration issue - contact support").
			WithManualSteps([]string{
				"Check server startup logs for initialization errors",
				"Verify the smoke test service is properly configured",
				"Contact support if the issue persists",
			}).
			InDomain("smoketest"))
		return result
	}

	if input.BuildResult == nil {
		failStage(result, s.timeProvider, errors.New(errors.CodeDependencyError, "build result not available from previous stage").
			WithRecovery(errors.RecoveryRetry, "Ensure build stage completes successfully first").
			WithManualSteps([]string{
				"Check if the build stage completed successfully",
				"Review build stage logs for errors",
				"Restart the pipeline from the build stage",
			}).
			InDomain("smoketest"))
		return result
	}

	// Find an artifact to test (prefer current platform)
	currentPlatform := s.service.CurrentPlatform()
	artifactPath := ""

	// First try current platform using artifact helper
	artifactPath = FindArtifactForPlatform(input.BuildResult.PlatformResults, currentPlatform)

	// If no current platform artifact, try any available
	if artifactPath == "" {
		currentPlatform, artifactPath = FindFirstReadyArtifact(input.BuildResult.PlatformResults)
	}

	if artifactPath == "" {
		failStage(result, s.timeProvider, errors.New(errors.CodeArtifactNotFound, "no built artifacts available for smoke testing").
			WithRecovery(errors.RecoveryRetry, "Ensure build stage produces artifacts").
			WithManualSteps([]string{
				"Check if the build stage completed successfully",
				"Verify build produced artifacts for at least one platform",
				"Review build logs for errors",
			}).
			InDomain("smoketest"))
		return result
	}

	scenarioName := input.Config.ScenarioName
	result.Logs = append(result.Logs,
		fmt.Sprintf("Running smoke test for: %s", scenarioName),
		fmt.Sprintf("Platform: %s", currentPlatform),
		fmt.Sprintf("Artifact: %s", artifactPath),
	)

	// Generate smoke test ID
	smokeTestID := fmt.Sprintf("smoke-%s-%d", scenarioName, time.Now().UnixMilli())

	// Initialize smoke test status in store BEFORE launching async goroutine.
	// The smoke test service checks if the status exists and exits immediately if not,
	// so we must create it here first.
	if s.store != nil {
		now := time.Unix(s.timeProvider.Now(), 0)
		initialStatus := &smoketest.Status{
			SmokeTestID:  smokeTestID,
			ScenarioName: scenarioName,
			Platform:     currentPlatform,
			Status:       "running",
			ArtifactPath: artifactPath,
			StartedAt:    now,
			Logs:         []string{},
		}
		s.store.Save(initialStatus)
	}

	// Start the async smoke test
	go s.service.PerformSmokeTest(ctx, smokeTestID, scenarioName, artifactPath, currentPlatform)

	// Wait for smoke test to complete
	smokeStatus, waitErr := s.waitForSmokeTest(ctx, smokeTestID)
	if waitErr != nil {
		// Handle wait error based on its kind
		if waitErr.Kind == WaitErrorStore {
			failStage(result, s.timeProvider, errors.New(errors.CodeServiceStartError, "Smoke test tracking service unavailable").
				WithRecovery(errors.RecoveryContactSupport, "Server configuration issue - contact support").
				WithManualSteps([]string{
					"Check server startup logs for initialization errors",
					"Verify the smoke test store is properly configured",
					"Contact support if the issue persists",
				}).
				InDomain("smoketest"))
		} else if waitErr.Kind == WaitErrorTimeout {
			failStage(result, s.timeProvider, errors.ErrSmokeTestTimeout(
				DefaultSmokeTestTimeout.String(),
				map[string]string{"smoke_test_id": smokeTestID},
			))
		} else if waitErr.Kind == WaitErrorCancelled {
			failStage(result, s.timeProvider, errors.ErrSmokeTestCancelled())
		} else {
			failStage(result, s.timeProvider, errors.ErrSmokeTestExecutionFailed(
				waitErr,
				map[string]string{"smoke_test_id": smokeTestID},
			))
		}
		return result
	}

	// Check smoke test result
	switch smokeStatus.Status {
	case SmokeTestStatusPassed:
		result.Logs = append(result.Logs, "Smoke test passed")
		if smokeStatus.TelemetryUploaded {
			result.Logs = append(result.Logs, "Telemetry uploaded successfully")
		}
	case SmokeTestStatusFailed:
		// Create error with context from the smoke test status
		context := map[string]string{
			"smoke_test_id": smokeTestID,
			"platform":      currentPlatform,
		}
		if smokeStatus.ErrorKind != nil {
			context["error_kind"] = smokeStatus.ErrorKind.String()
		}
		failStage(result, s.timeProvider, errors.ErrSmokeTestExecutionFailed(
			fmt.Errorf("%s", smokeStatus.Error),
			context,
		).WithRecovery(errors.RecoveryRetry, smokeStatus.SuggestedAction))
		result.Details = smokeStatus
		return result
	default:
		failStage(result, s.timeProvider, errors.New(errors.CodeSmokeTestFailed, fmt.Sprintf("unexpected smoke test status: %s", smokeStatus.Status)).
			WithDetail("smoke_test_id", smokeTestID).
			WithDetail("status", smokeStatus.Status).
			InDomain("smoketest"))
		result.Details = smokeStatus
		return result
	}

	// Update input with result
	input.SmokeTestResult = smokeStatus

	completeStage(result, s.timeProvider, smokeStatus)

	// Add logs from the smoke test
	if len(smokeStatus.Logs) > 0 {
		result.Logs = append(result.Logs, "Smoke test logs:")
		for _, log := range smokeStatus.Logs {
			result.Logs = append(result.Logs, "  "+log)
		}
	}

	return result
}

// waitForSmokeTest polls for smoke test completion using the generic Poller.
func (s *SmokeTestStage) waitForSmokeTest(ctx context.Context, smokeTestID string) (*smoketest.Status, *WaitError) {
	if s.store == nil {
		return nil, &WaitError{
			Kind:       WaitErrorStore,
			EntityType: "smoke test",
			EntityID:   smokeTestID,
		}
	}

	poller := &Poller[*smoketest.Status]{
		Config: PollerConfig{
			EntityType:   "Smoke test",
			Timeout:      DefaultSmokeTestTimeout,
			PollInterval: DefaultSmokePollInterval,
			LogInterval:  10, // Log every ~20 seconds
		},
		GetStatus: s.store.Get,
		IsComplete: func(status *smoketest.Status) bool {
			switch status.Status {
			case SmokeTestStatusPassed, SmokeTestStatusFailed:
				return true
			}
			return false
		},
	}

	return poller.Wait(ctx, smokeTestID)
}
