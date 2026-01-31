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
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if home directory is unavailable
			// This is a defensive fallback - in practice, home dir is almost always available
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				// Last resort: use a relative path (will be resolved against cwd at runtime)
				o.scenarioRoot = "scenarios"
			} else {
				o.scenarioRoot = filepath.Join(cwd, "scenarios")
			}
			if o.logger != nil {
				o.logger.Warn("UserHomeDir unavailable, using fallback",
					"error", err.Error(),
					"fallback_root", o.scenarioRoot,
				)
			}
		} else {
			o.scenarioRoot = filepath.Join(home, "Vrooli", "scenarios")
		}
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
// If an idempotency key is provided and a pipeline with that key exists, the existing
// pipeline status is returned instead of starting a new one. This enables safe retries
// where "running twice is no worse than running once".
func (o *DefaultOrchestrator) RunPipeline(ctx context.Context, config *Config) (*Status, error) {
	// Validate config
	if config.ScenarioName == "" {
		return nil, fmt.Errorf("scenario_name is required")
	}

	// Validate stop_after_stage if provided
	if config.StopAfterStage != "" && !IsValidStageName(config.StopAfterStage) {
		return nil, fmt.Errorf("invalid stop_after_stage: %s", config.StopAfterStage)
	}

	// Validate resume_from_stage if provided
	if config.ResumeFromStage != "" && !IsValidStageName(config.ResumeFromStage) {
		return nil, fmt.Errorf("invalid resume_from_stage: %s", config.ResumeFromStage)
	}

	// Idempotency check: if an idempotency key is provided, check for existing pipeline
	// This enables safe retries where replaying a request returns the existing pipeline
	// instead of starting duplicate work.
	if config.IdempotencyKey != "" {
		if existing, ok := o.store.GetByIdempotencyKey(config.IdempotencyKey); ok {
			o.logger.Info("Idempotency key matched existing pipeline",
				"idempotency_key", config.IdempotencyKey,
				"pipeline_id", existing.PipelineID,
				"status", existing.Status,
			)
			return existing, nil
		}
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
		PipelineID:     pipelineID,
		ScenarioName:   config.ScenarioName,
		Status:         StatusPending,
		Stages:         make(map[string]*StageResult),
		StageOrder:     stageOrder,
		Config:         config,
		StartedAt:      o.timeProvider.Now(),
		IdempotencyKey: config.IdempotencyKey, // Store for future lookups
	}

	// Set initial progress
	status.UpdateProgress()

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

// CreateIdlePipeline creates a pipeline in "idle" state without starting execution.
// The pipeline will remain idle until explicitly started via StartPipeline.
// This is used for auto-creating pipelines when a scenario is selected.
func (o *DefaultOrchestrator) CreateIdlePipeline(config *Config) (*Status, error) {
	// Validate config
	if config.ScenarioName == "" {
		return nil, fmt.Errorf("scenario_name is required")
	}

	// Validate stop_after_stage if provided
	if config.StopAfterStage != "" && !IsValidStageName(config.StopAfterStage) {
		return nil, fmt.Errorf("invalid stop_after_stage: %s", config.StopAfterStage)
	}

	// Idempotency check: if an idempotency key is provided, check for existing pipeline
	if config.IdempotencyKey != "" {
		if existing, ok := o.store.GetByIdempotencyKey(config.IdempotencyKey); ok {
			o.logger.Info("Idempotency key matched existing pipeline",
				"idempotency_key", config.IdempotencyKey,
				"pipeline_id", existing.PipelineID,
				"status", existing.Status,
			)
			return existing, nil
		}
	}

	// Apply defaults - don't set platforms yet since the user hasn't configured them
	// The config will be updated when the user starts the pipeline

	// Generate pipeline ID
	pipelineID := o.idGenerator.Generate()

	// Build stage order
	stageOrder := make([]string, 0, len(o.stages))
	for _, stage := range o.stages {
		stageOrder = append(stageOrder, stage.Name())
	}

	// Create idle status - NOT started yet
	status := &Status{
		PipelineID:     pipelineID,
		ScenarioName:   config.ScenarioName,
		Status:         StatusIdle,
		Stages:         make(map[string]*StageResult),
		StageOrder:     stageOrder,
		Config:         config,
		StartedAt:      o.timeProvider.Now(), // Track creation time
		IdempotencyKey: config.IdempotencyKey,
	}

	// Set initial progress message for idle state
	status.ProgressPercent = 0
	status.ProgressMessage = "Pipeline created, waiting to start"

	// Save initial status
	o.store.Save(status)

	o.logger.Info("Created idle pipeline",
		"pipeline_id", pipelineID,
		"scenario", config.ScenarioName,
	)

	return status, nil
}

// StartPipeline starts execution of an existing idle pipeline.
// Returns an error if the pipeline is not in idle state or doesn't exist.
func (o *DefaultOrchestrator) StartPipeline(ctx context.Context, pipelineID string) (*Status, error) {
	// Get existing status
	status, ok := o.store.Get(pipelineID)
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", pipelineID)
	}

	// Verify pipeline is in idle state
	if !status.IsIdle() {
		return nil, fmt.Errorf("pipeline cannot be started: status is %s (must be idle)", status.Status)
	}

	config := status.Config
	if config == nil {
		return nil, fmt.Errorf("pipeline has no config")
	}

	// Apply defaults now that we're starting
	if len(config.Platforms) == 0 {
		config.Platforms = []string{currentPlatform()}
	}

	// Update status to pending
	o.store.Update(pipelineID, func(s *Status) {
		s.Status = StatusPending
		s.UpdateProgress()
	})

	// Create cancellable context
	pipelineCtx, cancel := context.WithCancel(ctx)
	o.cancelManager.Set(pipelineID, cancel)

	// Run pipeline asynchronously
	go o.runPipelineAsync(pipelineCtx, pipelineID, config)

	// Get updated status
	updatedStatus, _ := o.store.Get(pipelineID)
	return updatedStatus, nil
}

