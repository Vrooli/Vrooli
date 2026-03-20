package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/records"
	"scenario-to-desktop-api/scenario"
	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/system"
	"scenario-to-desktop-api/toolexecution"
)

// systemBuildStoreAdapter adapts build.Store to system.BuildStore interface
type systemBuildStoreAdapter struct {
	store *build.InMemoryStore
}

func (a *systemBuildStoreAdapter) Snapshot() map[string]*system.BuildStatus {
	snapshot := a.store.Snapshot()
	result := make(map[string]*system.BuildStatus, len(snapshot))
	for id, status := range snapshot {
		result[id] = &system.BuildStatus{
			Status: status.Status,
		}
	}
	return result
}

// pipelineStoreAdapter adapts pipeline.Orchestrator to tasks.PipelineStore interface
type pipelineStoreAdapter struct {
	store pipeline.Orchestrator
}

func (a *pipelineStoreAdapter) Get(pipelineID string) (*pipeline.Status, bool) {
	return a.store.GetStatus(pipelineID)
}

// toolPipelineOrchestratorAdapter adapts pipeline.Orchestrator to toolexecution.PipelineOrchestrator.
type toolPipelineOrchestratorAdapter struct {
	orchestrator pipeline.Orchestrator
}

func (a *toolPipelineOrchestratorAdapter) RunPipeline(ctx context.Context, config *toolexecution.PipelineConfig) (*toolexecution.PipelineStatus, error) {
	if a.orchestrator == nil {
		return nil, fmt.Errorf("pipeline orchestrator not configured")
	}
	status, err := a.orchestrator.RunPipeline(ctx, toPipelineConfig(config))
	if err != nil {
		return nil, err
	}
	return toToolPipelineStatus(status), nil
}

func (a *toolPipelineOrchestratorAdapter) ResumePipeline(ctx context.Context, pipelineID string, config *toolexecution.PipelineConfig) (*toolexecution.PipelineStatus, error) {
	if a.orchestrator == nil {
		return nil, fmt.Errorf("pipeline orchestrator not configured")
	}
	status, err := a.orchestrator.ResumePipeline(ctx, pipelineID, toPipelineConfig(config))
	if err != nil {
		return nil, err
	}
	return toToolPipelineStatus(status), nil
}

func (a *toolPipelineOrchestratorAdapter) GetStatus(pipelineID string) (*toolexecution.PipelineStatus, bool) {
	if a.orchestrator == nil {
		return nil, false
	}
	status, ok := a.orchestrator.GetStatus(pipelineID)
	if !ok {
		return nil, false
	}
	return toToolPipelineStatus(status), true
}

func (a *toolPipelineOrchestratorAdapter) CancelPipeline(pipelineID string) bool {
	if a.orchestrator == nil {
		return false
	}
	return a.orchestrator.CancelPipeline(pipelineID)
}

func (a *toolPipelineOrchestratorAdapter) ListPipelines() []*toolexecution.PipelineStatus {
	if a.orchestrator == nil {
		return nil
	}
	statuses := a.orchestrator.ListPipelines()
	result := make([]*toolexecution.PipelineStatus, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, toToolPipelineStatus(status))
	}
	return result
}

func toPipelineConfig(config *toolexecution.PipelineConfig) *pipeline.Config {
	if config == nil {
		return &pipeline.Config{}
	}
	pcfg := &pipeline.Config{
		ScenarioName:   config.ScenarioName,
		Platforms:      config.Platforms,
		DeploymentMode: config.DeploymentMode,
		TemplateType:   config.TemplateType,
		LocationMode:   config.LocationMode,
		StopAfterStage: config.StopAfterStage,
		SkipPreflight:  config.SkipPreflight,
		SkipSmokeTest:  config.SkipSmokeTest,
		Sign:           config.Sign,
		Clean:          config.Clean,
		Version:        config.Version,
		ProxyURL:       config.ProxyURL,
	}

	// Preserve backward-compatible tool flags by mapping them to DeployConfig.
	if config.DeployTarget != "" || config.DeployTo != "" || config.RemoteProfile != "" || config.AppKey != "" {
		pcfg.DeployConfig = &pipeline.DeployConfig{
			TargetName:    config.DeployTarget,
			ScenarioName:  config.DeployTo,
			RemoteProfile: config.RemoteProfile,
			AppKey:        config.AppKey,
		}
	}
	return pcfg
}

