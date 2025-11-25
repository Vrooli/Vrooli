package services

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

const (
	defaultRecordingWorkflowFolder = "/recordings"
	defaultRecordingWorkflowName   = "Extension Recording"

	maxRecordingFrames              = 400
	maxRecordingArchiveBytes        = 200 * 1024 * 1024 // 200MB
	maxRecordingAssetBytes          = 12 * 1024 * 1024  // 12MB per frame
	recordingDefaultFrameDurationMs = 1600
)

var (
	errManifestMissingFrames = errors.New("recording manifest did not include any frames")
	errManifestTooLarge      = errors.New("recording archive exceeds maximum allowed size")
	errTooManyFrames         = fmt.Errorf("recording exceeds maximum frame count (%d)", maxRecordingFrames)
)

// Exported error aliases for handler detection.
var (
	ErrRecordingManifestMissingFrames = errManifestMissingFrames
	ErrRecordingArchiveTooLarge       = errManifestTooLarge
	ErrRecordingTooManyFrames         = errTooManyFrames
)

// RecordingRepository captures the repository functionality required for recording ingestion.
type RecordingRepository interface {
	GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error)
	GetProjectByName(ctx context.Context, name string) (*database.Project, error)
	CreateProject(ctx context.Context, project *database.Project) error
	DeleteProject(ctx context.Context, id uuid.UUID) error

	GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error)
	GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error)
	CreateWorkflow(ctx context.Context, workflow *database.Workflow) error
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error

	CreateExecution(ctx context.Context, execution *database.Execution) error
	UpdateExecution(ctx context.Context, execution *database.Execution) error
	DeleteExecution(ctx context.Context, id uuid.UUID) error

	CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error
	UpdateExecutionStep(ctx context.Context, step *database.ExecutionStep) error
	CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error
	CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error
}

// RecordingImportOptions capture optional metadata supplied during ingestion.
type RecordingImportOptions struct {
	ProjectID    *uuid.UUID
	ProjectName  string
	WorkflowID   *uuid.UUID
	WorkflowName string
}

// RecordingImportResult summarises the outcome of a recording import.
type RecordingImportResult struct {
	Execution  *database.Execution `json:"execution"`
	Workflow   *database.Workflow  `json:"workflow"`
	Project    *database.Project   `json:"project"`
	FrameCount int                 `json:"frame_count"`
	AssetCount int                 `json:"asset_count"`
	DurationMs int                 `json:"duration_ms"`
}

// RecordingService ingests Chrome extension recording archives and normalises them into execution artifacts.
type RecordingService struct {
	repo           RecordingRepository
	storage        storage.StorageInterface
	recorder       autorecorder.Recorder
	log            *logrus.Logger
	recordingsRoot string
}

