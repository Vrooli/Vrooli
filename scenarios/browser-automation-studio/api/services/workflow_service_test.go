package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

func TestExtractAIErrorMessage(t *testing.T) {
	msg := extractAIErrorMessage(map[string]any{
		"error": "Cannot generate workflow using placeholder domain",
	})

	if msg != "Cannot generate workflow using placeholder domain" {
		t.Fatalf("expected message to be preserved, got %q", msg)
	}

	nested := extractAIErrorMessage(map[string]any{
		"workflow": map[string]any{
			"error": "Nested error detected",
		},
	})

	if nested != "Nested error detected" {
		t.Fatalf("expected nested message, got %q", nested)
	}
}

func TestNormalizeFlowDefinitionReturnsAIWorkflowError(t *testing.T) {
	definition := map[string]any{
		"error": "Cannot generate workflow using placeholder domain",
	}

	_, err := normalizeFlowDefinition(definition)
	if err == nil {
		t.Fatalf("expected error for AI failure payload")
	}

	var aiErr *AIWorkflowError
	if !errors.As(err, &aiErr) {
		t.Fatalf("expected AIWorkflowError, got %T", err)
	}

	if aiErr.Reason == "" {
		t.Fatalf("expected AIWorkflowError to include a reason")
	}
}

func TestNormalizeFlowDefinitionStripsPreviewScreenshots(t *testing.T) {
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "node-1",
				"position": map[string]any{"x": 0, "y": 0},
				"data": map[string]any{
					"url":                         "https://example.com",
					"previewScreenshot":           "data:image/png;base64,AAAA",
					"previewScreenshotCapturedAt": time.Now().UTC().Format(time.RFC3339),
					"previewScreenshotSourceUrl":  "https://example.com",
					"preview": map[string]any{
						"screenshot": "data:image/png;base64,BBBB",
						"metadata":   "keep",
					},
				},
			},
		},
		"edges": []any{},
	}

	normalized, err := normalizeFlowDefinition(definition)
	if err != nil {
		t.Fatalf("expected normalization to succeed, got %v", err)
	}

	nodes := toInterfaceSlice(normalized["nodes"])
	if len(nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(nodes))
	}

	node, ok := nodes[0].(map[string]any)
	if !ok {
		if jsonNode, okJSON := nodes[0].(database.JSONMap); okJSON {
			node = map[string]any(jsonNode)
		} else {
			t.Fatalf("expected node to be a map, got %T", nodes[0])
		}
	}
	data, ok := node["data"].(map[string]any)
	if !ok {
		if jsonData, okJSON := node["data"].(database.JSONMap); okJSON {
			data = map[string]any(jsonData)
		} else {
			t.Fatalf("expected node data to be a map, got %T", node["data"])
		}
	}
	if _, exists := data["previewScreenshot"]; exists {
		t.Fatalf("expected previewScreenshot to be stripped")
	}
	if _, exists := data["previewScreenshotCapturedAt"]; exists {
		t.Fatalf("expected previewScreenshotCapturedAt to be stripped")
	}
	if _, exists := data["previewScreenshotSourceUrl"]; exists {
		t.Fatalf("expected previewScreenshotSourceUrl to be stripped")
	}
	previewValue, hasPreview := data["preview"].(map[string]any)
	if !hasPreview {
		t.Fatalf("expected preview map to remain with metadata")
	}
	if _, exists := previewValue["screenshot"]; exists {
		t.Fatalf("expected nested preview screenshot to be stripped")
	}
	if previewValue["metadata"] != "keep" {
		t.Fatalf("expected preview metadata to be preserved, got %v", previewValue["metadata"])
	}
}

