package pipeline

import (
	"context"
	"fmt"
)

// Manager coordinates pipeline lifecycle for scenarios.
// It wraps the orchestrator and index store to provide high-level
// operations like "get or create active pipeline" and "archive and create new".
type Manager struct {
	orchestrator Orchestrator
	indexStore   *ScenarioIndexStore
	logger       Logger
}

// ManagerOption configures a Manager.
type ManagerOption func(*Manager)

// WithManagerOrchestrator sets the orchestrator.
func WithManagerOrchestrator(o Orchestrator) ManagerOption {
	return func(m *Manager) {
		m.orchestrator = o
	}
}

// WithManagerIndexStore sets the index store.
func WithManagerIndexStore(s *ScenarioIndexStore) ManagerOption {
	return func(m *Manager) {
		m.indexStore = s
	}
}

// WithManagerLogger sets the logger.
func WithManagerLogger(l Logger) ManagerOption {
	return func(m *Manager) {
		m.logger = l
	}
}

// NewManager creates a new pipeline manager.
func NewManager(opts ...ManagerOption) *Manager {
	m := &Manager{}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// GetOrCreateActivePipeline returns the active pipeline for a scenario.
// If no active pipeline exists, it creates a new one in "idle" state (not running).
// Returns the pipeline status and whether it was newly created.
func (m *Manager) GetOrCreateActivePipeline(ctx context.Context, scenarioName string, defaultConfig *Config) (*Status, bool, error) {
	if m.orchestrator == nil {
		return nil, false, fmt.Errorf("orchestrator not configured")
	}
	if m.indexStore == nil {
		return nil, false, fmt.Errorf("index store not configured")
	}

	// Check for existing active pipeline
	idx := m.indexStore.GetOrCreate(scenarioName)
	if idx.ActivePipelineID != "" {
		status, ok := m.orchestrator.GetStatus(idx.ActivePipelineID)
		if ok {
			m.logInfo("returning existing active pipeline",
				"scenario", scenarioName,
				"pipeline_id", idx.ActivePipelineID,
			)
			return status, false, nil
		}
		// Pipeline no longer exists, clear the index
		m.logWarn("active pipeline not found in store, clearing index",
			"scenario", scenarioName,
			"pipeline_id", idx.ActivePipelineID,
		)
		_ = m.indexStore.ClearActive(scenarioName)
	}

	// Create new IDLE pipeline (not running)
	// The pipeline will remain in "idle" state until the user explicitly starts it
	config := m.buildConfig(scenarioName, defaultConfig)
	status, err := m.orchestrator.CreateIdlePipeline(config)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create pipeline: %w", err)
	}

	// Update index
	if err := m.indexStore.SetActivePipeline(scenarioName, status.PipelineID); err != nil {
		m.logError("failed to update index after creating pipeline",
			"scenario", scenarioName,
			"pipeline_id", status.PipelineID,
			"error", err,
		)
	}

	m.logInfo("created new idle pipeline",
		"scenario", scenarioName,
		"pipeline_id", status.PipelineID,
	)

	return status, true, nil
}

// CreateNewPipeline archives the current active pipeline and creates a new idle one.
// Returns the new pipeline status and the archived pipeline ID (if any).
// The new pipeline is created in "idle" state, ready to be configured and started.
func (m *Manager) CreateNewPipeline(ctx context.Context, scenarioName string, config *Config) (*Status, string, error) {
	if m.orchestrator == nil {
		return nil, "", fmt.Errorf("orchestrator not configured")
	}
	if m.indexStore == nil {
		return nil, "", fmt.Errorf("index store not configured")
	}

	// Archive current active pipeline
	archivedID, err := m.indexStore.ArchiveActive(scenarioName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to archive active pipeline: %w", err)
	}

	if archivedID != "" {
		m.logInfo("archived active pipeline",
			"scenario", scenarioName,
			"archived_id", archivedID,
		)
	}

	// Create new IDLE pipeline (not running)
	pipelineConfig := m.buildConfig(scenarioName, config)
	status, err := m.orchestrator.CreateIdlePipeline(pipelineConfig)
	if err != nil {
		return nil, archivedID, fmt.Errorf("failed to create pipeline: %w", err)
	}

	// Update index
	if err := m.indexStore.SetActivePipeline(scenarioName, status.PipelineID); err != nil {
		m.logError("failed to update index after creating pipeline",
			"scenario", scenarioName,
			"pipeline_id", status.PipelineID,
			"error", err,
		)
	}

	m.logInfo("created new idle pipeline",
		"scenario", scenarioName,
		"pipeline_id", status.PipelineID,
		"archived_id", archivedID,
	)

	return status, archivedID, nil
}

