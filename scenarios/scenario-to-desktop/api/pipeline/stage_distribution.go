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
	// Skip if:
	// 1. Distribution is not enabled in config
	// 2. No build artifacts available
	if input.Config == nil || !input.Config.Distribute {
		return true
	}
	if input.BuildResult == nil || len(input.BuildResult.Artifacts) == 0 {
		return true
	}
	return false
}

// Execute runs the distribution stage.
func (s *DistributionStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := &StageResult{
		Stage:     s.Name(),
		Status:    StatusRunning,
		StartedAt: s.timeProvider.Now(),
		Logs:      []string{},
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
		result.Error = "distribution service not configured"
		return result
	}

	// Build artifacts map from build result
	artifacts := make(map[string]string)
	if input.BuildResult != nil {
		for platform, platResult := range input.BuildResult.PlatformResults {
			if platResult.Status == "ready" && platResult.Artifact != "" {
				artifacts[platform] = platResult.Artifact
			}
		}
	}

	if len(artifacts) == 0 {
		result.Status = StatusSkipped
		result.CompletedAt = s.timeProvider.Now()
		result.Logs = append(result.Logs, "No artifacts to distribute")
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
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = fmt.Sprintf("distribution failed: %v", err)
		return result
	}

	result.Logs = append(result.Logs, fmt.Sprintf("Distribution started: %s", resp.DistributionID))

	// Wait for distribution to complete
	distStatus, err := s.waitForDistribution(ctx, resp.DistributionID)
	if err != nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = err.Error()
		return result
	}

	// Check distribution result
	switch distStatus.Status {
	case distribution.StatusCompleted:
		result.Status = StatusCompleted
		result.Logs = append(result.Logs, "All targets uploaded successfully")
	case distribution.StatusPartial:
		result.Status = StatusCompleted // Still consider success if some worked
		result.Logs = append(result.Logs, "Distribution completed with some failures")
	case distribution.StatusFailed:
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = distStatus.Error
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

	result.CompletedAt = s.timeProvider.Now()
	result.Details = distStatus

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
		return nil, fmt.Errorf("distribution store not configured for status polling")
	}

	// Poll with timeout (uploads can take time for large files)
	timeout := time.After(30 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("distribution cancelled")
		case <-timeout:
			return nil, fmt.Errorf("distribution timed out after 30 minutes")
		case <-ticker.C:
			status, ok := s.store.Get(distributionID)
			if !ok {
				continue
			}

			switch status.Status {
			case distribution.StatusCompleted, distribution.StatusPartial, distribution.StatusFailed, distribution.StatusCancelled:
				return status, nil
			}
		}
	}
}