func TestSanitizeWorkflowDefinitionHandlesJSONMapData(t *testing.T) {
	definition := database.JSONMap{
		"nodes": []any{
			database.JSONMap{
				"id": "node-json",
				"data": database.JSONMap{
					"selector":       "#login",
					"previewImage":   "data:image/png;base64,CCCC",
					"preview":        database.JSONMap{"image": "data:image/png;base64,DDDD", "note": "keep"},
					"other_metadata": "persist",
				},
			},
		},
	}

	sanitized := sanitizeWorkflowDefinition(definition)
	nodes := toInterfaceSlice(sanitized["nodes"])
	if len(nodes) != 1 {
		t.Fatalf("expected one node after sanitization, got %d", len(nodes))
	}
	node, ok := nodes[0].(map[string]any)
	if !ok {
		if jsonNode, okJSON := nodes[0].(database.JSONMap); okJSON {
			node = map[string]any(jsonNode)
		} else {
			t.Fatalf("expected node to be a map, got %T", nodes[0])
		}
	}
	data, ok := node["data"].(map[string]any)
	if !ok {
		if jsonData, okJSON := node["data"].(database.JSONMap); okJSON {
			data = map[string]any(jsonData)
		} else {
			t.Fatalf("expected data to be a map, got %T", node["data"])
		}
	}
	if _, exists := data["previewImage"]; exists {
		t.Fatalf("expected previewImage to be stripped")
	}
	if data["other_metadata"] != "persist" {
		t.Fatalf("expected other metadata to remain, got %v", data["other_metadata"])
	}
	previewValue, hasPreview := data["preview"].(map[string]any)
	if !hasPreview {
		t.Fatalf("expected preview map to remain after sanitization")
	}
	if _, exists := previewValue["image"]; exists {
		t.Fatalf("expected nested preview image to be stripped")
	}
	if previewValue["note"] != "keep" {
		t.Fatalf("expected non-image preview data to persist, got %v", previewValue["note"])
	}
}

type workflowUpdateRepoMock struct {
	timelineRepositoryMock
	project      *database.Project
	workflow     *database.Workflow
	updateCalls  []*database.Workflow
	versionCalls []*database.WorkflowVersion
	versions     []*database.WorkflowVersion
}

func (m *workflowUpdateRepoMock) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	if m.project != nil && m.project.ID == id {
		clone := *m.project
		return &clone, nil
	}
	return nil, database.ErrNotFound
}

func (m *workflowUpdateRepoMock) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	if m.workflow != nil && m.workflow.ID == id {
		return cloneWorkflow(m.workflow), nil
	}
	return nil, database.ErrNotFound
}

func (m *workflowUpdateRepoMock) GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error) {
	if m.workflow != nil && m.workflow.Name == name && m.workflow.FolderPath == folderPath {
		return cloneWorkflow(m.workflow), nil
	}
	return nil, database.ErrNotFound
}

func (m *workflowUpdateRepoMock) GetWorkflowByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*database.Workflow, error) {
	if m.workflow != nil && m.workflow.ProjectID != nil && *m.workflow.ProjectID == projectID && m.workflow.Name == name {
		return cloneWorkflow(m.workflow), nil
	}
	return nil, database.ErrNotFound
}

func (m *workflowUpdateRepoMock) UpdateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	clone := cloneWorkflow(workflow)
	m.workflow = clone
	m.updateCalls = append(m.updateCalls, clone)
	return nil
}

func (m *workflowUpdateRepoMock) CreateWorkflowVersion(ctx context.Context, version *database.WorkflowVersion) error {
	clone := *version
	if version.FlowDefinition != nil {
		clone.FlowDefinition = cloneJSONMap(version.FlowDefinition)
	}
	m.versionCalls = append(m.versionCalls, &clone)
	m.versions = append(m.versions, &clone)
	return nil
}

func (m *workflowUpdateRepoMock) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.WorkflowVersion, error) {
	if m.workflow == nil || m.workflow.ID != workflowID {
		return []*database.WorkflowVersion{}, nil
	}

	if len(m.versions) == 0 {
		return []*database.WorkflowVersion{}, nil
	}

	sorted := make([]*database.WorkflowVersion, len(m.versions))
	for i, version := range m.versions {
		clone := *version
		if version.FlowDefinition != nil {
			clone.FlowDefinition = cloneJSONMap(version.FlowDefinition)
		}
		sorted[i] = &clone
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Version > sorted[j].Version
	})

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(sorted)
	}
	if offset >= len(sorted) {
		return []*database.WorkflowVersion{}, nil
	}

	end := offset + limit
	if end > len(sorted) {
		end = len(sorted)
	}

	return sorted[offset:end], nil
}

func (m *workflowUpdateRepoMock) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*database.WorkflowVersion, error) {
	if m.workflow == nil || m.workflow.ID != workflowID {
		return nil, database.ErrNotFound
	}

	for _, entry := range m.versions {
		if entry.Version != version {
			continue
		}
		clone := *entry
		if entry.FlowDefinition != nil {
			clone.FlowDefinition = cloneJSONMap(entry.FlowDefinition)
		}
		return &clone, nil
	}

	return nil, database.ErrNotFound
}

