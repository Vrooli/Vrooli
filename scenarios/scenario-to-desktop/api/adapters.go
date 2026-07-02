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

// smokeTestRecordAdapter adapts smoketest.FileStore to records.SmokeTestStoreAdapter interface.
type smokeTestRecordAdapter struct {
	store *smoketest.FileStore
}

func (a *smokeTestRecordAdapter) GetByScenario(scenarioName string) (string, *records.ScreenRecordingView, bool) {
	st, ok := a.store.GetByScenario(scenarioName)
	if !ok {
		return "", nil, false
	}
	var sr *records.ScreenRecordingView
	if st.ScreenRecording != nil {
		sr = &records.ScreenRecordingView{
			Recorded:      st.ScreenRecording.Recorded,
			DurationMS:    st.ScreenRecording.DurationMs,
			FileSizeBytes: st.ScreenRecording.FileSizeBytes,
			Error:         st.ScreenRecording.Error,
		}
	}
	return st.SmokeTestID, sr, true
}

// recordsBuildStoreAdapter adapts build.InMemoryStore to records.BuildStoreAdapter interface
type recordsBuildStoreAdapter struct {
	store *build.InMemoryStore
}

func (a *recordsBuildStoreAdapter) Get(id string) (*records.BuildStatusView, bool) {
	status, ok := a.store.Get(id)
	if !ok {
		return nil, false
	}
	return &records.BuildStatusView{
		Status:     status.Status,
		OutputPath: status.OutputPath,
		Metadata:   status.Metadata,
	}, true
}

func (a *recordsBuildStoreAdapter) Update(id string, fn func(status *records.BuildStatusView)) error {
	ok := a.store.Update(id, func(status *build.Status) {
		view := &records.BuildStatusView{
			Status:     status.Status,
			OutputPath: status.OutputPath,
			Metadata:   status.Metadata,
		}
		fn(view)
		status.Status = view.Status
		status.OutputPath = view.OutputPath
		status.Metadata = view.Metadata
	})
	if !ok {
		return fmt.Errorf("build %q not found", id)
	}
	return nil
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
