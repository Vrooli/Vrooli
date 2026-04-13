package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"scenario-to-desktop-api/bundle"
	"scenario-to-desktop-api/shared/errors"
	sharedpath "scenario-to-desktop-api/shared/path"
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
		s.scenarioRoot = sharedpath.DetectScenariosRoot()
		if s.scenarioRoot == "" {
			s.scenarioRoot = filepath.Clean("scenarios")
		}
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
	outputRoot, desktopPath := resolvePipelineOutputPaths(input.Config, scenarioPath, input.PipelineID, framework)

	if input.Config != nil && input.Config.Clean {
		// Clean outputs without deleting source/config (in "proper" location mode).
		// In staging/temp mode, removing the whole framework dir is safe.
		if desktopPath != "" && strings.Contains(desktopPath, filepath.Join("platforms", framework)) {
			appendInfo(result, "Cleaning desktop outputs under: %s", desktopPath)
			if err := cleanDesktopOutputs(desktopPath, input.Config.LocationMode); err != nil {
				failStage(result, s.timeProvider, errors.ErrBundlePackagingFailed(err, scenarioPath))
				return result
			}
		}
	}

	// Resolve or generate the bundle manifest
	manifestPath, manifestErr := s.resolveManifest(ctx, result, input.Config, scenarioPath, framework)
	if manifestErr != nil {
		failStage(result, s.timeProvider, manifestErr)
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
	packageResult, err := s.packager.Package(scenarioPath, manifestPath, framework, input.Config.Platforms, outputRoot)
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

// resolveManifest determines the manifest path, optionally generating it. Returns an error on failure.
func (s *BundleStage) resolveManifest(ctx context.Context, result *StageResult, config *Config, scenarioPath, framework string) (string, *errors.DomainError) {
	manifestPath := config.BundleManifestPath
	if manifestPath == "" {
		manifestPath = filepath.Join(scenarioPath, "platforms", framework, "bundle", "bundle.json")
	}

	if s.manifestGenerator != nil {
		appendInfo(result, "Generating manifest via deployment-manager...")
		outputDir := filepath.Dir(manifestPath)
		generatedPath, genErr := s.manifestGenerator.GenerateManifest(ctx, config.ScenarioName, outputDir)
		if genErr != nil {
			return "", errors.ErrBundleManifestGeneration(genErr)
		}
		appendInfo(result, "Generated manifest: %s", generatedPath)
		return generatedPath, nil
	}

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return "", errors.ErrBundleManifestNotFound(manifestPath)
	}
	return manifestPath, nil
}
