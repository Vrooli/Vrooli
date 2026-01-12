package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultOrchestrator implements the Orchestrator interface.
type DefaultOrchestrator struct {
	store         Store
	cancelManager CancelManager
	idGenerator   IDGenerator
	timeProvider  TimeProvider
	logger        Logger
	stages        []Stage
	scenarioRoot  string
}

// OrchestratorOption configures a DefaultOrchestrator.
type OrchestratorOption func(*DefaultOrchestrator)

// WithStore sets the pipeline store.
func WithStore(store Store) OrchestratorOption {
	return func(o *DefaultOrchestrator) {
		o.store = store
	}
}

// WithCancelManager sets the cancel manager.
func WithCancelManager(cm CancelManager) OrchestratorOption {
	return func(o *DefaultOrchestrator) {
		o.cancelManager = cm
	}
}

// WithIDGenerator sets the ID generator.
func WithIDGenerator(gen IDGenerator) OrchestratorOption {
	return func(o *DefaultOrchestrator) {
		o.idGenerator = gen
	}
}

// WithTimeProvider sets the time provider.
func WithTimeProvider(tp TimeProvider) OrchestratorOption {
	return func(o *DefaultOrchestrator) {
		o.timeProvider = tp
	}
}

// WithLogger sets the logger.
func WithLogger(l Logger) OrchestratorOption {
	return func(o *DefaultOrchestrator) {
		o.logger = l
	}
}

// WithStages sets the pipeline stages.
func WithStages(stages ...Stage) OrchestratorOption {
	return func(o *DefaultOrchestrator) {
		o.stages = stages
	}
}

// WithOrchestratorScenarioRoot sets the scenario root path.
func WithOrchestratorScenarioRoot(root string) OrchestratorOption {
	return func(o *DefaultOrchestrator) {
		o.scenarioRoot = root
	}
}

// NewOrchestrator creates a new pipeline orchestrator.
func NewOrchestrator(opts ...OrchestratorOption) *DefaultOrchestrator {
	o := &DefaultOrchestrator{
		store:         NewInMemoryStore(),
		cancelManager: NewInMemoryCancelManager(),
		idGenerator:   NewUUIDGenerator(),
		timeProvider:  NewRealTimeProvider(),
	}

	for _, opt := range opts {
		opt(o)
	}

	// Default logger
	if o.logger == nil {
		o.logger = &SlogLogger{Logger: slog.Default()}
	}

	// Default scenario root
	if o.scenarioRoot == "" {
		home, _ := os.UserHomeDir()
		o.scenarioRoot = filepath.Join(home, "Vrooli", "scenarios")
	}

	// Default stages if none provided
	if len(o.stages) == 0 {
		o.stages = []Stage{
			NewBundleStage(WithScenarioRoot(o.scenarioRoot)),
			NewPreflightStage(),
			NewGenerateStage(WithGenerateScenarioRoot(o.scenarioRoot)),
			NewBuildStage(),
			NewSmokeTestStage(),
		}
	}

	return o
}

// SlogLogger adapts slog.Logger to the Logger interface.
type SlogLogger struct {
	Logger *slog.Logger
}

func (l *SlogLogger) Info(msg string, args ...interface{})  { l.Logger.Info(msg, args...) }
func (l *SlogLogger) Warn(msg string, args ...interface{})  { l.Logger.Warn(msg, args...) }
func (l *SlogLogger) Error(msg string, args ...interface{}) { l.Logger.Error(msg, args...) }
func (l *SlogLogger) Debug(msg string, args ...interface{}) { l.Logger.Debug(msg, args...) }

// RunPipeline starts a new pipeline execution.
func (o *DefaultOrchestrator) RunPipeline(ctx context.Context, config *Config) (*Status, error) {
	// Validate config
	if config.ScenarioName == "" {
		return nil, fmt.Errorf("scenario_name is required")
	}

	// Apply defaults
	if len(config.Platforms) == 0 {
		config.Platforms = []string{currentPlatform()}
	}

	// Generate pipeline ID
	pipelineID := o.idGenerator.Generate()

	// Build stage order
	stageOrder := make([]string, 0, len(o.stages))
	for _, stage := range o.stages {
		stageOrder = append(stageOrder, stage.Name())
	}

	// Create initial status
	status := &Status{
		PipelineID:   pipelineID,
		ScenarioName: config.ScenarioName,
		Status:       StatusPending,
		Stages:       make(map[string]*StageResult),
		StageOrder:   stageOrder,
		Config:       config,
		StartedAt:    o.timeProvider.Now(),
	}

	// Save initial status
	o.store.Save(status)

	// Create cancellable context
	pipelineCtx, cancel := context.WithCancel(ctx)
	o.cancelManager.Set(pipelineID, cancel)

	// Run pipeline asynchronously
	go o.runPipelineAsync(pipelineCtx, pipelineID, config)

	// Return immediately with pipeline ID
	return status, nil
}