func toToolPipelineStatus(status *pipeline.Status) *toolexecution.PipelineStatus {
	if status == nil {
		return nil
	}
	result := &toolexecution.PipelineStatus{
		PipelineID:   status.PipelineID,
		ScenarioName: status.ScenarioName,
		Status:       status.Status,
		CurrentStage: status.CurrentStage,
		Error:        status.Error,
		CreatedAt:    time.Unix(status.StartedAt, 0),
	}
	if status.CompletedAt != 0 {
		completed := time.Unix(status.CompletedAt, 0)
		result.CompletedAt = &completed
	}

	result.Stages = make([]toolexecution.StageStatus, 0, len(status.Stages))
	for stageName, stage := range status.Stages {
		if stage == nil {
			continue
		}
		stageStatus := toolexecution.StageStatus{
			Name:   stageName,
			Status: stage.Status,
			Error:  stage.Error,
		}
		started := time.Unix(stage.StartedAt, 0)
		stageStatus.StartedAt = &started
		if stage.CompletedAt != 0 {
			completed := time.Unix(stage.CompletedAt, 0)
			stageStatus.EndedAt = &completed
		}
		result.Stages = append(result.Stages, stageStatus)
	}

	return result
}

// generationBuildStoreAdapter adapts build.InMemoryStore to generation.BuildStore interface
type generationBuildStoreAdapter struct {
	store *build.InMemoryStore
}

func (a *generationBuildStoreAdapter) Create(buildID string) *generation.BuildStatus {
	now := time.Now()
	status := &generation.BuildStatus{
		BuildID:   buildID,
		Status:    "building",
		StartedAt: now,
		BuildLog:  []string{},
		ErrorLog:  []string{},
		Artifacts: map[string]string{},
		Metadata:  map[string]interface{}{},
	}
	// Save to underlying store
	a.store.Save(&build.Status{
		BuildID:   buildID,
		Status:    "building",
		CreatedAt: now,
		BuildLog:  []string{},
		ErrorLog:  []string{},
		Artifacts: map[string]string{},
		Metadata:  map[string]interface{}{},
	})
	return status
}

func (a *generationBuildStoreAdapter) Get(buildID string) (*generation.BuildStatus, bool) {
	status, ok := a.store.Get(buildID)
	if !ok {
		return nil, false
	}
	return &generation.BuildStatus{
		BuildID:     status.BuildID,
		Status:      status.Status,
		OutputPath:  status.OutputPath,
		StartedAt:   status.CreatedAt,
		CompletedAt: status.CompletedAt,
		BuildLog:    status.BuildLog,
		ErrorLog:    status.ErrorLog,
		Artifacts:   status.Artifacts,
		Metadata:    status.Metadata,
	}, true
}

func (a *generationBuildStoreAdapter) Update(buildID string, fn func(status *generation.BuildStatus)) {
	a.store.Update(buildID, func(status *build.Status) {
		// Convert to generation.BuildStatus, apply fn, convert back
		genStatus := &generation.BuildStatus{
			BuildID:     status.BuildID,
			Status:      status.Status,
			OutputPath:  status.OutputPath,
			StartedAt:   status.CreatedAt,
			CompletedAt: status.CompletedAt,
			BuildLog:    status.BuildLog,
			ErrorLog:    status.ErrorLog,
			Artifacts:   status.Artifacts,
			Metadata:    status.Metadata,
		}
		fn(genStatus)
		// Copy back relevant fields
		status.Status = genStatus.Status
		status.OutputPath = genStatus.OutputPath
		status.CompletedAt = genStatus.CompletedAt
		status.BuildLog = genStatus.BuildLog
		status.ErrorLog = genStatus.ErrorLog
		status.Artifacts = genStatus.Artifacts
		status.Metadata = genStatus.Metadata
	})
}

// toolBuildStoreAdapter adapts build.InMemoryStore to toolexecution.BuildStore interface
type toolBuildStoreAdapter struct {
	store *build.InMemoryStore
}

func (a *toolBuildStoreAdapter) Get(buildID string) (toolexecution.BuildStatus, bool) {
	status, ok := a.store.Get(buildID)
	if !ok {
		return toolexecution.BuildStatus{}, false
	}
	return toolexecution.BuildStatus{
		BuildID:      status.BuildID,
		ScenarioName: status.ScenarioName,
		Status:       status.Status,
		Platforms:    status.Platforms,
		OutputPath:   status.OutputPath,
		ErrorLog:     status.ErrorLog,
		BuildLog:     status.BuildLog,
		Artifacts:    status.Artifacts,
		CreatedAt:    status.CreatedAt,
		CompletedAt:  status.CompletedAt,
		Metadata:     status.Metadata,
	}, true
}