// ResetActivePipeline archives the current active pipeline and clears the active slot.
// Returns the archived pipeline ID (if any).
func (m *Manager) ResetActivePipeline(scenarioName string) (string, error) {
	if m.indexStore == nil {
		return "", fmt.Errorf("index store not configured")
	}

	archivedID, err := m.indexStore.ArchiveActive(scenarioName)
	if err != nil {
		return "", fmt.Errorf("failed to archive active pipeline: %w", err)
	}

	m.logInfo("reset active pipeline",
		"scenario", scenarioName,
		"archived_id", archivedID,
	)

	return archivedID, nil
}

// GetPipelineHistory returns the history of pipelines for a scenario.
// Returns pipeline statuses for each historical pipeline ID that still exists.
func (m *Manager) GetPipelineHistory(scenarioName string, limit int) ([]*Status, int, error) {
	if m.orchestrator == nil {
		return nil, 0, fmt.Errorf("orchestrator not configured")
	}
	if m.indexStore == nil {
		return nil, 0, fmt.Errorf("index store not configured")
	}

	historyIDs := m.indexStore.GetHistory(scenarioName, limit)
	total := len(historyIDs)

	// Get full status for each pipeline
	pipelines := make([]*Status, 0, len(historyIDs))
	for _, id := range historyIDs {
		status, ok := m.orchestrator.GetStatus(id)
		if ok {
			pipelines = append(pipelines, status)
		}
	}

	return pipelines, total, nil
}

// IsRunning checks if the active pipeline for a scenario is currently running.
// Returns false for idle pipelines (not yet started).
func (m *Manager) IsRunning(scenarioName string) bool {
	if m.indexStore == nil {
		return false
	}

	idx := m.indexStore.Get(scenarioName)
	if idx == nil || idx.ActivePipelineID == "" {
		return false
	}

	if m.orchestrator == nil {
		return false
	}

	status, ok := m.orchestrator.GetStatus(idx.ActivePipelineID)
	if !ok {
		return false
	}

	// Idle pipelines are NOT running (they haven't started yet)
	return status.Status == StatusRunning || status.Status == StatusPending
}

// GetActivePipelineStatus returns the status of the active pipeline for a scenario.
// Returns nil if no active pipeline exists.
func (m *Manager) GetActivePipelineStatus(scenarioName string) (*Status, bool) {
	if m.indexStore == nil || m.orchestrator == nil {
		return nil, false
	}

	idx := m.indexStore.Get(scenarioName)
	if idx == nil || idx.ActivePipelineID == "" {
		return nil, false
	}

	return m.orchestrator.GetStatus(idx.ActivePipelineID)
}

// StartActivePipeline starts the active pipeline for a scenario.
// If the pipeline is idle, updates its config and starts it.
// If already running, returns the current status.
// If completed/failed, creates a new pipeline with config, updates index store, and starts it.
func (m *Manager) StartActivePipeline(ctx context.Context, scenarioName string, configOverrides *Config) (*Status, error) {
	if m.orchestrator == nil {
		return nil, fmt.Errorf("orchestrator not configured")
	}
	if m.indexStore == nil {
		return nil, fmt.Errorf("index store not configured")
	}

	// Get or create the active pipeline
	status, _, err := m.GetOrCreateActivePipeline(ctx, scenarioName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create active pipeline: %w", err)
	}

	// Check current status
	switch status.Status {
	case StatusRunning, StatusPending:
		// Already running - return current status
		m.logInfo("active pipeline already running",
			"scenario", scenarioName,
			"pipeline_id", status.PipelineID,
			"status", status.Status,
		)
		return status, nil

	case StatusIdle:
		// Pipeline is idle - update config if provided and start it
		if configOverrides != nil {
			if err := m.updateIdlePipelineConfig(status.PipelineID, configOverrides); err != nil {
				return nil, fmt.Errorf("failed to update pipeline config: %w", err)
			}
		}

		startedStatus, err := m.orchestrator.StartPipeline(ctx, status.PipelineID)
		if err != nil {
			return nil, fmt.Errorf("failed to start pipeline: %w", err)
		}

		m.logInfo("started active pipeline",
			"scenario", scenarioName,
			"pipeline_id", status.PipelineID,
		)

		return startedStatus, nil

	case StatusCompleted, StatusFailed, StatusCancelled:
		// Pipeline finished - create a new one, update index, and start it
		m.logInfo("active pipeline already completed, creating new one",
			"scenario", scenarioName,
			"old_pipeline_id", status.PipelineID,
			"old_status", status.Status,
		)

		// Archive the old pipeline and create a new one
		newStatus, archivedID, err := m.CreateNewPipeline(ctx, scenarioName, configOverrides)
		if err != nil {
			return nil, fmt.Errorf("failed to create new pipeline: %w", err)
		}

		if archivedID != "" {
			m.logInfo("archived previous pipeline",
				"scenario", scenarioName,
				"archived_id", archivedID,
			)
		}

		// Start the new pipeline
		startedStatus, err := m.orchestrator.StartPipeline(ctx, newStatus.PipelineID)
		if err != nil {
			return nil, fmt.Errorf("failed to start new pipeline: %w", err)
		}

		m.logInfo("started new active pipeline",
			"scenario", scenarioName,
			"pipeline_id", newStatus.PipelineID,
		)

		return startedStatus, nil

	default:
		return nil, fmt.Errorf("unexpected pipeline status: %s", status.Status)
	}
}

