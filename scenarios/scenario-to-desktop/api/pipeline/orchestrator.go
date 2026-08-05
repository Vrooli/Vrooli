// DOC: docs/reference/api-architecture.md#pipeline-system-core-engine
// DOC: docs/internal/SEAMS.md#pipeline-orchestrator-seam
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"

	"scenario-to-desktop-api/deploy"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/shared/validation"
	"scenario-to-desktop-api/storagepaths"

	sharedpath "scenario-to-desktop-api/shared/path"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
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
		o.scenarioRoot = sharedpath.DetectScenariosRoot()
		if o.scenarioRoot == "" {
			o.scenarioRoot = filepath.Clean("scenarios")
			if o.logger != nil {
				o.logger.Warn("Scenario root unavailable from repo contract", "fallback_root", o.scenarioRoot)
			}
		}
	}

	// Default stages if none provided
	if len(o.stages) == 0 {
		// Create analyzer for bundleability checking in preflight stage
		// NewAnalyzer expects vrooliRoot (parent of scenarios directory), not scenarioRoot
		vrooliRoot := filepath.Dir(o.scenarioRoot)
		analyzer := generation.NewAnalyzer(vrooliRoot)
		var targetRepo *deploy.TargetRepository
		if locator, err := storagepaths.NewLocator(); err == nil {
			if path, err := locator.DeployTargetsPath(); err == nil {
				targetRepo = deploy.NewTargetRepository(path)
			}
		}
		o.stages = []Stage{
			NewResolveDeploymentStage(WithResolveDeploymentScenarioRoot(o.scenarioRoot)),
			NewBundleStage(WithScenarioRoot(o.scenarioRoot)),
			NewPreflightStage(WithBundleabilityChecker(analyzer)),
			NewGenerateStage(WithGenerateScenarioRoot(o.scenarioRoot)),
			NewBuildStage(),
			NewSmokeTestStage(),
			NewDeployStage(WithDeployTargetRepo(targetRepo)),
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
	if err := validatePipelineConfig(config); err != nil {
		return nil, err
	}
	if existing := o.idempotentPipeline(config.IdempotencyKey); existing != nil {
		return existing, nil
	}
	if err := normalizePipelinePlatforms(config); err != nil {
		return nil, err
	}
	status := o.newPipelineStatus(config)
	o.store.Save(status)
	// A pipeline is server-owned work. Its lifetime must not be coupled to the
	// short-lived Connect request that created it; explicit cancellation remains
	// available through the cancel manager.
	pipelineCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	o.cancelManager.Set(status.PipelineID, cancel)
	go o.runPipelineAsync(pipelineCtx, status.PipelineID, config)
	return status, nil
}

func validatePipelineConfig(config *Config) error {
	if err := config.ValidateFramework(); err != nil {
		return err
	}
	if !validation.IsSafeScenarioName(config.ScenarioName) {
		if config.ScenarioName == "" {
			return fmt.Errorf("scenario_name is required")
		}
		return fmt.Errorf("invalid scenario_name: contains path traversal characters")
	}
	if config.StopAfterStage != "" && !IsValidStageName(config.StopAfterStage) {
		return fmt.Errorf("invalid stop_after_stage: %s", config.StopAfterStage)
	}
	if config.ResumeFromStage != "" && !IsValidStageName(config.ResumeFromStage) {
		return fmt.Errorf("invalid resume_from_stage: %s", config.ResumeFromStage)
	}
	for _, stage := range config.GetStages() {
		if !IsValidStageName(stage) {
			return fmt.Errorf("invalid stage name: %q", stage)
		}
	}
	return nil
}

func (o *DefaultOrchestrator) idempotentPipeline(key string) *Status {
	if key == "" {
		return nil
	}
	existing, ok := o.store.GetByIdempotencyKey(key)
	if !ok {
		return nil
	}
	o.logger.Info("Idempotency key matched existing pipeline", "idempotency_key", key, "pipeline_id", existing.PipelineID, "status", existing.Status)
	return existing
}

func normalizePipelinePlatforms(config *Config) error {
	if len(config.Platforms) == 0 {
		config.Platforms = []string{currentPlatform()}
	}
	normalized := make([]string, 0, len(config.Platforms))
	for _, value := range config.Platforms {
		platform, err := normalizeDesktopPlatform(value)
		if err != nil {
			return err
		}
		normalized = append(normalized, platform.String())
	}
	config.Platforms = normalized
	return nil
}

func (o *DefaultOrchestrator) newPipelineStatus(config *Config) *Status {
	stages := o.stages
	if requested := config.GetStages(); len(requested) > 0 {
		stages = o.filterStages(requested)
	}
	order := make([]string, 0, len(stages))
	for _, stage := range stages {
		order = append(order, stage.Name())
	}
	status := &Status{PipelineID: o.idGenerator.Generate(), ScenarioName: config.ScenarioName, Status: StatusPending, Stages: make(map[string]*StageResult), StageOrder: order, Config: config, StartedAt: o.timeProvider.Now(), IdempotencyKey: config.IdempotencyKey}
	status.TransitionTo(PipelineStateCreated, "Pipeline created and queued")
	status.UpdateProgress()
	return status
}

func normalizeDesktopPlatform(value string) (resourcedeployment.Platform, error) {
	value = strings.TrimSpace(value)
	if !strings.ContainsAny(value, "-_/") {
		return resourcedeployment.CanonicalPlatform(value, runtime.GOARCH)
	}
	return resourcedeployment.ParsePlatform(value)
}

// RunPipelineBlocking runs a pipeline and blocks until completion or timeout.
// It starts the pipeline asynchronously, then polls for completion.
// Returns the final status when complete, failed, or cancelled.
// Returns an error if the timeout is exceeded or the pipeline disappears.
func (o *DefaultOrchestrator) RunPipelineBlocking(ctx context.Context, config *Config, timeoutSecs int) (*Status, error) {
	// Start pipeline async
	status, err := o.RunPipeline(ctx, config)
	if err != nil {
		return nil, err
	}

	return o.pollForCompletion(ctx, status.PipelineID, timeoutSecs)
}

// StartPipelineBlocking starts an existing idle pipeline and blocks until completion or timeout.
// It starts the pipeline, then polls for completion.
// Returns the final status when complete, failed, or cancelled.
// Returns an error if the timeout is exceeded, the pipeline disappears, or the pipeline is not idle.
func (o *DefaultOrchestrator) StartPipelineBlocking(ctx context.Context, pipelineID string, timeoutSecs int) (*Status, error) {
	// Start the idle pipeline
	_, err := o.StartPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}

	return o.pollForCompletion(ctx, pipelineID, timeoutSecs)
}

