package export

import (
	"context"
	"io"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
)

type timelineRepositoryMock struct {
	execution        *database.Execution
	steps            []*database.ExecutionStep
	artifacts        []*database.ExecutionArtifact
	logs             []*database.ExecutionLog
	executionUpdates []*database.Execution
}

// Project operations
func (m *timelineRepositoryMock) CreateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *timelineRepositoryMock) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return nil, database.ErrNotFound
}
func (m *timelineRepositoryMock) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, database.ErrNotFound
}
func (m *timelineRepositoryMock) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, database.ErrNotFound
}
func (m *timelineRepositoryMock) UpdateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *timelineRepositoryMock) DeleteProject(ctx context.Context, id uuid.UUID) error { return nil }
func (m *timelineRepositoryMock) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return nil
}
func (m *timelineRepositoryMock) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, nil
}
func (m *timelineRepositoryMock) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return map[string]any{}, nil
}
func (m *timelineRepositoryMock) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	return map[uuid.UUID]*database.ProjectStats{}, nil
}

// Workflow operations
func (m *timelineRepositoryMock) CreateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *timelineRepositoryMock) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, database.ErrNotFound
}
func (m *timelineRepositoryMock) GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error) {
	return nil, database.ErrNotFound
}
func (m *timelineRepositoryMock) GetWorkflowByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*database.Workflow, error) {
	return nil, database.ErrNotFound
}
func (m *timelineRepositoryMock) UpdateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *timelineRepositoryMock) DeleteWorkflow(ctx context.Context, id uuid.UUID) error { return nil }
func (m *timelineRepositoryMock) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *timelineRepositoryMock) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *timelineRepositoryMock) CreateWorkflowVersion(ctx context.Context, version *database.WorkflowVersion) error {
	return nil
}
func (m *timelineRepositoryMock) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*database.WorkflowVersion, error) {
	return nil, nil
}
func (m *timelineRepositoryMock) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.WorkflowVersion, error) {
	return nil, nil
}

// Execution operations
func (m *timelineRepositoryMock) CreateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *timelineRepositoryMock) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	if m.execution != nil && m.execution.ID == id {
		return m.execution, nil
	}
	return nil, database.ErrNotFound
}
func (m *timelineRepositoryMock) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	clone := *execution
	m.execution = &clone
	m.executionUpdates = append(m.executionUpdates, &clone)
	return nil
}
func (m *timelineRepositoryMock) DeleteExecution(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *timelineRepositoryMock) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}
func (m *timelineRepositoryMock) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *timelineRepositoryMock) UpdateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *timelineRepositoryMock) ListExecutionSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	return m.steps, nil
}
func (m *timelineRepositoryMock) CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error {
	return nil
}
func (m *timelineRepositoryMock) ListExecutionArtifacts(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionArtifact, error) {
	return m.artifacts, nil
}

// Screenshot operations
func (m *timelineRepositoryMock) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	return nil
}
func (m *timelineRepositoryMock) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}

// Log operations
func (m *timelineRepositoryMock) CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error {
	clone := *log
	m.logs = append(m.logs, &clone)
	return nil
}
func (m *timelineRepositoryMock) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionLog, error) {
	return m.logs, nil
}

// Extracted data operations
func (m *timelineRepositoryMock) CreateExtractedData(ctx context.Context, data *database.ExtractedData) error {
	return nil
}
func (m *timelineRepositoryMock) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*database.ExtractedData, error) {
	return nil, nil
}

// Folder operations
func (m *timelineRepositoryMock) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	return nil
}
func (m *timelineRepositoryMock) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	return nil, database.ErrNotFound
}
func (m *timelineRepositoryMock) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	return nil, nil
}

