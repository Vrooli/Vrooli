package pipeline

import (
	"context"
	"os"
	"path/filepath"

	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/shared/errors"
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
	return ShouldSkipBundle(input.Config)
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

	// Determine framework (default to electron)
	framework := input.Config.Framework
	if framework == "" {
		framework = "electron"
	}

	// Determine manifest path
	manifestPath := input.Config.BundleManifestPath
	if manifestPath == "" {
		manifestPath = filepath.Join(scenarioPath, "platforms", framework, "bundle", "bundle.json")
	}

	// Always regenerate manifest for fresh detection when generator is configured
	if s.manifestGenerator != nil {
		appendInfo(result, "Generating manifest via deployment-manager...")

		// Generate manifest
		outputDir := filepath.Dir(manifestPath)
		generatedPath, genErr := s.manifestGenerator.GenerateManifest(ctx, input.Config.ScenarioName, outputDir)
		if genErr != nil {
			failStage(result, s.timeProvider, errors.ErrBundleManifestGeneration(genErr))
			return result
		}

		manifestPath = generatedPath
		appendInfo(result, "Generated manifest: %s", manifestPath)
	} else if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		// No generator configured and manifest doesn't exist - fail
		failStage(result, s.timeProvider, errors.ErrBundleManifestNotFound(manifestPath))
		return result
	}

	appendInfo(result, "Using manifest: %s", manifestPath)
	appendInfo(result, "Packaging for platforms: %v", input.Config.Platforms)

	// Check for packager
	if s.packager == nil {
		failStage(result, s.timeProvider, errors.ErrBundleServiceNotConfigured())
		return result
	}

	// Run the packager
	packageResult, err := s.packager.Package(scenarioPath, manifestPath, framework, input.Config.Platforms)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrBundlePackagingFailed(err, scenarioPath))
		return result
	}

	// Update input for next stage
	input.BundleResult = packageResult

	appendInfo(result, "Bundle created: %s", packageResult.BundleDir)
	appendInfo(result, "Total size: %s", packageResult.TotalSizeHuman)

	if packageResult.SizeWarning != nil {
		appendWarn(result, "Bundle size warning: %s", packageResult.SizeWarning.Message)
	}

	completeStage(result, s.timeProvider, packageResult)

	return result
}