// UpdatePipelineConfig updates the config of an idle pipeline.
// Returns error if pipeline is not idle or doesn't exist.
// This allows updating stop_after_stage and other config fields before starting.
func (o *DefaultOrchestrator) UpdatePipelineConfig(pipelineID string, configUpdates *Config) error {
	// Get existing status
	status, ok := o.store.Get(pipelineID)
	if !ok {
		return fmt.Errorf("pipeline not found: %s", pipelineID)
	}

	// Verify pipeline is in idle state
	if !status.IsIdle() {
		return fmt.Errorf("pipeline config cannot be updated: status is %s (must be idle)", status.Status)
	}

	if configUpdates == nil {
		return nil // Nothing to update
	}

	// Update config fields
	o.store.Update(pipelineID, func(s *Status) {
		if s.Config == nil {
			s.Config = &Config{}
		}

		// Only update fields that are explicitly set in configUpdates
		if configUpdates.StopAfterStage != "" {
			s.Config.StopAfterStage = configUpdates.StopAfterStage
		}
		if configUpdates.ResumeFromStage != "" {
			s.Config.ResumeFromStage = configUpdates.ResumeFromStage
		}
		if len(configUpdates.Platforms) > 0 {
			s.Config.Platforms = configUpdates.Platforms
		}
		if configUpdates.BundleManifestPath != "" {
			s.Config.BundleManifestPath = configUpdates.BundleManifestPath
		}
		if configUpdates.DeploymentMode != "" {
			s.Config.DeploymentMode = configUpdates.DeploymentMode
		}
		if configUpdates.Framework != "" {
			s.Config.Framework = configUpdates.Framework
		}
		if configUpdates.TemplateType != "" {
			s.Config.TemplateType = configUpdates.TemplateType
		}
		if configUpdates.ProxyURL != "" {
			s.Config.ProxyURL = configUpdates.ProxyURL
		}
		if configUpdates.Version != "" {
			s.Config.Version = configUpdates.Version
		}
		if configUpdates.PreflightTimeoutSeconds > 0 {
			s.Config.PreflightTimeoutSeconds = configUpdates.PreflightTimeoutSeconds
		}
		if len(configUpdates.PreflightSecrets) > 0 {
			s.Config.PreflightSecrets = configUpdates.PreflightSecrets
		}
		if len(configUpdates.DistributionTargets) > 0 {
			s.Config.DistributionTargets = configUpdates.DistributionTargets
		}
		// Boolean fields - only update if explicitly true (since default is false)
		if configUpdates.SkipPreflight {
			s.Config.SkipPreflight = true
		}
		if configUpdates.SkipSmokeTest {
			s.Config.SkipSmokeTest = true
		}
		if configUpdates.Clean {
			s.Config.Clean = true
		}
		if configUpdates.Sign {
			s.Config.Sign = true
		}
		if configUpdates.Publish {
			s.Config.Publish = true
		}
		if configUpdates.Distribute {
			s.Config.Distribute = true
		}
		if configUpdates.StopOnFailure != nil {
			s.Config.StopOnFailure = configUpdates.StopOnFailure
		}
	})

	o.logger.Info("Updated pipeline config",
		"pipeline_id", pipelineID,
		"stop_after_stage", configUpdates.StopAfterStage,
	)

	return nil
}