// StartActivePipelineBlocking starts the active pipeline and blocks until completion or timeout.
// This is similar to StartActivePipeline but waits for the pipeline to finish.
// Returns the final status when complete, failed, or cancelled.
// Returns an error if the timeout is exceeded or the pipeline disappears.
func (m *Manager) StartActivePipelineBlocking(ctx context.Context, scenarioName string, configOverrides *Config, timeoutSecs int) (*Status, error) {
	if m.orchestrator == nil {
		return nil, fmt.Errorf("orchestrator not configured")
	}
	if m.indexStore == nil {
		return nil, fmt.Errorf("index store not configured")
	}

	// Get or create the active pipeline
	status, _, err := m.GetOrCreateActivePipeline(ctx, scenarioName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create active pipeline: %w", err)
	}

	// Check current status
	switch status.Status {
	case StatusRunning, StatusPending:
		// Already running - poll for completion
		m.logInfo("active pipeline already running, waiting for completion",
			"scenario", scenarioName,
			"pipeline_id", status.PipelineID,
			"status", status.Status,
		)
		return m.pollForCompletion(ctx, status.PipelineID, timeoutSecs)

	case StatusIdle:
		// Pipeline is idle - update config if provided and start it
		if configOverrides != nil {
			if err := m.updateIdlePipelineConfig(status.PipelineID, configOverrides); err != nil {
				return nil, fmt.Errorf("failed to update pipeline config: %w", err)
			}
		}

		finalStatus, err := m.orchestrator.StartPipelineBlocking(ctx, status.PipelineID, timeoutSecs)
		if err != nil {
			return finalStatus, err
		}

		m.logInfo("active pipeline completed",
			"scenario", scenarioName,
			"pipeline_id", status.PipelineID,
			"final_status", finalStatus.Status,
		)

		return finalStatus, nil

	case StatusCompleted, StatusFailed, StatusCancelled:
		// Pipeline finished - create a new one, update index, and start it with blocking
		m.logInfo("active pipeline already completed, creating new one",
			"scenario", scenarioName,
			"old_pipeline_id", status.PipelineID,
			"old_status", status.Status,
		)

		// Archive the old pipeline and create a new one
		newStatus, archivedID, err := m.CreateNewPipeline(ctx, scenarioName, configOverrides)
		if err != nil {
			return nil, fmt.Errorf("failed to create new pipeline: %w", err)
		}

		if archivedID != "" {
			m.logInfo("archived previous pipeline",
				"scenario", scenarioName,
				"archived_id", archivedID,
			)
		}

		// Start the new pipeline with blocking
		finalStatus, err := m.orchestrator.StartPipelineBlocking(ctx, newStatus.PipelineID, timeoutSecs)
		if err != nil {
			return finalStatus, err
		}

		m.logInfo("new active pipeline completed",
			"scenario", scenarioName,
			"pipeline_id", newStatus.PipelineID,
			"final_status", finalStatus.Status,
		)

		return finalStatus, nil

	default:
		return nil, fmt.Errorf("unexpected pipeline status: %s", status.Status)
	}
}

func (m *Manager) updateIdlePipelineConfig(pipelineID string, config *Config) error {
	updater, ok := m.orchestrator.(ConfigUpdatingOrchestrator)
	if !ok {
		return fmt.Errorf("orchestrator does not support updating idle pipeline configuration")
	}
	if err := updater.UpdatePipelineConfig(pipelineID, config); err != nil {
		return fmt.Errorf("failed to update pipeline config: %w", err)
	}
	return nil
}

