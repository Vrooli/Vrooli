package recorder

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestSanitizeOutcomeDOMTruncationAddsHash(t *testing.T) {
	longHTML := strings.Repeat("a", contracts.DOMSnapshotMaxBytes+10)
	out := sanitizeOutcome(contracts.StepOutcome{
		DOMSnapshot: &contracts.DOMSnapshot{
			HTML: longHTML,
		},
	})

	if out.DOMSnapshot == nil || !out.DOMSnapshot.Truncated {
		t.Fatalf("expected DOM snapshot to be truncated")
	}
	if len(out.DOMSnapshot.HTML) != contracts.DOMSnapshotMaxBytes {
		t.Fatalf("expected DOM snapshot length %d, got %d", contracts.DOMSnapshotMaxBytes, len(out.DOMSnapshot.HTML))
	}
	if out.DOMSnapshot.Hash == "" {
		t.Fatalf("expected hash to be set on truncated DOM")
	}
	if out.Notes["dom_truncated_hash"] != out.DOMSnapshot.Hash {
		t.Fatalf("expected dom_truncated_hash note to match hash")
	}
}

func TestSanitizeOutcomeConsoleTruncation(t *testing.T) {
	long := strings.Repeat("x", contracts.ConsoleEntryMaxBytes+5)
	now := time.Now()
	out := sanitizeOutcome(contracts.StepOutcome{
		ConsoleLogs: []contracts.ConsoleLogEntry{
			{Type: "log", Text: long, Timestamp: now},
		},
	})
	if len(out.ConsoleLogs) != 1 {
		t.Fatalf("expected one console log")
	}
	entry := out.ConsoleLogs[0]
	if !strings.HasSuffix(entry.Text, "[truncated]") {
		t.Fatalf("expected console text to be truncated, got %s", entry.Text)
	}
	if entry.Timestamp.Location() != time.UTC {
		t.Fatalf("expected console timestamp UTC, got %v", entry.Timestamp.Location())
	}
	if entry.Location == "" {
		t.Fatalf("expected truncation hash to be appended to location")
	}
}

func TestSanitizeOutcomeScreenshotClamping(t *testing.T) {
	oversized := make([]byte, contracts.ScreenshotMaxBytes+10)
	for i := range oversized {
		oversized[i] = 0x01
	}
	out := sanitizeOutcome(contracts.StepOutcome{
		Screenshot: &contracts.Screenshot{
			Data: oversized,
		},
	})
	if out.Screenshot == nil {
		t.Fatalf("expected screenshot to remain present")
	}
	if len(out.Screenshot.Data) != contracts.ScreenshotMaxBytes {
		t.Fatalf("expected screenshot to be clamped to %d bytes, got %d", contracts.ScreenshotMaxBytes, len(out.Screenshot.Data))
	}
	if out.Screenshot.MediaType == "" {
		t.Fatalf("expected default media type set")
	}
	if out.Screenshot.Width == 0 || out.Screenshot.Height == 0 {
		t.Fatalf("expected default dimensions set")
	}
	if out.Notes["screenshot_truncated"] == "" {
		t.Fatalf("expected screenshot_truncated note set")
	}
}

func TestSanitizeOutcomeNetworkTruncation(t *testing.T) {
	long := strings.Repeat("y", contracts.NetworkPayloadPreviewMaxBytes+10)
	now := time.Now()
	out := sanitizeOutcome(contracts.StepOutcome{
		Network: []contracts.NetworkEvent{
			{
				Type:                "request",
				URL:                 "https://example.com",
				RequestBodyPreview:  long,
				ResponseBodyPreview: long,
				Timestamp:           now,
			},
		},
	})
	if len(out.Network) != 1 {
		t.Fatalf("expected one network event")
	}
	ev := out.Network[0]
	if ev.Timestamp.Location() != time.UTC {
		t.Fatalf("expected network timestamp UTC, got %v", ev.Timestamp.Location())
	}
	if len(ev.RequestBodyPreview) != contracts.NetworkPayloadPreviewMaxBytes {
		t.Fatalf("expected request preview truncated to %d, got %d", contracts.NetworkPayloadPreviewMaxBytes, len(ev.RequestBodyPreview))
	}
	if len(ev.ResponseBodyPreview) != contracts.NetworkPayloadPreviewMaxBytes {
		t.Fatalf("expected response preview truncated to %d, got %d", contracts.NetworkPayloadPreviewMaxBytes, len(ev.ResponseBodyPreview))
	}
	if !ev.Truncated {
		t.Fatalf("expected truncated flag to be set")
	}
}

