package pipeline

import (
	"context"
	"fmt"
	"time"

	"scenario-to-desktop-api/distribution"
)

// DistributionStage implements the distribution stage of the pipeline.
type DistributionStage struct {
	service      distribution.Service
	store        distribution.Store
	timeProvider TimeProvider
}

// DistributionStageOption configures a DistributionStage.
type DistributionStageOption func(*DistributionStage)

// WithDistributionService sets the distribution service.
func WithDistributionService(svc distribution.Service) DistributionStageOption {
	return func(s *DistributionStage) {
		s.service = svc
	}
}

// WithDistributionStore sets the distribution store for status polling.
func WithDistributionStore(store distribution.Store) DistributionStageOption {
	return func(s *DistributionStage) {
		s.store = store
	}
}

// WithDistributionTimeProvider sets the time provider.
func WithDistributionTimeProvider(tp TimeProvider) DistributionStageOption {
	return func(s *DistributionStage) {
		s.timeProvider = tp
	}
}

// NewDistributionStage creates a new distribution stage.
func NewDistributionStage(opts ...DistributionStageOption) *DistributionStage {
	s := &DistributionStage{
		timeProvider: NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the stage name.
func (s *DistributionStage) Name() string {
	return StageDistribution
}

// Dependencies returns stages that must complete before this one.
func (s *DistributionStage) Dependencies() []string {
	return []string{StageBuild}
}

// CanSkip returns whether this stage can be skipped.
func (s *DistributionStage) CanSkip(input *StageInput) bool {
	return ShouldSkipDistribution(input.Config, input.BuildResult)
}

// Execute runs the distribution stage.
func (s *DistributionStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := newStageResult(s.Name(), s.timeProvider)

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}

	if s.service == nil {
		failStage(result, s.timeProvider, "distribution service not configured - this is a server configuration error; check startup logs or contact support")
		return result
	}

	// Build artifacts map from build result using artifact helper
	var artifacts map[string]string
	if input.BuildResult != nil {
		artifacts = GetReadyArtifacts(input.BuildResult.PlatformResults)
	} else {
		artifacts = make(map[string]string)
	}

	if len(artifacts) == 0 {
		skipStage(result, s.timeProvider, "No artifacts to distribute")
		return result
	}

	result.Logs = append(result.Logs,
		fmt.Sprintf("Distributing %d artifacts", len(artifacts)),
		fmt.Sprintf("Scenario: %s", input.Config.ScenarioName),
	)

	if input.Config.Version != "" {
		result.Logs = append(result.Logs, fmt.Sprintf("Version: %s", input.Config.Version))
	}

	// Start distribution
	req := &distribution.DistributeRequest{
		ScenarioName: input.Config.ScenarioName,
		Version:      input.Config.Version,
		Artifacts:    artifacts,
		TargetNames:  input.Config.DistributionTargets,
		Parallel:     true,
	}

	resp, err := s.service.Distribute(ctx, req)
	if err != nil {
		failStage(result, s.timeProvider, fmt.Sprintf("distribution failed: %v", err))
		return result
	}

	result.Logs = append(result.Logs, fmt.Sprintf("Distribution started: %s", resp.DistributionID))

	// Wait for distribution to complete
	distStatus, err := s.waitForDistribution(ctx, resp.DistributionID)
	if err != nil {
		failStage(result, s.timeProvider, err.Error())
		return result
	}

	// Check distribution result
	switch distStatus.Status {
	case distribution.StatusCompleted:
		result.Logs = append(result.Logs, "All targets uploaded successfully")
	case distribution.StatusPartial:
		result.Logs = append(result.Logs, "Distribution completed with some failures")
	case distribution.StatusFailed:
		failStage(result, s.timeProvider, distStatus.Error)
		result.Details = distStatus
		return result
	case distribution.StatusCancelled:
		result.Status = StatusCancelled
		result.CompletedAt = s.timeProvider.Now()
		result.Error = "distribution cancelled"
		return result
	}

	// Store result for next stage (if any)
	input.DistributionResult = distStatus

	completeStage(result, s.timeProvider, distStatus)

	// Log per-target results
	for targetName, targetDist := range distStatus.Targets {
		switch targetDist.Status {
		case distribution.StatusCompleted:
			result.Logs = append(result.Logs, fmt.Sprintf("  %s: uploaded", targetName))
		case distribution.StatusPartial:
			result.Logs = append(result.Logs, fmt.Sprintf("  %s: partial", targetName))
		default:
			result.Logs = append(result.Logs, fmt.Sprintf("  %s: %s", targetName, targetDist.Status))
		}

		// Log individual platform uploads
		for platform, upload := range targetDist.Uploads {
			if upload.URL != "" {
				result.Logs = append(result.Logs, fmt.Sprintf("    %s: %s", platform, upload.URL))
			}
		}
	}

	return result
}

// waitForDistribution polls for distribution completion.
func (s *DistributionStage) waitForDistribution(ctx context.Context, distributionID string) (*distribution.DistributionStatus, error) {
	if s.store == nil {
		return nil, fmt.Errorf("distribution store not configured for status polling - this is a server configuration error; check startup logs or contact support")
	}

	// Poll with timeout (uploads can take time for large files)
	timeout := time.After(DefaultDistributionTimeout)
	ticker := time.NewTicker(DefaultDistributionPollInterval)
	defer ticker.Stop()

	notFoundCount := 0

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("distribution cancelled")
		case <-timeout:
			return nil, fmt.Errorf("distribution timed out after %v", DefaultDistributionTimeout)
		case <-ticker.C:
			status, ok := s.store.Get(distributionID)
			if !ok {
				notFoundCount++
				if notFoundCount%10 == 0 {
					// Log every ~50 seconds (10 polls * 5 second interval) when status not found
					fmt.Printf("Distribution status not yet registered after %d polls, still waiting for %s...\n", notFoundCount, distributionID)
				}
				continue
			}

			switch status.Status {
			case distribution.StatusCompleted, distribution.StatusPartial, distribution.StatusFailed, distribution.StatusCancelled:
				return status, nil
			}
		}
	}
}
