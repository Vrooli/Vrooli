package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"scenario-to-desktop-api/generation"
)

// GenerateStage implements the desktop wrapper generation stage of the pipeline.
type GenerateStage struct {
	service      generation.Service
	analyzer     generation.ScenarioAnalyzer
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

	// Check for analyzer (required for scenario analysis)
	if s.analyzer == nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = "scenario analyzer not configured"
		return result
	}

	scenarioName := input.Config.ScenarioName
	result.Logs = append(result.Logs, fmt.Sprintf("Analyzing scenario: %s", scenarioName))

	// Analyze the scenario
	metadata, err := s.analyzer.AnalyzeScenario(scenarioName)
	if err != nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = fmt.Sprintf("scenario analysis failed: %v", err)
		return result
	}

	input.ScenarioMetadata = metadata
	result.Logs = append(result.Logs, fmt.Sprintf("Detected: %s v%s", metadata.DisplayName, metadata.Version))

	// Validate scenario is ready for desktop
	if err := s.analyzer.ValidateScenarioForDesktop(scenarioName); err != nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = fmt.Sprintf("scenario validation failed: %v", err)
		return result
	}

	// Create desktop config from metadata
	templateType := input.Config.GetTemplateType()
	desktopConfig, err := s.analyzer.CreateDesktopConfigFromMetadata(metadata, templateType)
	if err != nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = fmt.Sprintf("failed to create desktop config: %v", err)
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

	// Check for service
	if s.service == nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = "generation service not configured"
		return result
	}

	// Queue the generation
	buildID := fmt.Sprintf("gen-%s-%d", scenarioName, time.Now().UnixMilli())
	buildStatus := s.service.QueueBuild(desktopConfig, metadata, true)

	// Wait for generation to complete (poll with cancellation support)
	desktopPath, err := s.waitForGeneration(ctx, buildID, buildStatus)
	if err != nil {
		result.Status = StatusFailed
		result.CompletedAt = s.timeProvider.Now()
		result.Error = err.Error()
		return result
	}

	// Update input for next stage
	input.DesktopPath = desktopPath
	input.GenerationResult = &generation.GenerateResponse{
		BuildID:     buildID,
		Status:      "ready",
		DesktopPath: desktopPath,
	}

	result.Status = StatusCompleted
	result.CompletedAt = s.timeProvider.Now()
	result.Details = input.GenerationResult
	result.Logs = append(result.Logs,
		fmt.Sprintf("Desktop wrapper generated: %s", desktopPath),
	)

	return result
}

// waitForGeneration polls for generation completion.
func (s *GenerateStage) waitForGeneration(ctx context.Context, buildID string, status *generation.BuildStatus) (string, error) {
	// For synchronous generation, the status is already complete
	// The QueueBuild function may run async, so we need to check
	if status.Status == "ready" && status.OutputPath != "" {
		return status.OutputPath, nil
	}

	// If still building, wait with timeout
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("generation cancelled")
		case <-timeout:
			return "", fmt.Errorf("generation timed out after 5 minutes")
		case <-ticker.C:
			// Check status
			switch status.Status {
			case "ready":
				return status.OutputPath, nil
			case "failed":
				if len(status.ErrorLog) > 0 {
					return "", fmt.Errorf("generation failed: %s", status.ErrorLog[len(status.ErrorLog)-1])
				}
				return "", fmt.Errorf("generation failed")
			}
		}
	}
}