func TestRecordStepOutcomeTimelineParityWithFocusAndHighlights(t *testing.T) {
	repo := newRecorderTestRepo()
	rec := NewDBRecorder(repo, nil, nil)

	now := time.Now().UTC()
	outcome := contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    uuid.New(),
		StepIndex:      0,
		NodeID:         "node-1",
		StepType:       "screenshot",
		Success:        true,
		StartedAt:      now,
		CompletedAt:    ptr(time.Now().UTC()),
		DurationMs:     120,
		FinalURL:       "https://example.test/focused",
		Screenshot: &contracts.Screenshot{
			Data:        []byte{0x00, 0x01},
			MediaType:   "image/png",
			CaptureTime: now,
			Width:       320,
			Height:      180,
		},
		DOMSnapshot: &contracts.DOMSnapshot{
			HTML:        "<html><body><div id='hero'></div></body></html>",
			CollectedAt: now,
		},
		FocusedElement: &contracts.ElementFocus{
			Selector: "#hero",
			BoundingBox: &contracts.BoundingBox{
				X: 10, Y: 20, Width: 200, Height: 120,
			},
		},
		HighlightRegions: []contracts.HighlightRegion{
			{
				Selector: "#hero",
				BoundingBox: &contracts.BoundingBox{
					X: 10, Y: 20, Width: 200, Height: 120,
				},
				Padding: 12,
				Color:   "#ff00ff",
			},
		},
		MaskRegions: []contracts.MaskRegion{
			{
				Selector: ".mask",
				BoundingBox: &contracts.BoundingBox{
					X: 0, Y: 0, Width: 320, Height: 180,
				},
				Opacity: 0.5,
			},
		},
		ZoomFactor: 1.4,
		CursorTrail: []contracts.CursorPosition{
			{Point: contracts.Point{X: 1, Y: 2}, ElapsedMs: 5},
		},
		ConsoleLogs: []contracts.ConsoleLogEntry{{Type: "log", Text: "shot", Timestamp: now}},
		Network:     []contracts.NetworkEvent{{Type: "request", URL: "https://example.test", Timestamp: now}},
		Assertion:   &contracts.AssertionOutcome{Mode: "equals", Success: true, Message: "ok"},
	}

	plan := contracts.ExecutionPlan{
		ExecutionID: outcome.ExecutionID,
		WorkflowID:  uuid.New(),
		CreatedAt:   now,
	}

	result, err := rec.RecordStepOutcome(context.Background(), plan, outcome)
	if err != nil {
		t.Fatalf("record returned error: %v", err)
	}

	if result.TimelineArtifactID == nil {
		t.Fatalf("expected timeline artifact id")
	}

	var timeline *database.ExecutionArtifact
	for _, art := range repo.executionArtifacts {
		if art.ID == *result.TimelineArtifactID {
			timeline = art
			break
		}
	}
	if timeline == nil {
		t.Fatalf("timeline artifact not persisted")
	}

	payload := timeline.Payload
	expectKey := func(key string) {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected timeline payload to include %s, got %+v", key, payload)
		}
	}

	expectKey("focusedElement")
	expectKey("highlightRegions")
	expectKey("maskRegions")
	expectKey("cursorTrail")
	expectKey("domSnapshotArtifactId")
	expectKey("domSnapshotPreview")
	expectKey("artifactIds")
	expectKey("screenshotUrl")

	if url, _ := payload["screenshotUrl"].(string); url == "" || !strings.HasPrefix(url, "inline:") {
		t.Fatalf("expected inline screenshot url, got %v", payload["screenshotUrl"])
	}
	if zoom, _ := payload["zoomFactor"].(float64); zoom != 1.4 {
		t.Fatalf("expected zoomFactor 1.4, got %v", payload["zoomFactor"])
	}
	if url, _ := payload["finalUrl"].(string); url != outcome.FinalURL {
		t.Fatalf("expected finalUrl %s, got %v", outcome.FinalURL, payload["finalUrl"])
	}
}
func TestBuildTimelinePayloadPartial(t *testing.T) {
	outcome := contracts.StepOutcome{
		NodeID:    "node-1",
		StepType:  "click",
		Success:   false,
		StepIndex: 2,
		Failure: &contracts.StepFailure{
			Kind:    contracts.FailureKindEngine,
			Message: "boom",
		},
		DurationMs: 123,
		CursorTrail: []contracts.CursorPosition{
			{ElapsedMs: 5},
		},
	}

	domID := uuid.New()
	payload := buildTimelinePayload(outcome, "s3://shot.png", nil, &domID, "<html>", []uuid.UUID{uuid.New()})
	if payload["screenshotUrl"] != "s3://shot.png" {
		t.Fatalf("expected screenshotUrl to propagate, got %v", payload["screenshotUrl"])
	}
	if payload["partial"] != true {
		t.Fatalf("expected partial=true when failure present, got %v", payload["partial"])
	}
	if trail, ok := payload["cursorTrail"].([]contracts.CursorPosition); !ok || len(trail) != 1 {
		t.Fatalf("expected cursorTrail to be included, got %+v", payload["cursorTrail"])
	}
	if payload["stepIndex"] != 2 || payload["durationMs"] != 123 {
		t.Fatalf("expected stepIndex=2 duration=123, got %+v", payload)
	}
	if payload["domSnapshotArtifactId"] == "" {
		t.Fatalf("expected domSnapshotArtifactId to be set")
	}
	if payload["artifactIds"] == nil {
		t.Fatalf("expected artifactIds to be present")
	}
}