// NewRecordingService constructs a recording ingestion service.
func NewRecordingService(repo RecordingRepository, storageClient storage.StorageInterface, _ *wsHub.Hub, log *logrus.Logger, recordingsRoot string) *RecordingService {
	absRoot := recordingsRoot
	if absRoot == "" {
		absRoot = filepath.Join("scenarios", "browser-automation-studio", "data", "recordings")
	}
	if resolved, err := filepath.Abs(absRoot); err == nil {
		absRoot = resolved
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil && log != nil {
		log.WithError(err).Warn("Failed to ensure recordings root directory exists")
	}

	store := selectRecordingStorage(storageClient, absRoot, log)

	return &RecordingService{
		repo:           repo,
		storage:        store,
		recorder:       autorecorder.NewDBRecorder(repo, store, log),
		log:            log,
		recordingsRoot: absRoot,
	}
}

// ImportArchive ingests a Chrome extension recording archive located on disk.
func (s *RecordingService) ImportArchive(ctx context.Context, archivePath string, opts RecordingImportOptions) (*RecordingImportResult, error) {
	if s == nil {
		return nil, errors.New("recording service not configured")
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("recording archive unavailable: %w", err)
	}

	if info.Size() > maxRecordingArchiveBytes {
		return nil, errManifestTooLarge
	}

	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open recording archive: %w", err)
	}
	defer zr.Close()

	manifest, err := loadRecordingManifest(&zr.Reader)
	if err != nil {
		return nil, err
	}

	if len(manifest.Frames) == 0 {
		return nil, errManifestMissingFrames
	}

	if len(manifest.Frames) > maxRecordingFrames {
		return nil, errTooManyFrames
	}

	project, projectCreated, err := s.resolveProject(ctx, manifest, opts)
	if err != nil {
		return nil, err
	}

	workflow, workflowCreated, err := s.resolveWorkflow(ctx, manifest, project, opts)
	if err != nil {
		if projectCreated {
			s.cleanupRecordingArtifacts(context.Background(), true, false, project, nil, nil)
		}
		return nil, err
	}

	execution := &database.Execution{
		ID:          uuid.New(),
		WorkflowID:  workflow.ID,
		Status:      "running",
		TriggerType: "extension",
		TriggerMetadata: database.JSONMap{
			"source":     "chrome_extension",
			"runId":      manifest.RunID,
			"frameCount": len(manifest.Frames),
			"recordedAt": manifest.RecordedAt,
			"viewport":   manifest.Viewport,
			"extension":  manifest.Extension,
			"importedAt": time.Now().UTC().Format(time.RFC3339),
		},
		StartedAt: time.Now().UTC(),
		Progress:  0,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		s.cleanupRecordingArtifacts(context.Background(), projectCreated, workflowCreated, project, workflow, nil)
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	frameResult, err := s.persistFrames(ctx, &zr.Reader, project, workflow, execution, manifest)
	if err != nil {
		s.cleanupRecordingArtifacts(context.Background(), projectCreated, workflowCreated, project, workflow, execution)
		return nil, err
	}

	execution.Status = "completed"
	execution.Progress = 100
	completedAt := execution.StartedAt.Add(time.Duration(frameResult.totalDurationMs) * time.Millisecond)
	execution.CompletedAt = &completedAt
	execution.CurrentStep = frameResult.lastNodeID
	execution.Result = database.JSONMap{
		"origin":     "extension",
		"frameCount": frameResult.frameCount,
		"durationMs": frameResult.totalDurationMs,
	}
	execution.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to finalise execution: %w", err)
	}

	logEntry := &database.ExecutionLog{
		ID:          uuid.New(),
		ExecutionID: execution.ID,
		Timestamp:   time.Now().UTC(),
		Level:       "info",
		Message:     fmt.Sprintf("Imported recording with %d frame(s)", frameResult.frameCount),
		Metadata: database.JSONMap{
			"origin":     "extension",
			"frameCount": frameResult.frameCount,
			"durationMs": frameResult.totalDurationMs,
		},
	}
	_ = s.repo.CreateExecutionLog(ctx, logEntry)

	return &RecordingImportResult{
		Execution:  execution,
		Workflow:   workflow,
		Project:    project,
		FrameCount: frameResult.frameCount,
		AssetCount: frameResult.assetCount,
		DurationMs: frameResult.totalDurationMs,
	}, nil
}

type framePersistResult struct {
	frameCount      int
	assetCount      int
	totalDurationMs int
	lastNodeID      string
}

func (s *RecordingService) persistFrames(
	ctx context.Context,
	zr *zip.Reader,
	project *database.Project,
	workflow *database.Workflow,
	execution *database.Execution,
	manifest *recordingManifest,
) (*framePersistResult, error) {
	files := map[string]*zip.File{}
	for i := range zr.File {
		entry := zr.File[i]
		normalized := normalizeArchiveName(entry.Name)
		if normalized == "" {
			continue
		}
		files[normalized] = entry
	}

	startTime := execution.StartedAt
	if manifest.RecordedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, manifest.RecordedAt); err == nil {
			startTime = parsed
			execution.StartedAt = parsed
		}
	}

	adapter := newRecordingAdapter(execution.ID, workflow.ID, manifest, files, s.log)
	plan := adapter.executionPlan(startTime)

	totalDuration := 0
	assetCount := 0
	lastNodeID := ""

	for index, frame := range manifest.Frames {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		durationMs := frame.DurationMs
		if durationMs <= 0 {
			durationMs = deriveFrameDuration(manifest.Frames, index)
		}
		totalDuration += durationMs
		lastNodeID = frame.NodeID()

		outcome, hasScreenshot, err := adapter.outcomeForFrame(index, frame, startTime, durationMs)
		if err != nil {
			return nil, err
		}
		if hasScreenshot {
			assetCount++
		}

		if _, err := s.recorder.RecordStepOutcome(ctx, plan, outcome); err != nil {
			return nil, fmt.Errorf("failed to persist recording frame %d: %w", index, err)
		}
	}

	return &framePersistResult{
		frameCount:      len(manifest.Frames),
		assetCount:      assetCount,
		totalDurationMs: totalDuration,
		lastNodeID:      lastNodeID,
	}, nil
}

