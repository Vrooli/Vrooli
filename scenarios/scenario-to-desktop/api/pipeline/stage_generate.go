package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/shared/errors"
)

// GenerateStage implements the desktop wrapper generation stage of the pipeline.
type GenerateStage struct {
	service      generation.Service
	analyzer     generation.ScenarioAnalyzer
	buildStore   generation.BuildStore // for polling build status
	timeProvider TimeProvider
	scenarioRoot string
}

// GenerateStageOption configures a GenerateStage.
type GenerateStageOption func(*GenerateStage)

// WithGenerateService sets the generation service.
func WithGenerateService(svc generation.Service) GenerateStageOption {
	return func(s *GenerateStage) {
		s.service = svc
	}
}

// WithScenarioAnalyzer sets the scenario analyzer.
func WithScenarioAnalyzer(a generation.ScenarioAnalyzer) GenerateStageOption {
	return func(s *GenerateStage) {
		s.analyzer = a
	}
}

// WithGenerateTimeProvider sets the time provider.
func WithGenerateTimeProvider(tp TimeProvider) GenerateStageOption {
	return func(s *GenerateStage) {
		s.timeProvider = tp
	}
}

// WithGenerateScenarioRoot sets the scenario root path.
func WithGenerateScenarioRoot(root string) GenerateStageOption {
	return func(s *GenerateStage) {
		s.scenarioRoot = root
	}
}

// WithGenerateBuildStore sets the build store for polling build status.
func WithGenerateBuildStore(store generation.BuildStore) GenerateStageOption {
	return func(s *GenerateStage) {
		s.buildStore = store
	}
}

// NewGenerateStage creates a new generate stage.
func NewGenerateStage(opts ...GenerateStageOption) *GenerateStage {
	s := &GenerateStage{
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
func (s *GenerateStage) Name() string {
	return StageGenerate
}

// Dependencies returns stages that must complete before this one.
func (s *GenerateStage) Dependencies() []string {
	// Depends on preflight (which may have been skipped)
	return []string{StagePreflight}
}

// CanSkip returns whether this stage can be skipped.
// Generation is never skipped - it's always required.
func (s *GenerateStage) CanSkip(input *StageInput) bool {
	return false
}

// Execute runs the desktop generation stage.
func (s *GenerateStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := newStageResult(s.Name(), s.timeProvider)

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}

	if s.analyzer == nil {
		failStage(result, s.timeProvider, errors.ErrGenerateAnalyzerNotConfigured())
		return result
	}

	scenarioName := input.Config.ScenarioName
	result.Logs = append(result.Logs, fmt.Sprintf("Analyzing scenario: %s", scenarioName))

	// Analyze the scenario
	metadata, err := s.analyzer.AnalyzeScenario(scenarioName)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrScenarioAnalysisFailed(err, scenarioName))
		return result
	}

	input.ScenarioMetadata = metadata
	result.Logs = append(result.Logs, fmt.Sprintf("Detected: %s v%s", metadata.DisplayName, metadata.Version))

	// Validate scenario is ready for desktop
	if err := s.analyzer.ValidateScenarioForDesktop(scenarioName); err != nil {
		failStage(result, s.timeProvider, errors.ErrScenarioValidationFailed(err, scenarioName))
		return result
	}

	// Create desktop config from metadata
	templateType := input.Config.GetTemplateType()
	desktopConfig, err := s.analyzer.CreateDesktopConfigFromMetadata(metadata, templateType)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrDesktopConfigFailed(err))
		return result
	}

	// Apply pipeline config overrides
	desktopConfig.DeploymentMode = input.Config.GetDeploymentMode()
	desktopConfig.Platforms = input.Config.Platforms
	if input.Config.ProxyURL != "" {
		desktopConfig.ProxyURL = input.Config.ProxyURL
	}
	if input.BundleResult != nil {
		desktopConfig.BundleManifestPath = input.BundleResult.ManifestPath
		desktopConfig.BundleRuntimeRoot = input.BundleResult.BundleDir
	}

	result.Logs = append(result.Logs,
		fmt.Sprintf("Deployment mode: %s", desktopConfig.DeploymentMode),
		fmt.Sprintf("Template type: %s", templateType),
	)

	if s.service == nil {
		failStage(result, s.timeProvider, errors.ErrGenerateServiceNotConfigured())
		return result
	}

	// Queue the generation
	buildID := fmt.Sprintf("gen-%s-%d", scenarioName, time.Now().UnixMilli())
	buildStatus := s.service.QueueBuild(desktopConfig, metadata, true)

	// Wait for generation to complete (poll with cancellation support)
	desktopPath, err := s.waitForGeneration(ctx, buildID, buildStatus)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrGenerationFailed(err).WithDetail("build_id", buildID))
		return result
	}

	// Update input for next stage
	input.DesktopPath = desktopPath
	input.GenerationResult = &generation.GenerateResponse{
		BuildID:     buildID,
		Status:      "ready",
		DesktopPath: desktopPath,
	}

	completeStage(result, s.timeProvider, input.GenerationResult)
	result.Logs = append(result.Logs,
		fmt.Sprintf("Desktop wrapper generated: %s", desktopPath),
	)

	return result
}

// waitForGeneration polls for generation completion.
func (s *GenerateStage) waitForGeneration(ctx context.Context, buildID string, initialStatus *generation.BuildStatus) (string, error) {
	// Quick check for synchronous completion
	if initialStatus.Status == BuildStatusReady && initialStatus.OutputPath != "" {
		return initialStatus.OutputPath, nil
	}

	// If still building, wait with timeout
	timeout := time.After(DefaultGenerationTimeout)
	ticker := time.NewTicker(DefaultGenerationPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("generation cancelled")
		case <-timeout:
			return "", fmt.Errorf("generation timed out after %v", DefaultGenerationTimeout)
		case <-ticker.C:
			// FIX: Poll fresh status from store instead of checking stale pointer
			// The QueueBuild function spawns an async goroutine that updates the store,
			// so we must query the store directly to see the latest status.
			if s.buildStore == nil {
				// Fallback to checking initialStatus if no store configured
				// (this maintains backward compatibility but won't see async updates)
				switch initialStatus.Status {
				case BuildStatusReady:
					return initialStatus.OutputPath, nil
				case BuildStatusFailed:
					if len(initialStatus.ErrorLog) > 0 {
						return "", fmt.Errorf("generation failed: %s", initialStatus.ErrorLog[len(initialStatus.ErrorLog)-1])
					}
					return "", fmt.Errorf("generation failed")
				}
				continue
			}

			currentStatus, ok := s.buildStore.Get(buildID)
			if !ok {
				return "", fmt.Errorf("build status not found in store: %s", buildID)
			}

			switch currentStatus.Status {
			case BuildStatusReady:
				return currentStatus.OutputPath, nil
			case BuildStatusFailed:
				if len(currentStatus.ErrorLog) > 0 {
					return "", fmt.Errorf("generation failed: %s", currentStatus.ErrorLog[len(currentStatus.ErrorLog)-1])
				}
				return "", fmt.Errorf("generation failed")
			}
		}
	}
}