func TestDBRecorderPersistsCoreArtifactsAndTimeline(t *testing.T) {
	now := time.Now().UTC()
	execID := uuid.New()
	workflowID := uuid.New()

	repo := newRecorderTestRepo()
	rec := NewDBRecorder(repo, nil, nil)

	// Build a tiny PNG payload for inline screenshot storage.
	pngBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

	outcome := contracts.StepOutcome{
		ExecutionID: execID,
		StepIndex:   1,
		NodeID:      "node-1",
		StepType:    "click",
		Success:     true,
		StartedAt:   now,
		CompletedAt: &now,
		ConsoleLogs: []contracts.ConsoleLogEntry{{Type: "log", Text: "hello", Timestamp: now}},
		Network: []contracts.NetworkEvent{{
			Type:      "request",
			URL:       "https://example.com",
			Method:    "GET",
			Timestamp: now,
		}},
		Assertion: &contracts.AssertionOutcome{Success: true, Message: "ok"},
		ExtractedData: map[string]any{
			"value": "data",
		},
		Screenshot: &contracts.Screenshot{
			Data:        pngBytes,
			MediaType:   "image/png",
			CaptureTime: now,
			Hash:        "hash-1",
		},
		DOMSnapshot: &contracts.DOMSnapshot{
			HTML:        "<div>hi</div>",
			CollectedAt: now,
		},
	}

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		CreatedAt:   now,
	}

	if _, err := rec.RecordStepOutcome(nil, plan, outcome); err != nil {
		t.Fatalf("record returned error: %v", err)
	}

	if len(repo.executionSteps) != 1 {
		t.Fatalf("expected one execution step persisted, got %d", len(repo.executionSteps))
	}
	step := repo.executionSteps[0]
	if step.ExecutionID != execID || step.StepIndex != 1 || step.NodeID != "node-1" || step.Status != "completed" {
		t.Fatalf("unexpected step payload: %+v", step)
	}

	var types []string
	for _, art := range repo.executionArtifacts {
		types = append(types, art.ArtifactType)
	}

	expectedTypes := []string{
		"step_outcome",
		"console",
		"network",
		"assertion",
		"extracted_data",
		"screenshot_inline",
		"dom_snapshot",
		"timeline_frame",
	}
	for _, typ := range expectedTypes {
		if !containsType(types, typ) {
			t.Fatalf("expected artifact type %q, got %+v", typ, types)
		}
	}

	// Timeline frame should reference the inline screenshot label.
	var timelinePayload map[string]any
	for _, art := range repo.executionArtifacts {
		if art.ArtifactType == "timeline_frame" {
			timelinePayload = map[string]any(art.Payload)
			break
		}
	}
	if timelinePayload == nil {
		t.Fatalf("timeline_frame artifact not found")
	}
	if shot, ok := timelinePayload["screenshotUrl"].(string); !ok || shot == "" || !strings.HasPrefix(shot, "inline:") {
		t.Fatalf("expected inline screenshot url in timeline payload, got %v", timelinePayload["screenshotUrl"])
	}
	if idx, ok := timelinePayload["stepIndex"].(int); !ok || idx != 1 {
		t.Fatalf("expected stepIndex=1 in timeline payload, got %v", timelinePayload["stepIndex"])
	}
	if ids, ok := timelinePayload["artifactIds"].([]string); !ok || len(ids) == 0 {
		t.Fatalf("expected artifactIds present in timeline payload, got %v", timelinePayload["artifactIds"])
	}
}