// runPipelineAsync executes the pipeline stages sequentially.
func (o *DefaultOrchestrator) runPipelineAsync(ctx context.Context, pipelineID string, config *Config) {
	defer o.cancelManager.Clear(pipelineID)

	// Update status to running
	o.store.Update(pipelineID, func(s *Status) {
		s.Status = StatusRunning
		s.UpdateProgress()
	})

	// Build stage input
	input := &StageInput{
		Config:       config,
		PipelineID:   pipelineID,
		ScenarioPath: filepath.Join(o.scenarioRoot, config.ScenarioName),
		Logger:       o.logger,
	}

	// If resuming, restore input from parent pipeline
	if config.ParentPipelineID != "" {
		if parentStatus, ok := o.store.Get(config.ParentPipelineID); ok && parentStatus.ResumedInput != nil {
			// Copy relevant fields from parent's saved input
			input.BundleResult = parentStatus.ResumedInput.BundleResult
			input.PreflightResult = parentStatus.ResumedInput.PreflightResult
			input.GenerationResult = parentStatus.ResumedInput.GenerationResult
			input.BuildResult = parentStatus.ResumedInput.BuildResult
			input.SmokeTestResult = parentStatus.ResumedInput.SmokeTestResult
			input.DistributionResult = parentStatus.ResumedInput.DistributionResult
			input.ScenarioMetadata = parentStatus.ResumedInput.ScenarioMetadata
			input.DesktopPath = parentStatus.ResumedInput.DesktopPath
			o.logger.Info("Restored input from parent pipeline", "pipeline_id", pipelineID, "parent_id", config.ParentPipelineID)
		}
	}

	// Track whether we've reached the resume stage (if resuming)
	resumeFromStage := config.GetResumeFromStage()
	reachedResumeStage := resumeFromStage == "" // If not resuming, consider it reached

	// Execute stages sequentially
	for _, stage := range o.stages {
		stageName := stage.Name()

		// If resuming, skip stages until we reach the resume point
		if !reachedResumeStage {
			if stageName == resumeFromStage {
				reachedResumeStage = true
				o.logger.Info("Reached resume stage", "pipeline_id", pipelineID, "stage", stageName)
			} else {
				// Mark stage as skipped (resumed from later stage)
				result := &StageResult{
					Stage:       stageName,
					Status:      StatusSkipped,
					StartedAt:   o.timeProvider.Now(),
					CompletedAt: o.timeProvider.Now(),
					Logs:        []string{"Stage skipped - resuming from later stage"},
				}
				o.store.UpdateStage(pipelineID, stageName, result)
				o.logger.Info("Stage skipped (resuming)", "pipeline_id", pipelineID, "stage", stageName)
				continue
			}
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			o.store.Update(pipelineID, func(s *Status) {
				s.Status = StatusCancelled
				s.CompletedAt = o.timeProvider.Now()
				s.Error = "pipeline cancelled"
				s.UpdateProgress()
			})
			o.logger.Info("Pipeline cancelled", "pipeline_id", pipelineID)
			return
		default:
		}

		// Update current stage
		o.store.Update(pipelineID, func(s *Status) {
			s.CurrentStage = stageName
			s.UpdateProgress()
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

			// Even if skipped, check if we should stop after this stage
			if config.GetStopAfterStage() == stageName {
				o.stopAfterStage(pipelineID, stageName, input)
				return
			}
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
					s.UpdateProgress()
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
				s.UpdateProgress()
			})
			o.logger.Info("Pipeline cancelled at stage", "pipeline_id", pipelineID, "stage", stageName)
			return
		}

		// Check if we should stop after this stage
		if config.GetStopAfterStage() == stageName {
			o.stopAfterStage(pipelineID, stageName, input)
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
		s.UpdateProgress()
	})

	o.logger.Info("Pipeline completed", "pipeline_id", pipelineID)
}

