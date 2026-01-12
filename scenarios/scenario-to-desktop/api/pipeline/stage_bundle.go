package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"scenario-to-desktop-api/bundle"
)

// BundleStage implements the bundle packaging stage of the pipeline.
type BundleStage struct {
	packager     bundle.Packager
	timeProvider TimeProvider
	scenarioRoot string
}

// BundleStageOption configures a BundleStage.
type BundleStageOption func(*BundleStage)

// WithBundlePackager sets the bundle packager.
func WithBundlePackager(p bundle.Packager) BundleStageOption {
	return func(s *BundleStage) {
		s.packager = p
	}
}

// WithBundleTimeProvider sets the time provider.
func WithBundleTimeProvider(tp TimeProvider) BundleStageOption {
	return func(s *BundleStage) {
		s.timeProvider = tp
	}
}

// WithScenarioRoot sets the root path for scenarios.
func WithScenarioRoot(root string) BundleStageOption {
	return func(s *BundleStage) {
		s.scenarioRoot = root
	}
}

// NewBundleStage creates a new bundle stage.
func NewBundleStage(opts ...BundleStageOption) *BundleStage {
	s := &BundleStage{
		timeProvider: NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(s)
	}
	// Default scenario root
	if s.scenarioRoot == "" {
		home, _ := os.UserHomeDir()
		s.scenarioRoot = filepath.Join(home, "Vrooli", "scenarios")
	}
	return s
}

// Name returns the stage name.
func (s *BundleStage) Name() string {
	return StageBundle
}

// Dependencies returns stages that must complete before this one.
func (s *BundleStage) Dependencies() []string {
	return nil // Bundle is the first stage
}

// CanSkip returns whether this stage can be skipped.
// Bundle stage is only skipped if deployment mode is "proxy".
func (s *BundleStage) CanSkip(input *StageInput) bool {
	return input.Config.GetDeploymentMode() == "proxy"
}

// Execute runs the bundle packaging stage.
func (s *BundleStage) Execute(ctx context.Context, input *StageInput) *StageResult {
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
		result.Logs = append(result.Logs, "Skipping bundle stage: deployment mode is proxy")
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

	// Determine scenario path
	scenarioPath := input.ScenarioPath
	if scenarioPath == "" {
		scenarioPath = filepath.Join(s.scenarioRoot, input.Config.ScenarioName)
	}

	// Determine manifest path
	manifestPath := input.Config.BundleManifestPath
	if manifestPath == "" {
		manifestPath = filepath.Join(scenarioPath, "bundle", "bundle.json")
	}

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = fmt.Sprintf("bundle manifest not found: %s", manifestPath)
		return result
	}

	result.Logs = append(result.Logs, fmt.Sprintf("Using manifest: %s", manifestPath))
	result.Logs = append(result.Logs, fmt.Sprintf("Packaging for platforms: %v", input.Config.Platforms))

	// Check for packager
	if s.packager == nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = "bundle packager not configured"
		return result
	}

	// Run the packager
	packageResult, err := s.packager.Package(scenarioPath, manifestPath, input.Config.Platforms)
	if err != nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = fmt.Sprintf("bundle packaging failed: %v", err)
		return result
	}

	// Update input for next stage
	input.BundleResult = packageResult

	result.Status = StatusCompleted
	result.CompletedAt = s.timeProvider.Now()
	result.Details = packageResult
	result.Logs = append(result.Logs,
		fmt.Sprintf("Bundle created: %s", packageResult.BundleDir),
		fmt.Sprintf("Total size: %s", packageResult.TotalSizeHuman),
	)

	if packageResult.SizeWarning != nil {
		result.Logs = append(result.Logs,
			fmt.Sprintf("Warning: %s", packageResult.SizeWarning.Message),
		)
	}

	return result
}