func containsType(types []string, target string) bool {
	for _, typ := range types {
		if typ == target {
			return true
		}
	}
	return false
}

// recorderTestRepo is a lightweight repository stub that captures steps and artifacts.
type recorderTestRepo struct {
	executionSteps     []*database.ExecutionStep
	executionArtifacts []*database.ExecutionArtifact
}

func newRecorderTestRepo() *recorderTestRepo {
	return &recorderTestRepo{
		executionSteps:     []*database.ExecutionStep{},
		executionArtifacts: []*database.ExecutionArtifact{},
	}
}

// Project operations
func (m *recorderTestRepo) CreateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *recorderTestRepo) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return nil, database.ErrNotFound
}
func (m *recorderTestRepo) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, database.ErrNotFound
}
func (m *recorderTestRepo) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, database.ErrNotFound
}
func (m *recorderTestRepo) UpdateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *recorderTestRepo) DeleteProject(ctx context.Context, id uuid.UUID) error { return nil }
func (m *recorderTestRepo) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, nil
}
func (m *recorderTestRepo) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return nil, nil
}
func (m *recorderTestRepo) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	return nil, nil
}

// Workflow operations
func (m *recorderTestRepo) CreateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *recorderTestRepo) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, database.ErrNotFound
}
func (m *recorderTestRepo) GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error) {
	return nil, database.ErrNotFound
}
func (m *recorderTestRepo) GetWorkflowByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*database.Workflow, error) {
	return nil, database.ErrNotFound
}
func (m *recorderTestRepo) UpdateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *recorderTestRepo) DeleteWorkflow(ctx context.Context, id uuid.UUID) error { return nil }
func (m *recorderTestRepo) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return nil
}
func (m *recorderTestRepo) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *recorderTestRepo) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *recorderTestRepo) CreateWorkflowVersion(ctx context.Context, version *database.WorkflowVersion) error {
	return nil
}
func (m *recorderTestRepo) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*database.WorkflowVersion, error) {
	return nil, database.ErrNotFound
}
func (m *recorderTestRepo) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.WorkflowVersion, error) {
	return nil, nil
}

// Execution operations
func (m *recorderTestRepo) CreateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *recorderTestRepo) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return nil, database.ErrNotFound
}
func (m *recorderTestRepo) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *recorderTestRepo) DeleteExecution(ctx context.Context, id uuid.UUID) error { return nil }
func (m *recorderTestRepo) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}
func (m *recorderTestRepo) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	copyStep := *step
	if copyStep.ID == uuid.Nil {
		copyStep.ID = uuid.New()
	}
	m.executionSteps = append(m.executionSteps, &copyStep)
	step.ID = copyStep.ID
	return nil
}
func (m *recorderTestRepo) UpdateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *recorderTestRepo) ListExecutionSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	var steps []*database.ExecutionStep
	for _, step := range m.executionSteps {
		if step.ExecutionID == executionID {
			copyStep := *step
			steps = append(steps, &copyStep)
		}
	}
	return steps, nil
}
func (m *recorderTestRepo) CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error {
	copyArtifact := *artifact
	if copyArtifact.ID == uuid.Nil {
		copyArtifact.ID = uuid.New()
	}
	m.executionArtifacts = append(m.executionArtifacts, &copyArtifact)
	artifact.ID = copyArtifact.ID
	return nil
}

func ptr[T any](v T) *T {
	return &v
}
func (m *recorderTestRepo) ListExecutionArtifacts(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionArtifact, error) {
	var artifacts []*database.ExecutionArtifact
	for _, art := range m.executionArtifacts {
		if art.ExecutionID == executionID {
			copyArt := *art
			artifacts = append(artifacts, &copyArt)
		}
	}
	return artifacts, nil
}

// Screenshot operations
func (m *recorderTestRepo) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	return nil
}
func (m *recorderTestRepo) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}

// Log operations
func (m *recorderTestRepo) CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error {
	return nil
}
func (m *recorderTestRepo) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionLog, error) {
	return nil, nil
}

// Extracted data operations
func (m *recorderTestRepo) CreateExtractedData(ctx context.Context, data *database.ExtractedData) error {
	return nil
}
func (m *recorderTestRepo) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*database.ExtractedData, error) {
	return nil, nil
}

// Folder operations
func (m *recorderTestRepo) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	return nil
}
func (m *recorderTestRepo) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	return nil, database.ErrNotFound
}
func (m *recorderTestRepo) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	return nil, nil
}