// runPipelineAsync executes the pipeline stages sequentially.
func (o *DefaultOrchestrator) runPipelineAsync(ctx context.Context, pipelineID string, config *Config) {
	defer o.cancelManager.Clear(pipelineID)

	// Update status to running
	o.store.Update(pipelineID, func(s *Status) {
		s.Status = StatusRunning
	})

	// Build stage input
	input := &StageInput{
		Config:       config,
		PipelineID:   pipelineID,
		ScenarioPath: filepath.Join(o.scenarioRoot, config.ScenarioName),
		Logger:       o.logger,
	}

	// Execute stages sequentially
	for _, stage := range o.stages {
		stageName := stage.Name()

		// Check for cancellation
		select {
		case <-ctx.Done():
			o.store.Update(pipelineID, func(s *Status) {
				s.Status = StatusCancelled
				s.CompletedAt = o.timeProvider.Now()
				s.Error = "pipeline cancelled"
			})
			o.logger.Info("Pipeline cancelled", "pipeline_id", pipelineID)
			return
		default:
		}

		// Update current stage
		o.store.Update(pipelineID, func(s *Status) {
			s.CurrentStage = stageName
		})

		o.logger.Info("Starting stage", "pipeline_id", pipelineID, "stage", stageName)

		// Check if stage can be skipped
		if stage.CanSkip(input) {
			result := &StageResult{
				Stage:       stageName,
				Status:      StatusSkipped,
				StartedAt:   o.timeProvider.Now(),
				CompletedAt: o.timeProvider.Now(),
				Logs:        []string{"Stage skipped based on configuration"},
			}
			o.store.UpdateStage(pipelineID, stageName, result)
			o.logger.Info("Stage skipped", "pipeline_id", pipelineID, "stage", stageName)
			continue
		}

		// Execute stage
		result := stage.Execute(ctx, input)
		o.store.UpdateStage(pipelineID, stageName, result)

		o.logger.Info("Stage completed",
			"pipeline_id", pipelineID,
			"stage", stageName,
			"status", result.Status,
		)

		// Check for failure
		if result.Status == StatusFailed {
			if config.GetStopOnFailure() {
				o.store.Update(pipelineID, func(s *Status) {
					s.Status = StatusFailed
					s.CompletedAt = o.timeProvider.Now()
					s.Error = fmt.Sprintf("stage %s failed: %s", stageName, result.Error)
				})
				o.logger.Error("Pipeline failed", "pipeline_id", pipelineID, "stage", stageName, "error", result.Error)
				return
			}
			o.logger.Warn("Stage failed but continuing", "pipeline_id", pipelineID, "stage", stageName)
		}

		// Check for cancellation
		if result.Status == StatusCancelled {
			o.store.Update(pipelineID, func(s *Status) {
				s.Status = StatusCancelled
				s.CompletedAt = o.timeProvider.Now()
				s.Error = "pipeline cancelled"
			})
			o.logger.Info("Pipeline cancelled at stage", "pipeline_id", pipelineID, "stage", stageName)
			return
		}
	}

	// Collect final artifacts
	finalArtifacts := collectArtifacts(input)

	// Mark pipeline as completed
	o.store.Update(pipelineID, func(s *Status) {
		s.Status = StatusCompleted
		s.CompletedAt = o.timeProvider.Now()
		s.CurrentStage = ""
		s.FinalArtifacts = finalArtifacts
	})

	o.logger.Info("Pipeline completed", "pipeline_id", pipelineID)
}

// GetStatus retrieves the current status of a pipeline run.
func (o *DefaultOrchestrator) GetStatus(pipelineID string) (*Status, bool) {
	return o.store.Get(pipelineID)
}

// CancelPipeline cancels a running pipeline.
func (o *DefaultOrchestrator) CancelPipeline(pipelineID string) bool {
	cancel := o.cancelManager.Take(pipelineID)
	if cancel == nil {
		// No cancellation function found - pipeline may not be running
		return false
	}
	cancel()
	return true
}

// ListPipelines returns all tracked pipeline runs.
func (o *DefaultOrchestrator) ListPipelines() []*Status {
	return o.store.List()
}

// currentPlatform returns the current platform identifier.
func currentPlatform() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map to electron-builder platform names
	switch goos {
	case "darwin":
		if goarch == "arm64" {
			return "mac-arm64"
		}
		return "mac"
	case "windows":
		return "win"
	case "linux":
		return "linux"
	default:
		return goos
	}
}

// collectArtifacts gathers final artifact paths from the pipeline input.
func collectArtifacts(input *StageInput) map[string]string {
	artifacts := make(map[string]string)

	if input.BuildResult != nil {
		for platform, result := range input.BuildResult.PlatformResults {
			if result.Status == "ready" && result.Artifact != "" {
				artifacts[platform] = result.Artifact
			}
		}
	}

	return artifacts
}