func (s *RecordingService) cleanupRecordingArtifacts(baseCtx context.Context, projectCreated, workflowCreated bool, project *database.Project, workflow *database.Workflow, execution *database.Execution) {
	ctx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	if execution != nil {
		if err := s.repo.DeleteExecution(ctx, execution.ID); err != nil && s.log != nil {
			s.log.WithError(err).WithField("execution_id", execution.ID).Warn("Failed to delete partial execution during recording cleanup")
		}
		execDir := strings.TrimSpace(s.recordingsRoot)
		if execDir != "" {
			path := filepath.Join(execDir, execution.ID.String())
			if err := os.RemoveAll(path); err != nil && s.log != nil {
				s.log.WithError(err).WithField("path", path).Warn("Failed to remove partial recording assets")
			}
		}
	}

	if workflowCreated && workflow != nil {
		if err := s.repo.DeleteWorkflow(ctx, workflow.ID); err != nil && s.log != nil {
			s.log.WithError(err).WithField("workflow_id", workflow.ID).Warn("Failed to delete temporary workflow during recording cleanup")
		}
	}

	if projectCreated && project != nil {
		if err := s.repo.DeleteProject(ctx, project.ID); err != nil && s.log != nil {
			s.log.WithError(err).WithField("project_id", project.ID).Warn("Failed to delete temporary project during recording cleanup")
		}
	}
}