func TestDescribeExecutionExportHandlesZeroFrameExecutions(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()

	repo := &timelineRepositoryMock{
		execution: &database.Execution{
			ID:         executionID,
			WorkflowID: workflowID,
			Status:     "failed",
			StartedAt:  time.Now().Add(-2 * time.Minute),
			CompletedAt: func() *time.Time {
				ts := time.Now().Add(-time.Minute)
				return &ts
			}(),
		},
		steps:     []*database.ExecutionStep{},
		artifacts: []*database.ExecutionArtifact{},
		logs:      []*database.ExecutionLog{},
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	svc := NewWorkflowService(repo, nil, nil, log)

	preview, err := svc.DescribeExecutionExport(context.Background(), executionID)
	if err != nil {
		t.Fatalf("DescribeExecutionExport returned error: %v", err)
	}

	if preview.Status != "unavailable" {
		t.Fatalf("expected status 'unavailable', got %q", preview.Status)
	}
	if preview.Package != nil {
		t.Fatalf("expected package to be nil for zero-frame export")
	}
	if !strings.Contains(strings.ToLower(preview.Message), "unavailable") {
		t.Fatalf("expected message to explain unavailability, got %q", preview.Message)
	}
	if preview.CapturedFrameCount != 0 {
		t.Fatalf("expected captured frame count 0, got %d", preview.CapturedFrameCount)
	}
	if preview.AvailableAssetCount != 0 {
		t.Fatalf("expected available asset count 0, got %d", preview.AvailableAssetCount)
	}
	if preview.TotalDurationMs != 0 {
		t.Fatalf("expected total duration 0, got %d", preview.TotalDurationMs)
	}
	if preview.SpecID != executionID.String() {
		t.Fatalf("expected spec id %s, got %s", executionID, preview.SpecID)
	}
}

func TestDescribeExecutionExportPendingForRunningExecutions(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()

	repo := &timelineRepositoryMock{
		execution: &database.Execution{
			ID:         executionID,
			WorkflowID: workflowID,
			Status:     "running",
			StartedAt:  time.Now().Add(-30 * time.Second),
		},
		steps:     []*database.ExecutionStep{},
		artifacts: []*database.ExecutionArtifact{},
		logs:      []*database.ExecutionLog{},
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	svc := NewWorkflowService(repo, nil, nil, log)

	preview, err := svc.DescribeExecutionExport(context.Background(), executionID)
	if err != nil {
		t.Fatalf("DescribeExecutionExport returned error: %v", err)
	}

	if preview.Status != "pending" {
		t.Fatalf("expected status 'pending', got %q", preview.Status)
	}
	if preview.Package != nil {
		t.Fatalf("expected package to be nil while replay is pending")
	}
	if preview.CapturedFrameCount != 0 {
		t.Fatalf("expected captured frame count 0, got %d", preview.CapturedFrameCount)
	}
	if preview.SpecID != executionID.String() {
		t.Fatalf("expected spec id %s, got %s", executionID, preview.SpecID)
	}
}

func TestStopExecutionCancelsRunner(t *testing.T) {
	executionID := uuid.New()
	repo := &timelineRepositoryMock{
		execution: &database.Execution{
			ID:          executionID,
			WorkflowID:  uuid.New(),
			Status:      "running",
			StartedAt:   time.Now().Add(-time.Minute),
			Progress:    50,
			CurrentStep: "Navigate",
		},
	}

	svc := newTestService(repo)
	wasCancelled := false
	svc.executionCancels.Store(executionID, context.CancelFunc(func() {
		wasCancelled = true
	}))

	if err := svc.StopExecution(context.Background(), executionID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !wasCancelled {
		t.Fatalf("expected cancel function to be invoked")
	}

	if repo.execution == nil {
		t.Fatalf("expected repository execution to be updated")
	}
	if repo.execution.Status != "cancelled" {
		t.Fatalf("expected status 'cancelled', got %q", repo.execution.Status)
	}
	if repo.execution.CompletedAt == nil {
		t.Fatalf("expected CompletedAt to be set")
	}
	if len(repo.logs) == 0 {
		t.Fatalf("expected cancellation log entry to be recorded")
	}
}

func TestDescribeExecutionExportReadyIncludesMetrics(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()

	repo := &timelineRepositoryMock{
		execution: &database.Execution{
			ID:         executionID,
			WorkflowID: workflowID,
			Status:     "completed",
			StartedAt:  time.Now().Add(-2 * time.Minute),
			CompletedAt: func() *time.Time {
				s := time.Now().Add(-time.Minute)
				return &s
			}(),
		},
		artifacts: []*database.ExecutionArtifact{
			{
				ArtifactType: "timeline_frame",
				Payload: database.JSONMap{
					"stepIndex":       0,
					"nodeId":          "step-1",
					"stepType":        "navigate",
					"status":          "completed",
					"success":         true,
					"durationMs":      1600,
					"start_offset_ms": 0,
				},
			},
		},
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	svc := NewWorkflowService(repo, nil, nil, log)

	preview, err := svc.DescribeExecutionExport(context.Background(), executionID)
	if err != nil {
		t.Fatalf("DescribeExecutionExport returned error: %v", err)
	}

	if preview.Status != "ready" {
		t.Fatalf("expected status 'ready', got %q", preview.Status)
	}
	if preview.Package == nil {
		t.Fatalf("expected replay package when ready")
	}
	if preview.CapturedFrameCount != 1 {
		t.Fatalf("expected captured frame count 1, got %d", preview.CapturedFrameCount)
	}
	if preview.AvailableAssetCount < 0 {
		t.Fatalf("expected non-negative asset count, got %d", preview.AvailableAssetCount)
	}
	if preview.TotalDurationMs <= 0 {
		t.Fatalf("expected positive total duration, got %d", preview.TotalDurationMs)
	}
	if preview.SpecID != executionID.String() {
		t.Fatalf("expected spec id %s, got %s", executionID, preview.SpecID)
	}
}

func (m *workflowUpdateRepoMock) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	if m.project == nil || projectID != m.project.ID || m.workflow == nil {
		return []*database.Workflow{}, nil
	}
	return []*database.Workflow{cloneWorkflow(m.workflow)}, nil
}

func (m *workflowUpdateRepoMock) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	if m.workflow != nil && m.workflow.ID == id {
		m.workflow = nil
	}
	return nil
}

func cloneWorkflow(wf *database.Workflow) *database.Workflow {
	if wf == nil {
		return nil
	}
	clone := *wf
	if wf.ProjectID != nil {
		id := *wf.ProjectID
		clone.ProjectID = &id
	}
	if wf.FlowDefinition != nil {
		clone.FlowDefinition = cloneJSONMapTest(wf.FlowDefinition)
	}
	if wf.Tags != nil {
		clone.Tags = append(pq.StringArray{}, wf.Tags...)
	}
	return &clone
}

func cloneJSONMapTest(m database.JSONMap) database.JSONMap {
	if m == nil {
		return nil
	}
	bytes, _ := json.Marshal(m)
	var out database.JSONMap
	_ = json.Unmarshal(bytes, &out)
	return out
}

func seedWorkflowFile(t *testing.T, project *database.Project, workflow *database.Workflow) {
	t.Helper()
	workflowsDir := filepath.Join(project.FolderPath, "workflows")
	if err := os.MkdirAll(workflowsDir, 0o755); err != nil {
		t.Fatalf("failed to create workflows dir: %v", err)
	}
	fileName := fmt.Sprintf("%s--%s.workflow.json", sanitizeWorkflowSlug(workflow.Name), shortID(workflow.ID))
	filePath := filepath.Join(workflowsDir, fileName)
	payload := map[string]any{
		"id":                 workflow.ID.String(),
		"name":               workflow.Name,
		"folder_path":        workflow.FolderPath,
		"description":        workflow.Description,
		"tags":               []string(workflow.Tags),
		"version":            workflow.Version,
		"flow_definition":    workflow.FlowDefinition,
		"nodes":              workflow.FlowDefinition["nodes"],
		"edges":              workflow.FlowDefinition["edges"],
		"change_description": workflow.LastChangeDescription,
		"source":             workflow.LastChangeSource,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal workflow payload: %v", err)
	}
	if err := os.WriteFile(filePath, bytes, 0o644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}
}

func newTestWorkflow(projectID uuid.UUID, version int) *database.Workflow {
	now := time.Now().UTC()
	nodes := []any{
		map[string]any{
			"id":   "node-1",
			"type": "navigate",
			"position": map[string]any{
				"x": float64(100),
				"y": float64(200),
			},
			"data": map[string]any{"url": "https://example.com"},
		},
	}
	return &database.Workflow{
		ID:                    uuid.New(),
		ProjectID:             &projectID,
		Name:                  "Demo Workflow",
		FolderPath:            "/",
		Description:           "Initial",
		FlowDefinition:        database.JSONMap{"nodes": nodes, "edges": []any{}},
		Version:               version,
		Tags:                  []string{},
		CreatedAt:             now,
		UpdatedAt:             now,
		LastChangeSource:      fileSourceManual,
		LastChangeDescription: "Initial workflow creation",
	}
}

func newTestService(repo database.Repository) *WorkflowService {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	hub := wsHub.NewHub(logger)
	go hub.Run()
	return &WorkflowService{
		repo:  repo,
		log:   logger,
		wsHub: hub,
	}
}

func TestUpdateWorkflowSkipsNoOpSaves(t *testing.T) {
	projectID := uuid.New()
	project := &database.Project{ID: projectID, Name: "Test Project", FolderPath: t.TempDir()}
	workflow := newTestWorkflow(projectID, 3)
	mock := &workflowUpdateRepoMock{project: project, workflow: workflow}
	seedWorkflowFile(t, project, workflow)
	svc := newTestService(mock)

	version := workflow.Version
	input := WorkflowUpdateInput{
		Name:            workflow.Name,
		Description:     workflow.Description,
		FolderPath:      workflow.FolderPath,
		FlowDefinition:  map[string]any{"nodes": workflow.FlowDefinition["nodes"], "edges": workflow.FlowDefinition["edges"]},
		ExpectedVersion: &version,
	}

	updated, err := svc.UpdateWorkflow(context.Background(), workflow.ID, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mock.updateCalls) != 0 {
		t.Fatalf("expected no database updates, recorded %d", len(mock.updateCalls))
	}
	if len(mock.versionCalls) != 0 {
		t.Fatalf("expected no version rows, recorded %d", len(mock.versionCalls))
	}
	if updated.Version != workflow.Version {
		t.Fatalf("expected version to remain %d, got %d", workflow.Version, updated.Version)
	}
}

func TestUpdateWorkflowAutosaveDefaultsExpectedVersion(t *testing.T) {
	t.Run("[REQ:BAS-VERSION-AUTOSAVE] autosave defaults expected version", func(t *testing.T) {
		projectID := uuid.New()
		project := &database.Project{ID: projectID, Name: "Autosave Project", FolderPath: t.TempDir()}
		workflow := newTestWorkflow(projectID, 5)
		mock := &workflowUpdateRepoMock{project: project, workflow: workflow}
		seedWorkflowFile(t, project, workflow)
		svc := newTestService(mock)

		input := WorkflowUpdateInput{
			Name:           workflow.Name,
			Description:    workflow.Description,
			FolderPath:     workflow.FolderPath,
			Source:         fileSourceAutosave,
			FlowDefinition: map[string]any{"nodes": []any{map[string]any{"id": "node-1", "type": "navigate", "position": map[string]any{"x": float64(150), "y": float64(220)}, "data": map[string]any{"url": "https://example.com/login"}}, map[string]any{"id": "node-2", "type": "click", "position": map[string]any{"x": float64(300), "y": float64(240)}, "data": map[string]any{"selector": "#sign-in"}}}, "edges": []any{map[string]any{"id": "edge-1", "source": "node-1", "target": "node-2"}}},
		}

		updated, err := svc.UpdateWorkflow(context.Background(), workflow.ID, input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(mock.updateCalls) != 1 {
			t.Fatalf("expected one database update, recorded %d", len(mock.updateCalls))
		}
		if len(mock.versionCalls) != 1 {
			t.Fatalf("expected one workflow version record, got %d", len(mock.versionCalls))
		}
		if updated.Version != workflow.Version+1 {
			t.Fatalf("expected version %d, got %d", workflow.Version+1, updated.Version)
		}
		if updated.LastChangeSource != fileSourceAutosave {
			t.Fatalf("expected last change source %q, got %q", fileSourceAutosave, updated.LastChangeSource)
		}
		if updated.LastChangeDescription != "Autosave" {
			t.Fatalf("expected change description 'Autosave', got %q", updated.LastChangeDescription)
		}
		if mock.versionCalls[0].CreatedBy != fileSourceAutosave {
			t.Fatalf("expected version created by %q, got %q", fileSourceAutosave, mock.versionCalls[0].CreatedBy)
		}
		if mock.versionCalls[0].ChangeDescription != "Autosave" {
			t.Fatalf("expected version change description 'Autosave', got %q", mock.versionCalls[0].ChangeDescription)
		}
	})
}

func TestUpdateWorkflowMetadataOnly(t *testing.T) {
	projectID := uuid.New()
	project := &database.Project{ID: projectID, Name: "Metadata Project", FolderPath: t.TempDir()}
	workflow := newTestWorkflow(projectID, 2)
	mock := &workflowUpdateRepoMock{project: project, workflow: workflow}
	seedWorkflowFile(t, project, workflow)
	svc := newTestService(mock)

	version := workflow.Version
	input := WorkflowUpdateInput{
		Name:            workflow.Name,
		Description:     "Updated description",
		ExpectedVersion: &version,
	}

	updated, err := svc.UpdateWorkflow(context.Background(), workflow.ID, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mock.updateCalls) != 1 {
		t.Fatalf("expected one database update, recorded %d", len(mock.updateCalls))
	}
	if len(mock.versionCalls) != 1 {
		t.Fatalf("expected one workflow version record, got %d", len(mock.versionCalls))
	}
	if updated.Description != "Updated description" {
		t.Fatalf("expected description to update, got %q", updated.Description)
	}
	if updated.LastChangeDescription != "Metadata update" {
		t.Fatalf("expected change description 'Metadata update', got %q", updated.LastChangeDescription)
	}
}

func TestEnsureWorkflowChangeMetadataBackfillsDefaults(t *testing.T) {
	projectID := uuid.New()
	project := &database.Project{ID: projectID, Name: "Backfill Project", FolderPath: t.TempDir()}
	workflow := newTestWorkflow(projectID, 7)
	workflow.LastChangeSource = ""
	workflow.LastChangeDescription = ""
	mock := &workflowUpdateRepoMock{project: project, workflow: workflow}
	svc := newTestService(mock)

	if err := svc.ensureWorkflowChangeMetadata(context.Background(), workflow); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mock.updateCalls) != 1 {
		t.Fatalf("expected one update call, got %d", len(mock.updateCalls))
	}
	updated := mock.updateCalls[0]
	if strings.TrimSpace(updated.LastChangeSource) == "" {
		t.Fatalf("expected last change source to be backfilled")
	}
	if strings.TrimSpace(updated.LastChangeDescription) == "" {
		t.Fatalf("expected last change description to be backfilled")
	}
}

func TestListWorkflowVersionsSummaries(t *testing.T) {
	t.Run("[REQ:BAS-VERSION-AUTOSAVE] lists workflow version summaries", func(t *testing.T) {
		projectID := uuid.New()
		project := &database.Project{ID: projectID, Name: "Version Project", FolderPath: t.TempDir()}
		workflow := newTestWorkflow(projectID, 3)
		workflow.ProjectID = &projectID

		baseTime := time.Now().Add(-2 * time.Hour)

		version1 := &database.WorkflowVersion{
			WorkflowID:        workflow.ID,
			Version:           1,
			FlowDefinition:    database.JSONMap{"nodes": []any{map[string]any{"id": "v1-node"}}, "edges": []any{}},
			ChangeDescription: "Initial import",
			CreatedBy:         "manual",
			CreatedAt:         baseTime,
		}
		version2 := &database.WorkflowVersion{
			WorkflowID:        workflow.ID,
			Version:           2,
			FlowDefinition:    database.JSONMap{"nodes": []any{map[string]any{"id": "v2-node-1"}, map[string]any{"id": "v2-node-2"}}, "edges": []any{}},
			ChangeDescription: "Added validation",
			CreatedBy:         "autosave",
			CreatedAt:         baseTime.Add(30 * time.Minute),
		}
		version3 := &database.WorkflowVersion{
			WorkflowID:        workflow.ID,
			Version:           3,
			FlowDefinition:    database.JSONMap{"nodes": []any{map[string]any{"id": "v3-node"}}, "edges": []any{map[string]any{"id": "edge-1"}}},
			ChangeDescription: "Refined selectors",
			CreatedBy:         "manual",
			CreatedAt:         baseTime.Add(90 * time.Minute),
		}

		mock := &workflowUpdateRepoMock{
			project:  project,
			workflow: workflow,
			versions: []*database.WorkflowVersion{version1, version2, version3},
		}

		svc := newTestService(mock)

		versions, err := svc.ListWorkflowVersions(context.Background(), workflow.ID, 10, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(versions) != 3 {
			t.Fatalf("expected 3 versions, got %d", len(versions))
		}

		if versions[0].Version != 3 {
			t.Fatalf("expected latest version first, got %d", versions[0].Version)
		}
		if versions[0].NodeCount != 1 || versions[0].EdgeCount != 1 {
			t.Fatalf("expected version 3 to have 1 node and 1 edge, got nodes=%d edges=%d", versions[0].NodeCount, versions[0].EdgeCount)
		}
		if strings.TrimSpace(versions[0].DefinitionHash) == "" {
			t.Fatalf("expected version hash to be populated")
		}
	})
}

func TestRestoreWorkflowVersionAppliesHistoricDefinition(t *testing.T) {
	t.Run("[REQ:BAS-VERSION-AUTOSAVE] [REQ:BAS-VERSION-RESTORE] restores historic version", func(t *testing.T) {
		projectID := uuid.New()
		project := &database.Project{ID: projectID, Name: "Restore Project", FolderPath: t.TempDir()}
		workflow := newTestWorkflow(projectID, 3)
		workflow.ProjectID = &projectID
		workflow.FlowDefinition = database.JSONMap{"nodes": []any{map[string]any{"id": "current"}}, "edges": []any{}}

		priorVersion := &database.WorkflowVersion{
			WorkflowID:        workflow.ID,
			Version:           2,
			FlowDefinition:    database.JSONMap{"nodes": []any{map[string]any{"id": "restored-node-1"}, map[string]any{"id": "restored-node-2"}}, "edges": []any{}},
			ChangeDescription: "Added extra navigation",
			CreatedBy:         "manual",
			CreatedAt:         time.Now().Add(-time.Minute),
		}

		mock := &workflowUpdateRepoMock{
			project:  project,
			workflow: workflow,
			versions: []*database.WorkflowVersion{priorVersion},
		}

		seedWorkflowFile(t, project, workflow)

		svc := newTestService(mock)
		if err := svc.syncProjectWorkflows(context.Background(), projectID); err != nil {
			t.Fatalf("expected sync to succeed, got %v", err)
		}
		if mock.workflow == nil {
			t.Fatalf("expected workflow to remain after sync")
		}
		originalVersion := mock.workflow.Version

		restored, err := svc.RestoreWorkflowVersion(context.Background(), workflow.ID, 2, "")
		if err != nil {
			t.Fatalf("expected no error restoring version, got %v", err)
		}

		if restored.Version != originalVersion+1 {
			t.Fatalf("expected restored version %d, got %d", originalVersion+1, restored.Version)
		}

		nodes, ok := restored.FlowDefinition["nodes"].([]any)
		if !ok || len(nodes) != 2 {
			t.Fatalf("expected restored workflow to contain 2 nodes, got %v", restored.FlowDefinition["nodes"])
		}

		if restored.LastChangeDescription != "Restored from version 2" {
			t.Fatalf("unexpected change description %q", restored.LastChangeDescription)
		}

		if len(mock.versionCalls) == 0 {
			t.Fatalf("expected a new workflow version to be recorded")
		}
		lastRecorded := mock.versionCalls[len(mock.versionCalls)-1]
		if lastRecorded.Version != restored.Version {
			t.Fatalf("expected recorded version %d, got %d", restored.Version, lastRecorded.Version)
		}
		if lastRecorded.ChangeDescription != "Restored from version 2" {
			t.Fatalf("expected recorded change description to note restore, got %q", lastRecorded.ChangeDescription)
		}

		if mock.workflow == nil || mock.workflow.Version != restored.Version {
			t.Fatalf("expected mock workflow to update to version %d", restored.Version)
		}

		restoredNodes, ok := mock.workflow.FlowDefinition["nodes"].([]any)
		if !ok || len(restoredNodes) != 2 {
			t.Fatalf("expected mock workflow definition to be replaced with restored nodes")
		}
	})
}