// pollForCompletion polls for pipeline completion until it finishes or times out.
func (m *Manager) pollForCompletion(ctx context.Context, pipelineID string, timeoutSecs int) (*Status, error) {
	// Delegate to the orchestrator's blocking implementation
	// Create a config with the pipeline ID to pass context
	status, ok := m.orchestrator.GetStatus(pipelineID)
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", pipelineID)
	}

	// If already complete, return immediately
	if status.IsComplete() {
		return status, nil
	}

	// Use the blocking method on DefaultOrchestrator
	if o, ok := m.orchestrator.(*DefaultOrchestrator); ok {
		return o.pollForCompletion(ctx, pipelineID, timeoutSecs)
	}

	return nil, fmt.Errorf("orchestrator does not support blocking operations")
}

// buildConfig creates a pipeline config, applying defaults from the provided config.
func (m *Manager) buildConfig(scenarioName string, userConfig *Config) *Config {
	config := &Config{
		ScenarioName: scenarioName,
	}

	if userConfig != nil {
		// Copy all user-specified fields
		config.Platforms = userConfig.Platforms
		config.SkipPreflight = userConfig.SkipPreflight
		config.SkipSmokeTest = userConfig.SkipSmokeTest
		config.StopOnFailure = userConfig.StopOnFailure
		config.DeploymentMode = userConfig.DeploymentMode
		config.Framework = userConfig.Framework
		config.TemplateType = userConfig.TemplateType
		config.LocationMode = userConfig.LocationMode
		config.WebhookURL = userConfig.WebhookURL
		config.ProxyURL = userConfig.ProxyURL
		config.BundleManifestPath = userConfig.BundleManifestPath
		config.ResourceArtifactRoot = userConfig.ResourceArtifactRoot
		config.Clean = userConfig.Clean
		config.Sign = userConfig.Sign
		config.Publish = userConfig.Publish
		config.DeployConfig = userConfig.DeployConfig
		config.Version = userConfig.Version
		config.setVersionRollback(userConfig.versionRollback)
		config.PreflightTimeoutSeconds = userConfig.PreflightTimeoutSeconds
		config.PreflightSecrets = userConfig.PreflightSecrets
		config.StopAfterStage = userConfig.StopAfterStage
		config.ResumeFromStage = userConfig.ResumeFromStage
		config.ParentPipelineID = userConfig.ParentPipelineID
		config.IdempotencyKey = userConfig.IdempotencyKey
	}

	return config
}

// logInfo logs an info message if logger is configured.
func (m *Manager) logInfo(msg string, args ...interface{}) {
	if m.logger != nil {
		m.logger.Info(msg, args...)
	}
}

// logWarn logs a warning message if logger is configured.
func (m *Manager) logWarn(msg string, args ...interface{}) {
	if m.logger != nil {
		m.logger.Warn(msg, args...)
	}
}

// logError logs an error message if logger is configured.
func (m *Manager) logError(msg string, args ...interface{}) {
	if m.logger != nil {
		m.logger.Error(msg, args...)
	}
}

// RecoverStalePipelines marks pipelines stuck in "running" or "pending" state as failed.
// This should be called at startup to recover from unclean shutdowns where
// the executing goroutines were lost but the persisted state still says "running".
// Returns the number of pipelines recovered.
func (m *Manager) RecoverStalePipelines() int {
	if m.orchestrator == nil {
		return 0
	}

	recovered := 0
	for _, status := range m.orchestrator.ListPipelines() {
		if status.Status != StatusRunning && status.Status != StatusPending {
			continue
		}

		pipelineID := status.PipelineID
		currentStage := ""
		if status.Stages != nil {
			for name, stage := range status.Stages {
				if stage != nil && stage.Status == StatusRunning {
					currentStage = name
					break
				}
			}
		}

		// Mark the pipeline as failed due to server restart
		m.orchestrator.CancelPipeline(pipelineID)

		// CancelPipeline may not transition to a terminal state if the cancel
		// context was lost on restart. Force-update the status via the store.
		updated := false
		if do, ok := m.orchestrator.(*DefaultOrchestrator); ok && do.store != nil {
			updated = do.store.Update(pipelineID, func(s *Status) {
				s.Status = StatusFailed
				s.Error = "Pipeline interrupted by server restart"
				// Also fail any running stages
				for _, stage := range s.Stages {
					if stage != nil && stage.Status == StatusRunning {
						stage.Status = StatusFailed
						stage.Error = "Stage interrupted by server restart"
					}
				}
			})
		}

		m.logWarn("recovered stale pipeline",
			"pipeline_id", pipelineID,
			"previous_status", status.Status,
			"current_stage", currentStage,
			"force_updated", updated,
		)
		recovered++
	}

	if recovered > 0 {
		m.logInfo("startup recovery complete",
			"recovered_pipelines", recovered,
		)
	}

	return recovered
}