func (s *RecordingService) resolveProject(ctx context.Context, manifest *recordingManifest, opts RecordingImportOptions) (*database.Project, bool, error) {
	if opts.ProjectID != nil {
		if project, err := s.repo.GetProject(ctx, *opts.ProjectID); err == nil {
			return project, false, nil
		}
	}

	candidateNames := []string{}
	if opts.ProjectName != "" {
		candidateNames = append(candidateNames, opts.ProjectName)
	}
	if manifest.ProjectName != "" {
		candidateNames = append(candidateNames, manifest.ProjectName)
	}
	candidateNames = append(candidateNames, "Demo Browser Automations")
	candidateNames = append(candidateNames, "Extension Recordings")

	for _, name := range candidateNames {
		project, err := s.repo.GetProjectByName(ctx, name)
		if err == nil && project != nil {
			return project, false, nil
		}
	}

	project := &database.Project{
		ID:         uuid.New(),
		Name:       "Extension Recordings",
		FolderPath: filepath.Join("scenarios", "browser-automation-studio", "data", "projects", "extension-recordings"),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := os.MkdirAll(project.FolderPath, 0o755); err != nil && s.log != nil {
		s.log.WithError(err).Warn("Failed to ensure extension recordings project directory exists")
	}

	if err := s.repo.CreateProject(ctx, project); err != nil {
		return nil, false, fmt.Errorf("failed to create recordings project: %w", err)
	}

	return project, true, nil
}

func (s *RecordingService) resolveWorkflow(ctx context.Context, manifest *recordingManifest, project *database.Project, opts RecordingImportOptions) (*database.Workflow, bool, error) {
	if opts.WorkflowID != nil {
		if workflow, err := s.repo.GetWorkflow(ctx, *opts.WorkflowID); err == nil {
			return workflow, false, nil
		}
	}

	candidateNames := []string{}
	if opts.WorkflowName != "" {
		candidateNames = append(candidateNames, opts.WorkflowName)
	}
	if manifest.WorkflowName != "" {
		candidateNames = append(candidateNames, manifest.WorkflowName)
	}

	for _, name := range candidateNames {
		if name == "" {
			continue
		}
		workflow, err := s.repo.GetWorkflowByName(ctx, name, defaultRecordingWorkflowFolder)
		if err == nil && workflow != nil {
			return workflow, false, nil
		}
	}

	workflow := &database.Workflow{
		ID:         uuid.New(),
		ProjectID:  &project.ID,
		Name:       deriveWorkflowName(manifest, opts),
		FolderPath: defaultRecordingWorkflowFolder,
		FlowDefinition: database.JSONMap{
			"nodes": []any{},
			"edges": []any{},
		},
		Description: "Imported Chrome extension recording",
		Version:     1,
		CreatedBy:   "extension",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateWorkflow(ctx, workflow); err != nil {
		return nil, false, fmt.Errorf("failed to create recording workflow: %w", err)
	}

	return workflow, true, nil
}

func deriveFrameDuration(frames []recordingFrame, index int) int {
	if index < len(frames)-1 {
		current := frames[index]
		next := frames[index+1]
		delta := next.TimestampMs() - current.TimestampMs()
		if delta > 0 {
			return delta
		}
	}
	return recordingDefaultFrameDurationMs
}

func intPointer(v int) *int {
	value := v
	return &value
}

// --- Manifest types and helpers ---

type recordingManifest struct {
	RunID        string            `json:"runId"`
	ProjectID    string            `json:"projectId"`
	ProjectName  string            `json:"projectName"`
	WorkflowID   string            `json:"workflowId"`
	WorkflowName string            `json:"workflowName"`
	RecordedAt   string            `json:"recordedAt"`
	Extension    map[string]any    `json:"extension"`
	Viewport     recordingViewport `json:"viewport"`
	Frames       []recordingFrame  `json:"frames"`
	Metadata     map[string]any    `json:"metadata"`
}

type recordingViewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type recordingFrame struct {
	Index              int                    `json:"index"`
	Timestamp          any                    `json:"timestamp"`
	DurationMs         int                    `json:"durationMs"`
	Event              string                 `json:"event"`
	StepType           string                 `json:"stepType"`
	Node               string                 `json:"nodeId"`
	Title              string                 `json:"title"`
	Screenshot         string                 `json:"screenshot"`
	FinalURL           string                 `json:"url"`
	Cursor             recordingCursor        `json:"cursor"`
	CursorTrail        []recordingPoint       `json:"cursorTrail"`
	ClickPosition      *recordingPoint        `json:"clickPosition"`
	Click              recordingPoint         `json:"click"`
	FocusedElement     *recordingElementFocus `json:"focusedElement"`
	HighlightRegions   recordingRegions       `json:"highlightRegions"`
	MaskRegions        recordingRegions       `json:"maskRegions"`
	ZoomFactor         float64                `json:"zoomFactor"`
	ConsoleLogs        []runtimeConsoleLog    `json:"-"`
	NetworkEvents      []runtimeNetworkEvent  `json:"-"`
	ConsoleLogsRaw     json.RawMessage        `json:"consoleLogs"`
	ConsoleRaw         json.RawMessage        `json:"console"`
	NetworkRaw         json.RawMessage        `json:"network"`
	Assertion          *runtimeAssertion      `json:"assertion"`
	DomSnapshotHTML    string                 `json:"domSnapshotHtml"`
	DomSnapshotPreview string                 `json:"domSnapshotPreview"`
	Payload            map[string]any         `json:"payload"`
}

type recordingCursor struct {
	Path [][]float64 `json:"path"`
}

type recordingPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type recordingElementFocus struct {
	Selector    string                `json:"selector"`
	BoundingBox *recordingBoundingBox `json:"boundingBox"`
}

type recordingBoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type recordingRegions []recordingRegion

type recordingRegion struct {
	Selector    string                `json:"selector"`
	BoundingBox *recordingBoundingBox `json:"boundingBox"`
	Padding     int                   `json:"padding"`
	Color       string                `json:"color"`
	Opacity     float64               `json:"opacity"`
}

func (m *recordingManifest) normalise() {
	if len(m.Frames) == 0 {
		return
	}
	sort.Slice(m.Frames, func(i, j int) bool {
		return m.Frames[i].TimestampMs() < m.Frames[j].TimestampMs()
	})
	for idx := range m.Frames {
		if m.Frames[idx].Index == 0 {
			m.Frames[idx].Index = idx
		}
		m.Frames[idx].normalise()
	}
}

func (f *recordingFrame) normalise() {
	if len(f.Cursor.Path) == 0 && len(f.CursorTrail) == 0 && f.ClickPositionPoint() == nil {
		if cp := f.Click.toRuntimePoint(); cp != nil {
			f.CursorTrail = append(f.CursorTrail, f.Click)
		}
	}
	f.HighlightRegions = f.HighlightRegions.normalise()
	f.MaskRegions = f.MaskRegions.normalise()
	if len(f.ConsoleLogs) == 0 {
		f.ConsoleLogs = f.parseConsole()
	}
	if len(f.NetworkEvents) == 0 {
		f.NetworkEvents = f.parseNetwork()
	}
}

func (f *recordingFrame) TimestampMs() int {
	switch v := f.Timestamp.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		if iv, err := v.Int64(); err == nil {
			return int(iv)
		}
	}
	return f.Index * recordingDefaultFrameDurationMs
}

func (f *recordingFrame) NodeID() string {
	if strings.TrimSpace(f.Node) != "" {
		return f.Node
	}
	return fmt.Sprintf("recording-frame-%d", f.Index+1)
}

func (f *recordingFrame) StepTypeOrDefault() string {
	if strings.TrimSpace(f.StepType) != "" {
		return f.StepType
	}
	if strings.TrimSpace(f.Event) != "" {
		return strings.ToLower(strings.ReplaceAll(f.Event, " ", "_"))
	}
	return "recording_frame"
}

func (f *recordingFrame) CursorTrailPoints() []autocontracts.CursorPosition {
	points := []autocontracts.CursorPosition{}
	for _, point := range f.CursorTrail {
		if pt := point.toRuntimePoint(); pt != nil {
			points = append(points, autocontracts.CursorPosition{Point: *pt})
		}
	}
	for _, pathPoint := range f.Cursor.Path {
		if len(pathPoint) != 2 {
			continue
		}
		points = append(points, autocontracts.CursorPosition{
			Point: autocontracts.Point{X: pathPoint[0], Y: pathPoint[1]},
		})
	}
	return points
}

func (f *recordingFrame) ClickPositionPoint() *autocontracts.Point {
	if f.ClickPosition == nil {
		return nil
	}
	return f.ClickPosition.toRuntimePoint()
}

func (f *recordingFrame) focusedElementContracts() *autocontracts.ElementFocus {
	if f.FocusedElement == nil {
		return nil
	}
	return f.FocusedElement.toContracts()
}

func (f *recordingFrame) parseConsole() []runtimeConsoleLog {
	if len(f.ConsoleLogsRaw) > 0 {
		var logs []runtimeConsoleLog
		if err := json.Unmarshal(f.ConsoleLogsRaw, &logs); err == nil {
			return logs
		}
	}
	if len(f.ConsoleRaw) > 0 {
		var logs []runtimeConsoleLog
		if err := json.Unmarshal(f.ConsoleRaw, &logs); err == nil {
			return logs
		}
	}
	return nil
}

func (f *recordingFrame) parseNetwork() []runtimeNetworkEvent {
	if len(f.NetworkRaw) == 0 {
		return nil
	}
	var events []runtimeNetworkEvent
	if err := json.Unmarshal(f.NetworkRaw, &events); err == nil {
		return events
	}
	return nil
}

func (p recordingPoint) toRuntimePoint() *autocontracts.Point {
	if p.X == 0 && p.Y == 0 {
		return nil
	}
	return &autocontracts.Point{X: p.X, Y: p.Y}
}

func (r recordingRegions) normalise() recordingRegions {
	sort.Slice(r, func(i, j int) bool {
		return r[i].Selector < r[j].Selector
	})
	return r
}

func (r recordingRegions) toContracts() []autocontracts.HighlightRegion {
	result := make([]autocontracts.HighlightRegion, 0, len(r))
	for _, region := range r {
		runtimeRegion := autocontracts.HighlightRegion{Selector: region.Selector}
		if region.BoundingBox != nil {
			runtimeRegion.BoundingBox = region.BoundingBox.toContracts()
		}
		if region.Padding != 0 {
			runtimeRegion.Padding = region.Padding
		}
		if region.Color != "" {
			runtimeRegion.Color = region.Color
		}
		result = append(result, runtimeRegion)
	}
	return result
}

func (r recordingRegions) toMaskContracts() []autocontracts.MaskRegion {
	result := make([]autocontracts.MaskRegion, 0, len(r))
	for _, region := range r {
		maskRegion := autocontracts.MaskRegion{Selector: region.Selector}
		if region.BoundingBox != nil {
			maskRegion.BoundingBox = region.BoundingBox.toContracts()
		}
		if region.Opacity != 0 {
			maskRegion.Opacity = region.Opacity
		}
		result = append(result, maskRegion)
	}
	return result
}

func (b *recordingBoundingBox) toContracts() *autocontracts.BoundingBox {
	if b == nil {
		return nil
	}
	return &autocontracts.BoundingBox{
		X:      b.X,
		Y:      b.Y,
		Width:  b.Width,
		Height: b.Height,
	}
}

func (e *recordingElementFocus) toContracts() *autocontracts.ElementFocus {
	if e == nil {
		return nil
	}
	return &autocontracts.ElementFocus{
		Selector:    e.Selector,
		BoundingBox: e.BoundingBox.toContracts(),
	}
}

var (
	_ = image.Rect(0, 0, 0, 0)
)

// Exported for testing
var (
	DecodeDimensions = decodeDimensions
)
