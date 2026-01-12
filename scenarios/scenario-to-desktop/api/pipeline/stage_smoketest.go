package pipeline

import (
	"context"
	"fmt"
	"time"

	"scenario-to-desktop-api/smoketest"
)

// SmokeTestStage implements the smoke test stage of the pipeline.
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
	return input.Config.SkipSmokeTest
}

// Execute runs the smoke test stage.
func (s *SmokeTestStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := &StageResult{
		Stage:     s.Name(),
		Status:    StatusRunning,
		StartedAt: s.timeProvider.Now(),
		Logs:      []string{},
	}

	// Check if stage should be skipped
	if s.CanSkip(input) {
		result.Status = StatusSkipped
		result.CompletedAt = s.timeProvider.Now()
		result.Logs = append(result.Logs, "Skipping smoke test: explicitly skipped via config")
		return result
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		result.Status = StatusCancelled
		result.CompletedAt = s.timeProvider.Now()
		result.Error = "stage cancelled"
		return result
	default:
	}

	// Check for service
	if s.service == nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = "smoke test service not configured"
		return result
	}

	// Get build result from previous stage
	if input.BuildResult == nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = "build result not available from previous stage"
		return result
	}

	// Find an artifact to test (prefer current platform)
	currentPlatform := s.service.CurrentPlatform()
	artifactPath := ""

	// First try current platform
	if platResult, ok := input.BuildResult.PlatformResults[currentPlatform]; ok && platResult.Status == "ready" {
		artifactPath = platResult.Artifact
	}

	// If no current platform artifact, try any available
	if artifactPath == "" {
		for _, platResult := range input.BuildResult.PlatformResults {
			if platResult.Status == "ready" && platResult.Artifact != "" {
				artifactPath = platResult.Artifact
				currentPlatform = platResult.Platform
				break
			}
		}
	}

	if artifactPath == "" {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = "no built artifacts available for smoke testing"
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

	// Start the async smoke test
	go s.service.PerformSmokeTest(ctx, smokeTestID, scenarioName, artifactPath, currentPlatform)

	// Wait for smoke test to complete
	smokeStatus, err := s.waitForSmokeTest(ctx, smokeTestID)
	if err != nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = err.Error()
		return result
	}

	// Check smoke test result
	switch smokeStatus.Status {
	case "passed":
		result.Status = StatusCompleted
		result.Logs = append(result.Logs, "Smoke test passed")
		if smokeStatus.TelemetryUploaded {
			result.Logs = append(result.Logs, "Telemetry uploaded successfully")
		}
	case "failed":
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = smokeStatus.Error
		if result.Error == "" {
			result.Error = "smoke test failed"
		}
		result.Details = smokeStatus
		return result
	default:
		// Unexpected status
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = fmt.Sprintf("unexpected smoke test status: %s", smokeStatus.Status)
		result.Details = smokeStatus
		return result
	}

	// Update input with result
	input.SmokeTestResult = smokeStatus

	result.CompletedAt = s.timeProvider.Now()
	result.Details = smokeStatus

	// Add logs from the smoke test
	if len(smokeStatus.Logs) > 0 {
		result.Logs = append(result.Logs, "Smoke test logs:")
		for _, log := range smokeStatus.Logs {
			result.Logs = append(result.Logs, "  "+log)
		}
	}

	return result
}

// waitForSmokeTest polls for smoke test completion.
func (s *SmokeTestStage) waitForSmokeTest(ctx context.Context, smokeTestID string) (*smoketest.Status, error) {
	if s.store == nil {
		return nil, fmt.Errorf("smoke test store not configured for status polling")
	}

	// Poll with timeout
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("smoke test cancelled")
		case <-timeout:
			return nil, fmt.Errorf("smoke test timed out after 2 minutes")
		case <-ticker.C:
			status, ok := s.store.Get(smokeTestID)
			if !ok {
				// Smoke test not yet registered, keep waiting
				continue
			}

			switch status.Status {
			case "passed", "failed":
				return status, nil
			}
			// Still running, continue polling
		}
	}
}
