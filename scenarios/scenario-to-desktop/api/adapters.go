package main

import (
	"time"

	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/shared/packaging"
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

// defaultSmokeTestPackageFinder is the default package finder for smoke tests
type defaultSmokeTestPackageFinder struct{}

func (f *defaultSmokeTestPackageFinder) FindBuiltPackage(distPath, platform string) (string, error) {
	return packaging.FindBuiltPackage(distPath, platform)
}