// stopAfterStage marks the pipeline as completed after the specified stage and saves input for resumption.
func (o *DefaultOrchestrator) stopAfterStage(pipelineID, stageName string, input *StageInput) {
	finalArtifacts := collectArtifacts(input)

	o.store.Update(pipelineID, func(s *Status) {
		s.Status = StatusCompleted
		s.CompletedAt = o.timeProvider.Now()
		s.CurrentStage = ""
		s.StoppedAfterStage = stageName
		s.FinalArtifacts = finalArtifacts
		// Save the input so it can be restored when resuming
		s.ResumedInput = input
		s.UpdateProgress()
	})

	o.logger.Info("Pipeline stopped after stage",
		"pipeline_id", pipelineID,
		"stopped_after", stageName,
	)
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

// ResumePipeline resumes a stopped pipeline from its next stage.
func (o *DefaultOrchestrator) ResumePipeline(ctx context.Context, pipelineID string, config *Config) (*Status, error) {
	// Get the parent pipeline
	parentStatus, ok := o.store.Get(pipelineID)
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", pipelineID)
	}

	// Validate parent pipeline can be resumed
	if !parentStatus.CanResume() {
		if parentStatus.Status != StatusCompleted {
			return nil, fmt.Errorf("pipeline cannot be resumed: status is %s (must be completed)", parentStatus.Status)
		}
		return nil, fmt.Errorf("pipeline cannot be resumed: was not stopped after a stage")
	}

	// Determine the resume stage
	nextStage := parentStatus.GetNextResumeStage()
	if nextStage == "" {
		return nil, fmt.Errorf("pipeline cannot be resumed: already completed all stages")
	}

	// Create the resume config
	resumeConfig := &Config{
		ScenarioName:            parentStatus.Config.ScenarioName,
		Platforms:               parentStatus.Config.Platforms,
		DeploymentMode:          parentStatus.Config.DeploymentMode,
		TemplateType:            parentStatus.Config.TemplateType,
		ProxyURL:                parentStatus.Config.ProxyURL,
		BundleManifestPath:      parentStatus.Config.BundleManifestPath,
		Sign:                    parentStatus.Config.Sign,
		Distribute:              parentStatus.Config.Distribute,
		DistributionTargets:     parentStatus.Config.DistributionTargets,
		Version:                 parentStatus.Config.Version,
		PreflightSecrets:        parentStatus.Config.PreflightSecrets,
		PreflightTimeoutSeconds: parentStatus.Config.PreflightTimeoutSeconds,
		// Set the resume configuration
		ResumeFromStage:  nextStage,
		ParentPipelineID: pipelineID,
	}

	// Apply any overrides from the provided config
	if config != nil {
		if config.StopAfterStage != "" {
			resumeConfig.StopAfterStage = config.StopAfterStage
		}
		if config.SkipSmokeTest {
			resumeConfig.SkipSmokeTest = config.SkipSmokeTest
		}
		if config.Distribute {
			resumeConfig.Distribute = config.Distribute
		}
		if len(config.DistributionTargets) > 0 {
			resumeConfig.DistributionTargets = config.DistributionTargets
		}
	}

	// Run the resumed pipeline
	return o.RunPipeline(ctx, resumeConfig)
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
