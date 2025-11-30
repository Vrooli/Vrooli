package recorder

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestDBRecorderPersistsTraceVideoHarMeta(t *testing.T) {
	now := time.Now().UTC()
	execID := uuid.New()
	workflowID := uuid.New()
	tmpVideo := writeTempFile(t, "video", []byte("video-bytes"))
	tmpTrace := writeTempFile(t, "trace", []byte("trace-bytes"))
	tmpHAR := writeTempFile(t, "har", []byte("har-bytes"))

	repo := newRecorderTestRepo()
	rec := NewDBRecorder(repo, nil, nil)

	outcome := contracts.StepOutcome{
		ExecutionID: execID,
		StepIndex:   0,
		NodeID:      "n0",
		StepType:    "navigate",
		Success:     true,
		StartedAt:   now,
		CompletedAt: &now,
		Notes: map[string]string{
			"video_path": tmpVideo,
			"trace_path": tmpTrace,
			"har_path":   tmpHAR,
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

	var types []string
	meta := map[string]database.JSONMap{}
	for _, art := range repo.executionArtifacts {
		types = append(types, art.ArtifactType)
		if art.ArtifactType == "video_meta" || art.ArtifactType == "trace_meta" || art.ArtifactType == "har_meta" {
			meta[art.ArtifactType] = art.Payload
		}
	}
	for _, typ := range []string{"video_meta", "trace_meta", "har_meta"} {
		if !containsType(types, typ) {
			t.Fatalf("expected artifact type %q, got %+v", typ, types)
		}
		payload, ok := meta[typ]
		if !ok || payload == nil {
			t.Fatalf("expected payload for %s", typ)
		}
		if path, ok := payload["path"].(string); !ok || strings.TrimSpace(path) == "" {
			t.Fatalf("expected payload path for %s, got %+v", typ, payload)
		}
		if payload["inline"] != true && payload["base64"] == nil {
			t.Fatalf("expected inline/base64 for %s when small, got %+v", typ, payload)
		}
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

func writeTempFile(t *testing.T, prefix string, data []byte) string {
	t.Helper()
	tmp, err := os.CreateTemp("", prefix)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmp.Write(data); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	_ = tmp.Close()
	return tmp.Name()
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

// =============================================================================
// Nil/Empty Input Tests
// =============================================================================

func TestRecordStepOutcome_NilRecorder(t *testing.T) {
	var rec *DBRecorder
	result, err := rec.RecordStepOutcome(context.Background(), contracts.ExecutionPlan{}, contracts.StepOutcome{})
	require.NoError(t, err)
	assert.Empty(t, result.StepID)
}

func TestRecordStepOutcome_NilRepo(t *testing.T) {
	rec := &DBRecorder{repo: nil}
	result, err := rec.RecordStepOutcome(context.Background(), contracts.ExecutionPlan{}, contracts.StepOutcome{})
	require.NoError(t, err)
	assert.Empty(t, result.StepID)
}

func TestRecordTelemetry_NilRecorder(t *testing.T) {
	var rec *DBRecorder
	err := rec.RecordTelemetry(context.Background(), contracts.ExecutionPlan{}, contracts.StepTelemetry{})
	require.NoError(t, err)
}

func TestRecordTelemetry_NilRepo(t *testing.T) {
	rec := &DBRecorder{repo: nil}
	err := rec.RecordTelemetry(context.Background(), contracts.ExecutionPlan{}, contracts.StepTelemetry{})
	require.NoError(t, err)
}

func TestMarkCrash_NilRecorder(t *testing.T) {
	var rec *DBRecorder
	err := rec.MarkCrash(context.Background(), uuid.New(), contracts.StepFailure{})
	require.NoError(t, err)
}

func TestMarkCrash_NilRepo(t *testing.T) {
	rec := &DBRecorder{repo: nil}
	err := rec.MarkCrash(context.Background(), uuid.New(), contracts.StepFailure{})
	require.NoError(t, err)
}

// =============================================================================
// RecordTelemetry Tests
// =============================================================================

func TestRecordTelemetry_PersistsArtifact(t *testing.T) {
	repo := newRecorderTestRepo()
	rec := NewDBRecorder(repo, nil, nil)

	now := time.Now().UTC()
	execID := uuid.New()

	plan := contracts.ExecutionPlan{
		ExecutionID: execID,
		WorkflowID:  uuid.New(),
		CreatedAt:   now,
	}

	telemetry := contracts.StepTelemetry{
		SchemaVersion:  contracts.TelemetrySchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    execID,
		StepIndex:      0,
		Attempt:        1,
		Kind:           contracts.TelemetryKindHeartbeat,
		Timestamp:      now,
		ElapsedMs:      500,
		Heartbeat: &contracts.HeartbeatTelemetry{
			Progress: 50,
			Message:  "Waiting for element...",
		},
	}

	err := rec.RecordTelemetry(context.Background(), plan, telemetry)
	require.NoError(t, err)

	// Verify artifact was created
	require.Len(t, repo.executionArtifacts, 1)
	artifact := repo.executionArtifacts[0]
	assert.Equal(t, execID, artifact.ExecutionID)
	assert.Equal(t, "telemetry", artifact.ArtifactType)
	assert.NotNil(t, artifact.Payload["telemetry"])
}

// =============================================================================
// MarkCrash Tests
// =============================================================================

func TestMarkCrash_CreatesFailedStep(t *testing.T) {
	repo := newRecorderTestRepo()
	rec := NewDBRecorder(repo, nil, nil)

	execID := uuid.New()
	failure := contracts.StepFailure{
		Kind:    contracts.FailureKindEngine,
		Message: "Browser process crashed unexpectedly",
	}

	err := rec.MarkCrash(context.Background(), execID, failure)
	require.NoError(t, err)

	require.Len(t, repo.executionSteps, 1)
	step := repo.executionSteps[0]
	assert.Equal(t, execID, step.ExecutionID)
	assert.Equal(t, -1, step.StepIndex)
	assert.Equal(t, "crash", step.NodeID)
	assert.Equal(t, "crash", step.StepType)
	assert.Equal(t, "failed", step.Status)
	assert.Equal(t, "Browser process crashed unexpectedly", step.Error)
	assert.Equal(t, true, step.Metadata["partial"])
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestStatusFromOutcome(t *testing.T) {
	tests := []struct {
		name     string
		outcome  contracts.StepOutcome
		expected string
	}{
		{
			name:     "success returns completed",
			outcome:  contracts.StepOutcome{Success: true},
			expected: "completed",
		},
		{
			name:     "failure returns failed",
			outcome:  contracts.StepOutcome{Success: false},
			expected: "failed",
		},
		{
			name: "cancelled failure returns failed",
			outcome: contracts.StepOutcome{
				Success: false,
				Failure: &contracts.StepFailure{Kind: contracts.FailureKindCancelled},
			},
			expected: "failed",
		},
		{
			name: "engine failure returns failed",
			outcome: contracts.StepOutcome{
				Success: false,
				Failure: &contracts.StepFailure{Kind: contracts.FailureKindEngine},
			},
			expected: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusFromOutcome(tt.outcome)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeriveStepLabel(t *testing.T) {
	tests := []struct {
		name     string
		outcome  contracts.StepOutcome
		expected string
	}{
		{
			name:     "uses nodeID when present",
			outcome:  contracts.StepOutcome{NodeID: "click-btn-1", StepIndex: 0},
			expected: "click-btn-1",
		},
		{
			name:     "uses step index when nodeID empty",
			outcome:  contracts.StepOutcome{NodeID: "", StepIndex: 5},
			expected: "step-5",
		},
		{
			name:     "uses step index when nodeID whitespace",
			outcome:  contracts.StepOutcome{NodeID: "  ", StepIndex: 3},
			expected: "step-3",
		},
		{
			name:     "uses step for negative index and empty nodeID",
			outcome:  contracts.StepOutcome{NodeID: "", StepIndex: -1},
			expected: "step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveStepLabel(tt.outcome)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		limit    int
		expected string
	}{
		{"normal string within limit", "hello", 10, "hello"},
		{"normal string at limit", "hello", 5, "hello"},
		{"normal string exceeds limit", "hello", 3, "hel"},
		{"empty string", "", 5, ""},
		{"zero limit", "hello", 0, ""},
		{"negative limit", "hello", -1, ""},
		{"unicode string within limit", "日本語", 5, "日本語"},
		{"unicode string at limit", "日本語", 3, "日本語"},
		{"unicode string exceeds limit", "日本語", 2, "日本"},
		{"mixed unicode truncated correctly", "hello日本語", 7, "hello日本"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateRunes(tt.input, tt.limit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashString(t *testing.T) {
	hash1 := hashString("test content")
	hash2 := hashString("test content")
	hash3 := hashString("different content")

	// Same input produces same hash
	assert.Equal(t, hash1, hash2)
	// Different input produces different hash
	assert.NotEqual(t, hash1, hash3)
	// Hash is a valid hex string (64 chars for SHA-256)
	assert.Len(t, hash1, 64)
}

func TestAppendHash(t *testing.T) {
	tests := []struct {
		name     string
		location string
		hash     string
		expected string
	}{
		{
			name:     "empty location gets hash prefix",
			location: "",
			hash:     "abc123",
			expected: "hash:abc123",
		},
		{
			name:     "existing location gets hash appended",
			location: "file.js:10:5",
			hash:     "abc123",
			expected: "file.js:10:5 hash:abc123",
		},
		{
			name:     "empty hash returns location unchanged",
			location: "file.js:10:5",
			hash:     "",
			expected: "file.js:10:5",
		},
		{
			name:     "whitespace hash returns location unchanged",
			location: "file.js:10:5",
			hash:     "   ",
			expected: "file.js:10:5",
		},
		{
			name:     "both empty returns empty",
			location: "",
			hash:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendHash(tt.location, tt.hash)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToStringIDs(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	result := toStringIDs([]uuid.UUID{id1, id2})
	require.Len(t, result, 2)
	assert.Equal(t, id1.String(), result[0])
	assert.Equal(t, id2.String(), result[1])

	// Empty input
	result = toStringIDs([]uuid.UUID{})
	assert.Empty(t, result)
}

// =============================================================================
// Edge Cases for Sanitization
// =============================================================================

func TestSanitizeOutcome_EmptyOutcome(t *testing.T) {
	out := sanitizeOutcome(contracts.StepOutcome{})
	assert.NotNil(t, out.Notes, "Notes should be initialized")
}

func TestSanitizeOutcome_NilNotesInitialized(t *testing.T) {
	out := sanitizeOutcome(contracts.StepOutcome{Notes: nil})
	require.NotNil(t, out.Notes)
	assert.Empty(t, out.Notes)
}

func TestSanitizeOutcome_PreservesExistingNotes(t *testing.T) {
	out := sanitizeOutcome(contracts.StepOutcome{
		Notes: map[string]string{"key": "value"},
	})
	assert.Equal(t, "value", out.Notes["key"])
}

func TestSanitizeOutcome_ConsoleLogsLimitEnforced(t *testing.T) {
	// Create many console logs to test the limit
	logs := make([]contracts.ConsoleLogEntry, contracts.ConsoleEntryMaxBytes+10)
	for i := range logs {
		logs[i] = contracts.ConsoleLogEntry{
			Type:      "log",
			Text:      "message",
			Timestamp: time.Now(),
		}
	}

	out := sanitizeOutcome(contracts.StepOutcome{ConsoleLogs: logs})
	// Should be limited
	assert.LessOrEqual(t, len(out.ConsoleLogs), contracts.ConsoleEntryMaxBytes+1)
}

func TestSanitizeOutcome_NetworkEventsLimitEnforced(t *testing.T) {
	// Create many network events to test the limit
	events := make([]contracts.NetworkEvent, contracts.NetworkPayloadPreviewMaxBytes+10)
	for i := range events {
		events[i] = contracts.NetworkEvent{
			Type:      "request",
			URL:       "https://example.com",
			Timestamp: time.Now(),
		}
	}

	out := sanitizeOutcome(contracts.StepOutcome{Network: events})
	// Should be limited
	assert.LessOrEqual(t, len(out.Network), contracts.NetworkPayloadPreviewMaxBytes+1)
}

func TestSanitizeOutcome_ScreenshotDefaults(t *testing.T) {
	out := sanitizeOutcome(contracts.StepOutcome{
		Screenshot: &contracts.Screenshot{
			Data: []byte{0x01, 0x02},
			// No MediaType, Width, or Height set
		},
	})

	require.NotNil(t, out.Screenshot)
	assert.Equal(t, "image/png", out.Screenshot.MediaType)
	assert.Equal(t, contracts.DefaultScreenshotWidth, out.Screenshot.Width)
	assert.Equal(t, contracts.DefaultScreenshotHeight, out.Screenshot.Height)
}

// =============================================================================
// Error Recovery Tests
// =============================================================================

func TestNewDBRecorder(t *testing.T) {
	repo := newRecorderTestRepo()
	rec := NewDBRecorder(repo, nil, nil)
	require.NotNil(t, rec)
	assert.Equal(t, repo, rec.repo)
}

// errorTestRepo is a test repository that can simulate errors
type errorTestRepo struct {
	*recorderTestRepo
	createStepErr     error
	createArtifactErr error
}

func newErrorTestRepo() *errorTestRepo {
	return &errorTestRepo{
		recorderTestRepo: newRecorderTestRepo(),
	}
}

func (e *errorTestRepo) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	if e.createStepErr != nil {
		return e.createStepErr
	}
	return e.recorderTestRepo.CreateExecutionStep(ctx, step)
}

func (e *errorTestRepo) CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error {
	if e.createArtifactErr != nil {
		return e.createArtifactErr
	}
	return e.recorderTestRepo.CreateExecutionArtifact(ctx, artifact)
}

func TestRecordStepOutcome_CreateStepError(t *testing.T) {
	repo := newErrorTestRepo()
	repo.createStepErr = errors.New("database connection lost")
	rec := NewDBRecorder(repo, nil, nil)

	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
	}
	outcome := contracts.StepOutcome{
		StepIndex: 0,
		NodeID:    "test-node",
		StepType:  "click",
		Success:   true,
	}

	_, err := rec.RecordStepOutcome(context.Background(), plan, outcome)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection lost")
}

func TestRecordStepOutcome_ArtifactErrorNonFatal(t *testing.T) {
	repo := newErrorTestRepo()
	repo.createArtifactErr = errors.New("artifact storage failed")
	rec := NewDBRecorder(repo, nil, nil)

	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
	}
	outcome := contracts.StepOutcome{
		StepIndex: 0,
		NodeID:    "test-node",
		StepType:  "click",
		Success:   true,
	}

	// Artifact errors should be non-fatal
	_, err := rec.RecordStepOutcome(context.Background(), plan, outcome)
	require.NoError(t, err)

	// Step should still be created
	assert.Len(t, repo.executionSteps, 1)
}

// =============================================================================
// BuildTimelinePayload Tests
// =============================================================================

func TestBuildTimelinePayload_MinimalOutcome(t *testing.T) {
	outcome := contracts.StepOutcome{
		StepIndex: 0,
		NodeID:    "nav-1",
		StepType:  "navigate",
		Success:   true,
	}

	payload := buildTimelinePayload(outcome, "", nil, nil, "", nil)
	assert.Equal(t, 0, payload["stepIndex"])
	assert.Equal(t, "nav-1", payload["nodeId"])
	assert.Equal(t, "navigate", payload["stepType"])
	assert.Equal(t, true, payload["success"])
}

func TestBuildTimelinePayload_FailureIncludesError(t *testing.T) {
	outcome := contracts.StepOutcome{
		StepIndex: 1,
		NodeID:    "click-1",
		StepType:  "click",
		Success:   false,
		Failure: &contracts.StepFailure{
			Kind:    contracts.FailureKindTimeout,
			Message: "Element not found within timeout",
		},
	}

	payload := buildTimelinePayload(outcome, "", nil, nil, "", nil)
	assert.Equal(t, true, payload["partial"])
	assert.Equal(t, "Element not found within timeout", payload["error"])
}

func TestBuildTimelinePayload_AllOptionalFields(t *testing.T) {
	now := time.Now().UTC()
	screenshotID := uuid.New()
	domID := uuid.New()

	outcome := contracts.StepOutcome{
		StepIndex:   2,
		NodeID:      "assert-1",
		StepType:    "assert",
		Success:     true,
		Attempt:     1,
		DurationMs:  250,
		StartedAt:   now,
		CompletedAt: &now,
		FinalURL:    "https://example.com/page",
		ElementBoundingBox: &contracts.BoundingBox{
			X: 100, Y: 200, Width: 50, Height: 30,
		},
		ClickPosition: &contracts.Point{X: 125, Y: 215},
		FocusedElement: &contracts.ElementFocus{
			Selector: "#target",
		},
		HighlightRegions: []contracts.HighlightRegion{
			{Selector: ".highlight", Color: "#ff0000"},
		},
		MaskRegions: []contracts.MaskRegion{
			{Selector: ".mask", Opacity: 0.5},
		},
		ZoomFactor: 1.5,
		ExtractedData: map[string]any{
			"value": "extracted",
		},
		Assertion: &contracts.AssertionOutcome{
			Success: true,
			Message: "Assertion passed",
		},
		ConsoleLogs: []contracts.ConsoleLogEntry{{Type: "log", Text: "test"}},
		Network:     []contracts.NetworkEvent{{Type: "request", URL: "https://api.example.com"}},
		CursorTrail: []contracts.CursorPosition{{ElapsedMs: 10}},
	}

	artifactIDs := []uuid.UUID{uuid.New(), uuid.New()}
	payload := buildTimelinePayload(outcome, "s3://bucket/screenshot.png", &screenshotID, &domID, "<html>preview</html>", artifactIDs)

	// Verify all fields are populated
	assert.Equal(t, "s3://bucket/screenshot.png", payload["screenshotUrl"])
	assert.Equal(t, screenshotID.String(), payload["screenshotArtifactId"])
	assert.Equal(t, domID.String(), payload["domSnapshotArtifactId"])
	assert.Equal(t, "<html>preview</html>", payload["domSnapshotPreview"])
	assert.Equal(t, "https://example.com/page", payload["finalUrl"])
	assert.NotNil(t, payload["elementBoundingBox"])
	assert.NotNil(t, payload["clickPosition"])
	assert.NotNil(t, payload["focusedElement"])
	assert.NotNil(t, payload["highlightRegions"])
	assert.NotNil(t, payload["maskRegions"])
	assert.Equal(t, 1.5, payload["zoomFactor"])
	assert.NotNil(t, payload["extractedDataPreview"])
	assert.NotNil(t, payload["assertion"])
	assert.Equal(t, 1, payload["consoleLogCount"])
	assert.Equal(t, 1, payload["networkEventCount"])
	assert.NotNil(t, payload["cursorTrail"])
	assert.Len(t, payload["artifactIds"], 2)
}