func (a *toolBuildStoreAdapter) Save(status toolexecution.BuildStatus) {
	a.store.Save(&build.Status{
		BuildID:      status.BuildID,
		ScenarioName: status.ScenarioName,
		Status:       status.Status,
		Platforms:    status.Platforms,
		OutputPath:   status.OutputPath,
		ErrorLog:     status.ErrorLog,
		BuildLog:     status.BuildLog,
		Artifacts:    status.Artifacts,
		CreatedAt:    status.CreatedAt,
		CompletedAt:  status.CompletedAt,
		Metadata:     status.Metadata,
	})
}

func (a *toolBuildStoreAdapter) Snapshot() map[string]toolexecution.BuildStatus {
	snapshot := a.store.Snapshot()
	result := make(map[string]toolexecution.BuildStatus, len(snapshot))
	for id, status := range snapshot {
		result[id] = toolexecution.BuildStatus{
			BuildID:      status.BuildID,
			ScenarioName: status.ScenarioName,
			Status:       status.Status,
			Platforms:    status.Platforms,
			OutputPath:   status.OutputPath,
			ErrorLog:     status.ErrorLog,
			BuildLog:     status.BuildLog,
			Artifacts:    status.Artifacts,
			CreatedAt:    status.CreatedAt,
			CompletedAt:  status.CompletedAt,
			Metadata:     status.Metadata,
		}
	}
	return result
}

// screenrecordingExecutorAdapter adapts smoketest.ProcessExecutor to screenrecording.CommandExecutor.
type screenrecordingExecutorAdapter struct {
	executor smoketest.ProcessExecutor
}

func (a *screenrecordingExecutorAdapter) ExecuteWithResult(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*screenrecording.ExecutionResult, error) {
	result, err := a.executor.ExecuteWithResult(ctx, workDir, command, args, env, timeout)
	if result == nil {
		return nil, err
	}
	// Always pass through the result (even on error) so callers can access stderr
	// for diagnostic messages. ProcessExecutor returns both result and error when
	// the process exits non-zero.
	return &screenrecording.ExecutionResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}, err
}

// generationRecordStoreAdapter adapts records.FileStore to generation.RecordStore interface
type generationRecordStoreAdapter struct {
	store *records.FileStore
}

func (a *generationRecordStoreAdapter) Upsert(record *generation.DesktopAppRecord) error {
	if a.store == nil {
		log.Printf("[adapters] warning: generationRecordStoreAdapter.Upsert called with nil store - record for %q will not be persisted", record.ScenarioName)
		return nil // Gracefully handle nil store
	}
	// Convert generation.DesktopAppRecord to records.DesktopAppRecord
	r := &records.DesktopAppRecord{
		ID:              record.ID,
		BuildID:         record.BuildID,
		ScenarioName:    record.ScenarioName,
		AppDisplayName:  record.AppDisplayName,
		TemplateType:    record.TemplateType,
		Framework:       record.Framework,
		LocationMode:    record.LocationMode,
		OutputPath:      record.OutputPath,
		DestinationPath: record.DestinationPath,
		StagingPath:     record.StagingPath,
		CustomPath:      record.CustomPath,
		DeploymentMode:  record.DeploymentMode,
		Icon:            record.Icon,
	}
	return a.store.Upsert(r)
}

// scenarioRecordStoreAdapter adapts records.FileStore to scenario.RecordStore interface
type scenarioRecordStoreAdapter struct {
	store *records.FileStore
}

func (a *scenarioRecordStoreAdapter) List() []*scenario.DesktopAppRecord {
	if a.store == nil {
		log.Printf("[adapters] warning: scenarioRecordStoreAdapter.List called with nil store - returning empty list")
		return nil
	}
	list := a.store.List()
	result := make([]*scenario.DesktopAppRecord, len(list))
	for i, r := range list {
		result[i] = &scenario.DesktopAppRecord{
			ID:              r.ID,
			ScenarioName:    r.ScenarioName,
			OutputPath:      r.OutputPath,
			StagingPath:     r.StagingPath,
			CustomPath:      r.CustomPath,
			DestinationPath: r.DestinationPath,
			LocationMode:    r.LocationMode,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
		}
	}
	return result
}
