package pipeline

import (
	"context"
	"fmt"
	"path/filepath"

	"scenario-to-desktop-api/preflight"
)

// PreflightStage implements the preflight validation stage of the pipeline.
type PreflightStage struct {
	service      preflight.Service
	timeProvider TimeProvider
}

// PreflightStageOption configures a PreflightStage.
type PreflightStageOption func(*PreflightStage)

// WithPreflightService sets the preflight service.
func WithPreflightService(svc preflight.Service) PreflightStageOption {
	return func(s *PreflightStage) {
		s.service = svc
	}
}

// WithPreflightTimeProvider sets the time provider.
func WithPreflightTimeProvider(tp TimeProvider) PreflightStageOption {
	return func(s *PreflightStage) {
		s.timeProvider = tp
	}
}

// NewPreflightStage creates a new preflight stage.
func NewPreflightStage(opts ...PreflightStageOption) *PreflightStage {
	s := &PreflightStage{
		timeProvider: NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the stage name.
func (s *PreflightStage) Name() string {
	return StagePreflight
}

// Dependencies returns stages that must complete before this one.
func (s *PreflightStage) Dependencies() []string {
	return []string{StageBundle}
}

// CanSkip returns whether this stage can be skipped.
func (s *PreflightStage) CanSkip(input *StageInput) bool {
	// Skip if explicitly requested
	if input.Config.SkipPreflight {
		return true
	}
	// Skip if deployment mode is proxy (no bundle to validate)
	if input.Config.GetDeploymentMode() == "proxy" {
		return true
	}
	return false
}

// Execute runs the preflight validation stage.
func (s *PreflightStage) Execute(ctx context.Context, input *StageInput) *StageResult {
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
		if input.Config.SkipPreflight {
			result.Logs = append(result.Logs, "Skipping preflight: explicitly skipped via config")
		} else {
			result.Logs = append(result.Logs, "Skipping preflight: deployment mode is proxy")
		}
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
		result.Error = "preflight service not configured"
		return result
	}

	// Get bundle info from previous stage
	if input.BundleResult == nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = "bundle result not available from previous stage"
		return result
	}

	manifestPath := input.BundleResult.ManifestPath
	bundleRoot := filepath.Dir(input.BundleResult.BundleDir)

	result.Logs = append(result.Logs, fmt.Sprintf("Validating bundle: %s", bundleRoot))

	// Build preflight request
	request := preflight.Request{
		BundleManifestPath: manifestPath,
		BundleRoot:         bundleRoot,
		Secrets:            input.Config.PreflightSecrets,
		StartServices:      true,
	}

	if input.Config.PreflightTimeoutSeconds > 0 {
		request.TimeoutSeconds = input.Config.PreflightTimeoutSeconds
	} else {
		request.TimeoutSeconds = 60 // Default timeout
	}

	// Run preflight validation
	response, err := s.service.RunBundlePreflight(request)
	if err != nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = fmt.Sprintf("preflight validation failed: %v", err)
		return result
	}

	// Check validation result
	if response.Status == "error" || response.Status == "failed" {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		if len(response.Errors) > 0 {
			result.Error = fmt.Sprintf("preflight validation failed: %v", response.Errors)
		} else {
			result.Error = "preflight validation failed"
		}
		result.Details = response
		return result
	}

	// Update input for next stage
	input.PreflightResult = response

	result.Status = StatusCompleted
	result.CompletedAt = s.timeProvider.Now()
	result.Details = response
	result.Logs = append(result.Logs,
		fmt.Sprintf("Preflight status: %s", response.Status),
	)

	// Add validation details
	if response.Validation != nil {
		result.Logs = append(result.Logs,
			fmt.Sprintf("Validation checks: %d passed", len(response.Checks)),
		)
	}

	// Add readiness info
	if response.Ready != nil && response.Ready.Ready {
		result.Logs = append(result.Logs, "Bundle services are ready")
	}

	return result
}