// CreateIdlePipeline creates a pipeline in "idle" state without starting execution.
// The pipeline will remain idle until explicitly started via StartPipeline.
// This is used for auto-creating pipelines when a scenario is selected.
func (o *DefaultOrchestrator) CreateIdlePipeline(config *Config) (*Status, error) {
	// Validate config
	if !validation.IsSafeScenarioName(config.ScenarioName) {
		if config.ScenarioName == "" {
			return nil, fmt.Errorf("scenario_name is required")
		}
		return nil, fmt.Errorf("invalid scenario_name: contains path traversal characters")
	}

	// Validate stop_after_stage if provided
	if config.StopAfterStage != "" && !IsValidStageName(config.StopAfterStage) {
		return nil, fmt.Errorf("invalid stop_after_stage: %s", config.StopAfterStage)
	}

	// Validate stages if provided
	if stages := config.GetStages(); len(stages) > 0 {
		for _, stage := range stages {
			if !IsValidStageName(stage) {
				return nil, fmt.Errorf("invalid stage name: %q", stage)
			}
		}
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

	// Build stage order (filtered if specific stages requested)
	stagesToUse := o.stages
	if requestedStages := config.GetStages(); len(requestedStages) > 0 {
		stagesToUse = o.filterStages(requestedStages)
	}
	stageOrder := make([]string, 0, len(stagesToUse))
	for _, stage := range stagesToUse {
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

	// Set initial state and progress message for idle state
	status.TransitionTo(PipelineStateCreated, "Pipeline created")
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
	// Resuming follows the same server-owned lifetime rule as a new pipeline.
	pipelineCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
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
		applyConfigStringFields(s.Config, configUpdates)
		applyConfigBoolFields(s.Config, configUpdates)
		applyConfigComplexFields(s.Config, configUpdates)
		if len(configUpdates.Stages) > 0 {
			s.Config.Stages = configUpdates.Stages
			stagesToUse := o.filterStages(configUpdates.Stages)
			s.StageOrder = make([]string, 0, len(stagesToUse))
			for _, stage := range stagesToUse {
				s.StageOrder = append(s.StageOrder, stage.Name())
			}
		}
	})

	o.logger.Info("Updated pipeline config",
		"pipeline_id", pipelineID,
		"stop_after_stage", configUpdates.StopAfterStage,
	)

	return nil
}