func TestGetExecutionTimeline(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-TIMELINE-PERSISTENCE] builds timeline from execution artifacts", func(t *testing.T) {
		executionID := uuid.New()
		workflowID := uuid.New()
		stepID := uuid.New()

		startedAt := time.Now().Add(-2 * time.Minute).UTC()
		completedAt := startedAt.Add(1500 * time.Millisecond)

		execution := &database.Execution{
			ID:          executionID,
			WorkflowID:  workflowID,
			Status:      "completed",
			Progress:    100,
			StartedAt:   startedAt,
			CompletedAt: &completedAt,
		}

		step := &database.ExecutionStep{
			ID:          stepID,
			ExecutionID: executionID,
			StepIndex:   0,
			NodeID:      "node-1",
			StepType:    "screenshot",
			Status:      "completed",
			StartedAt:   startedAt,
			CompletedAt: &completedAt,
			DurationMs:  1500,
		}

		screenshotID := uuid.New()
		consoleID := uuid.New()
		domID := uuid.New()

		screenshotArtifact := &database.ExecutionArtifact{
			ID:           screenshotID,
			ExecutionID:  executionID,
			ArtifactType: "screenshot",
			StorageURL:   "https://storage/screenshot.png",
			ThumbnailURL: "https://storage/thumb.png",
			ContentType:  "image/png",
			Payload: database.JSONMap{
				"width":  1280,
				"height": 720,
			},
		}

		consoleArtifact := &database.ExecutionArtifact{
			ID:           consoleID,
			ExecutionID:  executionID,
			ArtifactType: "console",
			Payload: database.JSONMap{
				"entries": []map[string]any{{
					"type": "log",
					"text": "hello",
				}},
			},
		}

		domArtifact := &database.ExecutionArtifact{
			ID:           domID,
			ExecutionID:  executionID,
			ArtifactType: "dom_snapshot",
			Payload: database.JSONMap{
				"html": "<html><body>snapshot</body></html>",
			},
		}

		timelineArtifact := &database.ExecutionArtifact{
			ID:           uuid.New(),
			ExecutionID:  executionID,
			ArtifactType: "timeline_frame",
			Payload: database.JSONMap{
				"stepIndex":          0,
				"nodeId":             "node-1",
				"stepType":           "screenshot",
				"success":            true,
				"durationMs":         1500,
				"totalDurationMs":    1800,
				"progress":           100,
				"finalUrl":           "https://example.com",
				"consoleLogCount":    1,
				"networkEventCount":  0,
				"highlightRegions":   []map[string]any{{"selector": "#cta", "boundingBox": map[string]any{"x": 10, "y": 20, "width": 200, "height": 80}}},
				"maskRegions":        []map[string]any{{"selector": ".private", "opacity": 0.45}},
				"focusedElement":     map[string]any{"selector": "#cta"},
				"elementBoundingBox": map[string]any{"x": 0, "y": 0, "width": 300, "height": 200},
				"clickPosition":      map[string]any{"x": 150, "y": 90},
				"cursorTrail": []map[string]any{
					{"x": 120, "y": 70},
					{"x": 150, "y": 90},
				},
				"zoomFactor":         1.2,
				"retryAttempt":       2,
				"retryMaxAttempts":   3,
				"retryConfigured":    2,
				"retryDelayMs":       250,
				"retryBackoffFactor": 1.5,
				"retryHistory": []map[string]any{
					{"attempt": 1, "success": false, "durationMs": 700, "error": "timeout"},
					{"attempt": 2, "success": true, "durationMs": 1100},
				},
				"artifactIds":           []any{screenshotID.String(), consoleID.String(), domID.String()},
				"screenshotArtifactId":  screenshotID.String(),
				"domSnapshotPreview":    "<html><body>snapshot</body></html>",
				"domSnapshotArtifactId": domID.String(),
				"executionStepId":       stepID.String(),
				"startedAt":             startedAt.Format(time.RFC3339Nano),
				"completedAt":           completedAt.Format(time.RFC3339Nano),
				"extractedDataPreview":  map[string]any{"value": "Sample"},
				"assertion": map[string]any{
					"mode":     "exists",
					"selector": "#cta",
					"success":  true,
				},
			},
		}

		logID := uuid.New()
		executionLog := &database.ExecutionLog{
			ID:          logID,
			ExecutionID: executionID,
			Timestamp:   startedAt.Add(250 * time.Millisecond),
			Level:       "INFO",
			StepName:    "navigate",
			Message:     "navigate started",
		}

		repo := &timelineRepositoryMock{
			execution: execution,
			steps:     []*database.ExecutionStep{step},
			artifacts: []*database.ExecutionArtifact{consoleArtifact, screenshotArtifact, domArtifact, timelineArtifact},
			logs:      []*database.ExecutionLog{executionLog},
		}

		log := logrus.New()
		log.SetOutput(io.Discard)

		svc := workflow.NewWorkflowService(repo, nil, log)

		timeline, err := svc.GetExecutionTimeline(context.Background(), executionID)
		if err != nil {
			t.Fatalf("GetExecutionTimeline returned error: %v", err)
		}

		if timeline == nil {
			t.Fatalf("expected timeline result")
		}

		if timeline.ExecutionID != executionID {
			t.Fatalf("unexpected execution id: %s", timeline.ExecutionID)
		}

		if timeline.WorkflowID != workflowID {
			t.Fatalf("unexpected workflow id: %s", timeline.WorkflowID)
		}

		if len(timeline.Frames) != 1 {
			t.Fatalf("expected 1 frame, got %d", len(timeline.Frames))
		}

		if len(timeline.Logs) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(timeline.Logs))
		}

		if timeline.Logs[0].ID != logID.String() {
			t.Fatalf("unexpected log id: %s", timeline.Logs[0].ID)
		}
		if timeline.Logs[0].Level != "info" {
			t.Fatalf("expected log level info, got %s", timeline.Logs[0].Level)
		}
		if timeline.Logs[0].StepName != "navigate" {
			t.Fatalf("unexpected log step name: %s", timeline.Logs[0].StepName)
		}
		if timeline.Logs[0].Message != "navigate started" {
			t.Fatalf("unexpected log message: %s", timeline.Logs[0].Message)
		}

		frame := timeline.Frames[0]

		if frame.StepIndex != 0 {
			t.Fatalf("unexpected step index: %d", frame.StepIndex)
		}

		if frame.Status != "completed" {
			t.Fatalf("expected status completed, got %s", frame.Status)
		}

		if frame.Screenshot == nil {
			t.Fatalf("expected screenshot metadata")
		}

		if frame.Screenshot.ArtifactID != screenshotID.String() {
			t.Fatalf("unexpected screenshot artifact id: %s", frame.Screenshot.ArtifactID)
		}

		if frame.ZoomFactor != 1.2 {
			t.Fatalf("expected zoom factor 1.2, got %f", frame.ZoomFactor)
		}

		if len(frame.HighlightRegions) != 1 || frame.HighlightRegions[0].Selector != "#cta" {
			t.Fatalf("unexpected highlight regions: %+v", frame.HighlightRegions)
		}

		if len(frame.CursorTrail) != 2 {
			t.Fatalf("expected cursor trail of length 2, got %d", len(frame.CursorTrail))
		}

		if frame.CursorTrail[1].X != 150 || frame.CursorTrail[1].Y != 90 {
			t.Fatalf("unexpected cursor trail coordinates: %+v", frame.CursorTrail)
		}

		if frame.Assertion == nil || !frame.Assertion.Success || frame.Assertion.Selector != "#cta" {
			t.Fatalf("expected assertion metadata, got %+v", frame.Assertion)
		}

		if len(frame.Artifacts) != 3 {
			t.Fatalf("expected three artifact references, got %d", len(frame.Artifacts))
		}

		if frame.Artifacts[0].ID != screenshotID.String() {
			t.Fatalf("unexpected first artifact id: %s", frame.Artifacts[0].ID)
		}

		if frame.DomSnapshotPreview == "" {
			t.Fatalf("expected DOM snapshot preview to be populated")
		}
		if frame.DomSnapshot == nil || frame.DomSnapshot.ID != domID.String() {
			t.Fatalf("expected DOM snapshot artifact reference, got %+v", frame.DomSnapshot)
		}

		if frame.ConsoleLogCount != 1 {
			t.Fatalf("expected console count 1, got %d", frame.ConsoleLogCount)
		}

		if frame.ExtractedDataPreview == nil {
			t.Fatalf("expected extracted data preview")
		}

		if frame.TotalDurationMs != 1800 {
			t.Fatalf("expected total duration 1800, got %d", frame.TotalDurationMs)
		}
		if frame.RetryAttempt != 2 {
			t.Fatalf("expected retry attempt 2, got %d", frame.RetryAttempt)
		}
		if frame.RetryMaxAttempts != 3 {
			t.Fatalf("expected retry max attempts 3, got %d", frame.RetryMaxAttempts)
		}
		if frame.RetryConfigured != 2 {
			t.Fatalf("expected retry configured 2, got %d", frame.RetryConfigured)
		}
		if frame.RetryDelayMs != 250 {
			t.Fatalf("expected retry delay 250, got %d", frame.RetryDelayMs)
		}
		if math.Abs(frame.RetryBackoffFactor-1.5) > 0.0001 {
			t.Fatalf("expected retry backoff 1.5, got %f", frame.RetryBackoffFactor)
		}
		if len(frame.RetryHistory) != 2 {
			t.Fatalf("expected retry history length 2, got %d", len(frame.RetryHistory))
		}
		if frame.RetryHistory[0].Attempt != 1 || frame.RetryHistory[0].Success {
			t.Fatalf("unexpected retry history entry: %+v", frame.RetryHistory[0])
		}
		if frame.RetryHistory[1].Attempt != 2 || !frame.RetryHistory[1].Success {
			t.Fatalf("unexpected retry history entry: %+v", frame.RetryHistory[1])
		}
	})
}
