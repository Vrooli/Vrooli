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
	packager          bundle.Packager
	manifestGenerator ManifestGenerator
	timeProvider      TimeProvider
	scenarioRoot      string
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

// WithManifestGenerator sets the manifest generator for on-demand manifest creation.
func WithManifestGenerator(g ManifestGenerator) BundleStageOption {
	return func(s *BundleStage) {
		s.manifestGenerator = g
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
	result := newStageResult(s.Name(), s.timeProvider)

	if s.CanSkip(input) {
		skipStage(result, s.timeProvider, "Skipping bundle stage: deployment mode is proxy")
		return result
	}

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
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

	// Check if manifest exists, generate if not
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		if s.manifestGenerator == nil {
			failStage(result, s.timeProvider, fmt.Sprintf("bundle manifest not found: %s (no generator configured)", manifestPath))
			return result
		}

		result.Logs = append(result.Logs, "Manifest not found, generating via deployment-manager...")

		// Generate manifest
		outputDir := filepath.Dir(manifestPath)
		generatedPath, genErr := s.manifestGenerator.GenerateManifest(ctx, input.Config.ScenarioName, outputDir)
		if genErr != nil {
			failStage(result, s.timeProvider, fmt.Sprintf("failed to generate manifest: %v", genErr))
			return result
		}

		manifestPath = generatedPath
		result.Logs = append(result.Logs, fmt.Sprintf("Generated manifest: %s", manifestPath))
	}

	result.Logs = append(result.Logs, fmt.Sprintf("Using manifest: %s", manifestPath))
	result.Logs = append(result.Logs, fmt.Sprintf("Packaging for platforms: %v", input.Config.Platforms))

	// Check for packager
	if s.packager == nil {
		failStage(result, s.timeProvider, "bundle packager not configured")
		return result
	}

	// Run the packager
	packageResult, err := s.packager.Package(scenarioPath, manifestPath, input.Config.Platforms)
	if err != nil {
		failStage(result, s.timeProvider, fmt.Sprintf("bundle packaging failed: %v", err))
		return result
	}

	// Update input for next stage
	input.BundleResult = packageResult

	completeStage(result, s.timeProvider, packageResult)
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
