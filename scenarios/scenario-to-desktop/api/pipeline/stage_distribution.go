package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"scenario-to-desktop-api/distribution"
	"scenario-to-desktop-api/shared/errors"
	"scenario-to-desktop-api/updates"
)

// DistributionStage implements the distribution stage of the pipeline.
type DistributionStage struct {
	service           distribution.Service
	store             distribution.Store
	timeProvider      TimeProvider
	manifestGenerator *updates.ManifestGenerator
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

// WithUpdateManifestGenerator sets the manifest generator for auto-update support.
func WithUpdateManifestGenerator(mg *updates.ManifestGenerator) DistributionStageOption {
	return func(s *DistributionStage) {
		s.manifestGenerator = mg
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
		failStage(result, s.timeProvider, errors.ErrDistributionServiceNotConfigured())
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

	// Generate update manifests if configured
	if s.manifestGenerator != nil && input.Config.UpdateConfig != nil {
		manifestResult, err := s.generateUpdateManifests(ctx, input, artifacts)
		if err != nil {
			// Log warning but don't fail - distribution can proceed without manifests
			result.Logs = append(result.Logs,
				fmt.Sprintf("WARNING: Failed to generate update manifests: %v", err))
		} else if manifestResult != nil {
			// Add any warnings to logs
			for _, warning := range manifestResult.Warnings {
				result.Logs = append(result.Logs,
					fmt.Sprintf("Update manifest warning: [%s] %s", warning.Code, warning.Message))
			}

			// Add manifest files to artifacts for distribution
			if manifestResult.RequiresUpload && len(manifestResult.ManifestPaths) > 0 {
				for filename, path := range manifestResult.ManifestPaths {
					// Use a special key format for manifests: "manifest:<filename>"
					artifactKey := "manifest:" + filename
					artifacts[artifactKey] = path
					result.Logs = append(result.Logs,
						fmt.Sprintf("Generated update manifest: %s", filename))
				}
			}
		}
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
		targetStr := strings.Join(input.Config.DistributionTargets, ",")
		if targetStr == "" {
			targetStr = "all"
		}
		failStage(result, s.timeProvider, errors.ErrDistributionStartFailed(err, targetStr))
		return result
	}

	result.Logs = append(result.Logs, fmt.Sprintf("Distribution started: %s", resp.DistributionID))

	// Wait for distribution to complete
	distStatus, waitErr := s.waitForDistribution(ctx, resp.DistributionID)
	if waitErr != nil {
		// Handle wait error based on its kind
		if waitErr.Kind == WaitErrorStore {
			failStage(result, s.timeProvider, errors.ErrDistributionStoreNotConfigured())
		} else if waitErr.Kind == WaitErrorTimeout {
			failStage(result, s.timeProvider, errors.ErrDistributionTimeout(resp.DistributionID, DefaultDistributionTimeout.String()))
		} else if waitErr.Kind == WaitErrorCancelled {
			failStage(result, s.timeProvider, errors.New(errors.CodePipelineCancelled, "distribution cancelled").InDomain("distribution"))
		} else {
			failStage(result, s.timeProvider, errors.ErrDistributionFailed(waitErr, "unknown"))
		}
		return result
	}

	// Check distribution result
	switch distStatus.Status {
	case distribution.StatusCompleted:
		result.Logs = append(result.Logs, "All targets uploaded successfully")
	case distribution.StatusPartial:
		result.Logs = append(result.Logs, "Distribution completed with some failures")
	case distribution.StatusFailed:
		failStage(result, s.timeProvider, errors.ErrDistributionFailed(
			fmt.Errorf("%s", distStatus.Error),
			"unknown",
		))
		result.Details = distStatus
		return result
	case distribution.StatusCancelled:
		failStage(result, s.timeProvider, errors.New(errors.CodePipelineCancelled, "distribution cancelled").InDomain("distribution"))
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

// waitForDistribution polls for distribution completion using the generic Poller.
func (s *DistributionStage) waitForDistribution(ctx context.Context, distributionID string) (*distribution.DistributionStatus, *WaitError) {
	if s.store == nil {
		return nil, &WaitError{
			Kind:       WaitErrorStore,
			EntityType: "distribution",
			EntityID:   distributionID,
		}
	}

	poller := &Poller[*distribution.DistributionStatus]{
		Config: PollerConfig{
			EntityType:   "Distribution",
			Timeout:      DefaultDistributionTimeout,
			PollInterval: DefaultDistributionPollInterval,
			LogInterval:  10, // Log every ~50 seconds
		},
		GetStatus: s.store.Get,
		IsComplete: func(status *distribution.DistributionStatus) bool {
			switch status.Status {
			case distribution.StatusCompleted, distribution.StatusPartial, distribution.StatusFailed, distribution.StatusCancelled:
				return true
			}
			return false
		},
	}

	return poller.Wait(ctx, distributionID)
}

// generateUpdateManifests creates update manifest files for auto-update support.
func (s *DistributionStage) generateUpdateManifests(ctx context.Context, input *StageInput, artifacts map[string]string) (*updates.GenerateManifestsResult, error) {
	// Determine output directory for manifests
	// Use the same directory as the first artifact
	outputDir := ""
	for _, path := range artifacts {
		outputDir = filepath.Dir(path)
		break
	}

	req := &updates.GenerateManifestsRequest{
		Config:    input.Config.UpdateConfig,
		Version:   input.Config.Version,
		Artifacts: artifacts,
		OutputDir: outputDir,
	}

	return s.manifestGenerator.GenerateManifests(ctx, req)
}