// applyConfigStringFields applies non-empty string field updates from src to dst.
func applyConfigStringFields(dst, src *Config) {
	if src.StopAfterStage != "" {
		dst.StopAfterStage = src.StopAfterStage
	}
	if src.ResumeFromStage != "" {
		dst.ResumeFromStage = src.ResumeFromStage
	}
	if src.BundleManifestPath != "" {
		dst.BundleManifestPath = src.BundleManifestPath
	}
	if src.ResourceArtifactRoot != "" {
		dst.ResourceArtifactRoot = src.ResourceArtifactRoot
	}
	if src.DeploymentMode != "" {
		dst.DeploymentMode = src.DeploymentMode
	}
	if src.Framework != "" {
		dst.Framework = src.Framework
	}
	if src.TemplateType != "" {
		dst.TemplateType = src.TemplateType
	}
	if src.LocationMode != "" {
		dst.LocationMode = src.LocationMode
	}
	if src.ProxyURL != "" {
		dst.ProxyURL = src.ProxyURL
	}
	if src.Version != "" {
		dst.Version = src.Version
	}
	if src.UpdateConfig != nil {
		dst.UpdateConfig = src.UpdateConfig
	}
}

// applyConfigBoolFields applies boolean field updates from src to dst.
// Boolean fields are only updated when explicitly true (since default is false).
func applyConfigBoolFields(dst, src *Config) {
	if src.SkipPreflight {
		dst.SkipPreflight = true
	}
	if src.SkipSmokeTest {
		dst.SkipSmokeTest = true
	}
	if src.Clean {
		dst.Clean = true
	}
	if src.Sign {
		dst.Sign = true
	}
	if src.Publish {
		dst.Publish = true
	}
}

// applyConfigComplexFields applies non-nil/non-zero complex field updates from src to dst.
func applyConfigComplexFields(dst, src *Config) {
	if len(src.Platforms) > 0 {
		dst.Platforms = src.Platforms
	}
	if src.versionRollback != nil {
		dst.versionRollback = src.versionRollback
	}
	if src.PreflightTimeoutSeconds > 0 {
		dst.PreflightTimeoutSeconds = src.PreflightTimeoutSeconds
	}
	if len(src.PreflightSecrets) > 0 {
		dst.PreflightSecrets = src.PreflightSecrets
	}
	if src.DeployConfig != nil {
		dst.DeployConfig = src.DeployConfig
	}
	if src.StopOnFailure != nil {
		dst.StopOnFailure = src.StopOnFailure
	}
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
		ResourceArtifactRoot:    parentStatus.Config.ResourceArtifactRoot,
		Sign:                    parentStatus.Config.Sign,
		DeployConfig:            parentStatus.Config.DeployConfig,
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
		if config.DeployConfig != nil {
			resumeConfig.DeployConfig = config.DeployConfig
		}
		if len(config.Stages) > 0 {
			resumeConfig.Stages = config.Stages
		}
	}

	// Run the resumed pipeline
	return o.RunPipeline(ctx, resumeConfig)
}

// ListPipelines returns all tracked pipeline runs.
func (o *DefaultOrchestrator) ListPipelines() []*Status {
	return o.store.List()
}
