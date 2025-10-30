package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless/events"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
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

	GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error)
	GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error)
	CreateWorkflow(ctx context.Context, workflow *database.Workflow) error

	CreateExecution(ctx context.Context, execution *database.Execution) error
	UpdateExecution(ctx context.Context, execution *database.Execution) error

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
	storage        *storage.MinIOClient
	log            *logrus.Logger
	emitter        events.Sink
	recordingsRoot string
}

// NewRecordingService constructs a recording ingestion service.
func NewRecordingService(repo RecordingRepository, storageClient *storage.MinIOClient, hub *wsHub.Hub, log *logrus.Logger, recordingsRoot string) *RecordingService {
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

	var emitter events.Sink
	if hub != nil {
		emitter = events.NewEmitter(hub, log)
	}

	return &RecordingService{
		repo:           repo,
		storage:        storageClient,
		log:            log,
		emitter:        emitter,
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

	project, err := s.resolveProject(ctx, manifest, opts)
	if err != nil {
		return nil, err
	}

	workflow, err := s.resolveWorkflow(ctx, manifest, project, opts)
	if err != nil {
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
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	frameResult, err := s.persistFrames(ctx, &zr.Reader, project, workflow, execution, manifest)
	if err != nil {
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

	if s.emitter != nil {
		s.emitter.Emit(events.NewEvent(
			events.EventExecutionCompleted,
			execution.ID,
			workflow.ID,
			events.WithStatus("completed"),
			events.WithProgress(100),
			events.WithMessage("Recording imported"),
			events.WithPayload(map[string]any{
				"origin":     "extension",
				"frameCount": frameResult.frameCount,
				"durationMs": frameResult.totalDurationMs,
			}),
		))
	}

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

	executionDir := filepath.Join(s.recordingsRoot, execution.ID.String())
	framesDir := filepath.Join(executionDir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to prepare recordings directory: %w", err)
	}

	viewport := manifest.Viewport
	totalDuration := 0
	assetCount := 0
	lastNodeID := ""

	frameCount := len(manifest.Frames)

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

		stepIndex := index
		nodeID := frame.NodeID()
		lastNodeID = nodeID

		stepStartedAt := startTime.Add(time.Duration(frame.TimestampMs()) * time.Millisecond)
		completedAt := stepStartedAt.Add(time.Duration(durationMs) * time.Millisecond)
		progress := int(float64(index+1) / float64(frameCount) * 100)
		if progress > 100 {
			progress = 100
		}

		step := &database.ExecutionStep{
			ID:          uuid.New(),
			ExecutionID: execution.ID,
			StepIndex:   stepIndex,
			NodeID:      nodeID,
			StepType:    frame.StepTypeOrDefault(),
			Status:      "completed",
			StartedAt:   stepStartedAt,
			CompletedAt: &completedAt,
			DurationMs:  durationMs,
			Input: database.JSONMap{
				"origin": "extension",
			},
			Output: database.JSONMap{
				"success":    true,
				"durationMs": durationMs,
				"finalUrl":   frame.FinalURL,
			},
			Metadata: database.JSONMap{
				"origin":    "extension",
				"event":     frame.Event,
				"timestamp": frame.TimestampMs(),
			},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		if frame.FinalURL != "" {
			step.Metadata["finalUrl"] = frame.FinalURL
		}
		if frame.Title != "" {
			step.Metadata["title"] = frame.Title
		}
		if frame.Payload != nil {
			step.Metadata["payload"] = frame.Payload
		}

		if err := s.repo.CreateExecutionStep(ctx, step); err != nil {
			return nil, fmt.Errorf("failed to persist recording frame step: %w", err)
		}

		artifactIDs := make([]string, 0, 4)
		timelinePayload := database.JSONMap{
			"stepIndex":  stepIndex,
			"nodeId":     nodeID,
			"stepType":   step.StepType,
			"success":    true,
			"durationMs": durationMs,
			"progress":   progress,
			"origin":     "extension",
		}

		if frame.FinalURL != "" {
			timelinePayload["finalUrl"] = frame.FinalURL
		}

		cursorTrail := frame.CursorTrailPoints()
		if len(cursorTrail) > 0 {
			timelinePayload["cursorTrail"] = cursorTrail
		}
		if frame.ClickPosition != nil {
			timelinePayload["clickPosition"] = frame.ClickPosition.toRuntimePoint()
		}
		if frame.FocusedElement != nil {
			timelinePayload["focusedElement"] = frame.FocusedElement.toRuntime()
		}
		if len(frame.HighlightRegions) > 0 {
			timelinePayload["highlightRegions"] = frame.HighlightRegions.toRuntime()
		}
		if len(frame.MaskRegions) > 0 {
			timelinePayload["maskRegions"] = frame.MaskRegions.toRuntime()
		}
		if frame.ZoomFactor > 0 {
			timelinePayload["zoomFactor"] = frame.ZoomFactor
		}
		if frame.Assertion != nil {
			timelinePayload["assertion"] = frame.Assertion
		}
		if len(frame.ConsoleLogs) > 0 {
			timelinePayload["consoleLogCount"] = len(frame.ConsoleLogs)
		}
		if len(frame.NetworkEvents) > 0 {
			timelinePayload["networkEventCount"] = len(frame.NetworkEvents)
		}
		if frame.DomSnapshotPreview != "" {
			timelinePayload["domSnapshotPreview"] = frame.DomSnapshotPreview
		}

		if frame.DomSnapshotHTML != "" {
			domArtifact := &database.ExecutionArtifact{
				ID:           uuid.New(),
				ExecutionID:  execution.ID,
				StepID:       &step.ID,
				StepIndex:    intPointer(stepIndex),
				ArtifactType: "dom_snapshot",
				Label:        fmt.Sprintf("DOM %s", nodeID),
				Payload: database.JSONMap{
					"html":   frame.DomSnapshotHTML,
					"origin": "extension",
				},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := s.repo.CreateExecutionArtifact(ctx, domArtifact); err == nil {
				artifactIDs = append(artifactIDs, domArtifact.ID.String())
				timelinePayload["domSnapshotArtifactId"] = domArtifact.ID.String()
			}
		}

		screenshotInfo, err := s.persistScreenshotAsset(ctx, files, framesDir, execution, step, frame, viewport)
		if err != nil {
			return nil, err
		}
		if screenshotInfo != nil {
			artifactIDs = append(artifactIDs, screenshotInfo.artifactID)
			assetCount++
			timelinePayload["screenshotArtifactId"] = screenshotInfo.artifactID
			if screenshotInfo.url != "" {
				timelinePayload["screenshotUrl"] = screenshotInfo.url
			}
			if screenshotInfo.thumbnailURL != "" {
				timelinePayload["screenshotThumbnail"] = screenshotInfo.thumbnailURL
			}
			timelinePayload["width"] = screenshotInfo.width
			timelinePayload["height"] = screenshotInfo.height
		}

		if len(frame.ConsoleLogs) > 0 {
			consoleArtifact := &database.ExecutionArtifact{
				ID:           uuid.New(),
				ExecutionID:  execution.ID,
				StepID:       &step.ID,
				StepIndex:    intPointer(stepIndex),
				ArtifactType: "console_logs",
				Label:        fmt.Sprintf("Console %s", nodeID),
				Payload: database.JSONMap{
					"entries": frame.ConsoleLogs,
					"origin":  "extension",
				},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := s.repo.CreateExecutionArtifact(ctx, consoleArtifact); err == nil {
				artifactIDs = append(artifactIDs, consoleArtifact.ID.String())
			}
		}

		if len(frame.NetworkEvents) > 0 {
			networkArtifact := &database.ExecutionArtifact{
				ID:           uuid.New(),
				ExecutionID:  execution.ID,
				StepID:       &step.ID,
				StepIndex:    intPointer(stepIndex),
				ArtifactType: "network_events",
				Label:        fmt.Sprintf("Network %s", nodeID),
				Payload: database.JSONMap{
					"entries": frame.NetworkEvents,
					"origin":  "extension",
				},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := s.repo.CreateExecutionArtifact(ctx, networkArtifact); err == nil {
				artifactIDs = append(artifactIDs, networkArtifact.ID.String())
			}
		}

		if len(artifactIDs) > 0 {
			step.Metadata["artifactIds"] = artifactIDs
			if err := s.repo.UpdateExecutionStep(ctx, step); err != nil {
				if s.log != nil {
					s.log.WithError(err).Warn("Failed to update step metadata with artifact IDs")
				}
			}
		}

		timelineArtifact := &database.ExecutionArtifact{
			ID:           uuid.New(),
			ExecutionID:  execution.ID,
			StepID:       &step.ID,
			StepIndex:    intPointer(stepIndex),
			ArtifactType: "timeline_frame",
			Label:        frame.Title,
			Payload:      timelinePayload,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}
		if err := s.repo.CreateExecutionArtifact(ctx, timelineArtifact); err != nil {
			return nil, fmt.Errorf("failed to persist timeline artifact: %w", err)
		}
	}

	return &framePersistResult{
		frameCount:      frameCount,
		assetCount:      assetCount,
		totalDurationMs: totalDuration,
		lastNodeID:      lastNodeID,
	}, nil
}

func (s *RecordingService) persistScreenshotAsset(
	ctx context.Context,
	files map[string]*zip.File,
	framesDir string,
	execution *database.Execution,
	step *database.ExecutionStep,
	frame recordingFrame,
	viewport recordingViewport,
) (*persistedScreenshot, error) {
	if strings.TrimSpace(frame.Screenshot) == "" {
		return nil, nil
	}

	entry := files[normalizeArchiveName(frame.Screenshot)]
	if entry == nil {
		if s.log != nil {
			s.log.WithField("path", frame.Screenshot).Warn("Recording frame referenced screenshot that was not present in archive")
		}
		return nil, nil
	}

	if entry.UncompressedSize64 > maxRecordingAssetBytes {
		return nil, fmt.Errorf("frame screenshot exceeds maximum size (%d bytes)", maxRecordingAssetBytes)
	}

	data, contentType, err := readZipFile(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshot asset: %w", err)
	}

	if len(data) > maxRecordingAssetBytes {
		return nil, fmt.Errorf("frame screenshot exceeds maximum size (%d bytes)", maxRecordingAssetBytes)
	}

	fileExt := strings.ToLower(filepath.Ext(entry.Name))
	if fileExt == "" {
		fileExt = extFromContentType(contentType)
	}
	if fileExt == "" {
		fileExt = ".png"
	}

	filename := fmt.Sprintf("frame-%04d%s", step.StepIndex+1, fileExt)
	destPath := filepath.Join(framesDir, filename)
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to persist screenshot asset: %w", err)
	}

	width, height := decodeDimensions(data)
	if width == 0 {
		width = viewport.Width
	}
	if height == 0 {
		height = viewport.Height
	}

	var info *storage.ScreenshotInfo
	if s.storage != nil {
		if stored, err := s.storage.StoreScreenshot(ctx, execution.ID, step.NodeID, data, contentType); err == nil {
			info = stored
		} else if s.log != nil {
			s.log.WithError(err).Warn("Failed to upload screenshot to MinIO; using filesystem asset")
		}
	}

	publicURL := fmt.Sprintf("/api/v1/recordings/assets/%s/frames/%s", execution.ID.String(), filename)

	if info == nil {
		info = &storage.ScreenshotInfo{
			URL:          publicURL,
			ThumbnailURL: publicURL,
			SizeBytes:    int64(len(data)),
			Width:        width,
			Height:       height,
		}
	} else {
		if info.URL == "" {
			info.URL = publicURL
		}
		if info.ThumbnailURL == "" {
			info.ThumbnailURL = publicURL
		}
		if info.Width == 0 {
			info.Width = width
		}
		if info.Height == 0 {
			info.Height = height
		}
	}

	size := info.SizeBytes
	artifact := &database.ExecutionArtifact{
		ID:           uuid.New(),
		ExecutionID:  execution.ID,
		StepID:       &step.ID,
		StepIndex:    intPointer(step.StepIndex),
		ArtifactType: "screenshot",
		Label:        frame.Title,
		StorageURL:   info.URL,
		ThumbnailURL: info.ThumbnailURL,
		ContentType:  contentType,
		SizeBytes:    &size,
		Payload: database.JSONMap{
			"origin":    "extension",
			"width":     info.Width,
			"height":    info.Height,
			"viewport":  viewport,
			"fileName":  filename,
			"publicUrl": publicURL,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if frame.FocusMetadata() != nil {
		artifact.Payload["focusedElement"] = frame.FocusMetadata()
	}
	if len(frame.HighlightRegions) > 0 {
		artifact.Payload["highlightRegions"] = frame.HighlightRegions.toRuntime()
	}
	if len(frame.MaskRegions) > 0 {
		artifact.Payload["maskRegions"] = frame.MaskRegions.toRuntime()
	}
	if frame.ZoomFactor > 0 {
		artifact.Payload["zoomFactor"] = frame.ZoomFactor
	}

	if err := s.repo.CreateExecutionArtifact(ctx, artifact); err != nil {
		return nil, fmt.Errorf("failed to persist screenshot artifact: %w", err)
	}

	return &persistedScreenshot{
		artifactID:   artifact.ID.String(),
		url:          info.URL,
		thumbnailURL: info.ThumbnailURL,
		width:        info.Width,
		height:       info.Height,
	}, nil
}

type persistedScreenshot struct {
	artifactID   string
	url          string
	thumbnailURL string
	width        int
	height       int
}

func (s *RecordingService) resolveProject(ctx context.Context, manifest *recordingManifest, opts RecordingImportOptions) (*database.Project, error) {
	if opts.ProjectID != nil {
		if project, err := s.repo.GetProject(ctx, *opts.ProjectID); err == nil {
			return project, nil
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
			return project, nil
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
		return nil, fmt.Errorf("failed to create recordings project: %w", err)
	}

	return project, nil
}

func (s *RecordingService) resolveWorkflow(ctx context.Context, manifest *recordingManifest, project *database.Project, opts RecordingImportOptions) (*database.Workflow, error) {
	if opts.WorkflowID != nil {
		if workflow, err := s.repo.GetWorkflow(ctx, *opts.WorkflowID); err == nil {
			return workflow, nil
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
			return workflow, nil
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
		return nil, fmt.Errorf("failed to create recording workflow: %w", err)
	}

	return workflow, nil
}

func deriveWorkflowName(manifest *recordingManifest, opts RecordingImportOptions) string {
	if opts.WorkflowName != "" {
		return opts.WorkflowName
	}
	if manifest.WorkflowName != "" {
		return manifest.WorkflowName
	}
	if manifest.RunID != "" {
		return fmt.Sprintf("Recording %s", manifest.RunID)
	}
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("Extension Recording %s", timestamp)
}

func loadRecordingManifest(zr *zip.Reader) (*recordingManifest, error) {
	var manifestFile *zip.File
	for i := range zr.File {
		entry := zr.File[i]
		name := strings.ToLower(entry.Name)
		if strings.HasSuffix(name, "manifest.json") || strings.HasSuffix(name, "recording.json") {
			manifestFile = entry
			break
		}
	}

	if manifestFile == nil {
		return nil, errors.New("recording archive is missing manifest.json")
	}

	data, _, err := readZipFile(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest recordingManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid recording manifest: %w", err)
	}

	manifest.normalise()
	return &manifest, nil
}

func readZipFile(entry *zip.File) ([]byte, string, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, "", err
	}
	defer reader.Close()

	buf := bytes.NewBuffer(make([]byte, 0, entry.UncompressedSize64))
	if _, err := io.Copy(buf, io.LimitReader(reader, maxRecordingAssetBytes+1)); err != nil {
		return nil, "", err
	}

	data := buf.Bytes()
	contentType := http.DetectContentType(firstN(data, 512))
	return data, contentType, nil
}

func firstN(data []byte, n int) []byte {
	if len(data) <= n {
		return data
	}
	return data[:n]
}

func decodeDimensions(data []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func extFromContentType(contentType string) string {
	switch strings.ToLower(contentType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

func normalizeArchiveName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	normalized := filepath.ToSlash(trimmed)
	normalized = strings.TrimPrefix(normalized, "./")
	normalized = strings.Trim(normalized, "/")
	if strings.Contains(normalized, "..") {
		return ""
	}
	return normalized
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
	Index              int                      `json:"index"`
	Timestamp          any                      `json:"timestamp"`
	DurationMs         int                      `json:"durationMs"`
	Event              string                   `json:"event"`
	StepType           string                   `json:"stepType"`
	Node               string                   `json:"nodeId"`
	Title              string                   `json:"title"`
	Screenshot         string                   `json:"screenshot"`
	FinalURL           string                   `json:"url"`
	Cursor             recordingCursor          `json:"cursor"`
	CursorTrail        []recordingPoint         `json:"cursorTrail"`
	ClickPosition      *recordingPoint          `json:"clickPosition"`
	Click              recordingPoint           `json:"click"`
	FocusedElement     *recordingElementFocus   `json:"focusedElement"`
	HighlightRegions   recordingRegions         `json:"highlightRegions"`
	MaskRegions        recordingRegions         `json:"maskRegions"`
	ZoomFactor         float64                  `json:"zoomFactor"`
	ConsoleLogs        []runtime.ConsoleLog     `json:"-"`
	NetworkEvents      []runtime.NetworkEvent   `json:"-"`
	ConsoleLogsRaw     json.RawMessage          `json:"consoleLogs"`
	ConsoleRaw         json.RawMessage          `json:"console"`
	NetworkRaw         json.RawMessage          `json:"network"`
	Assertion          *runtime.AssertionResult `json:"assertion"`
	DomSnapshotHTML    string                   `json:"domSnapshotHtml"`
	DomSnapshotPreview string                   `json:"domSnapshotPreview"`
	Payload            map[string]any           `json:"payload"`
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
	if len(f.Cursor.Path) == 0 && len(f.CursorTrail) == 0 && len(f.ClickPositionPointSlice()) == 0 {
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

func (f *recordingFrame) CursorTrailPoints() []runtime.Point {
	points := []runtime.Point{}
	for _, point := range f.CursorTrail {
		if pt := point.toRuntimePoint(); pt != nil {
			points = append(points, *pt)
		}
	}
	for _, pathPoint := range f.Cursor.Path {
		if len(pathPoint) != 2 {
			continue
		}
		points = append(points, runtime.Point{X: pathPoint[0], Y: pathPoint[1]})
	}
	return points
}

func (f *recordingFrame) FocusMetadata() any {
	if f.FocusedElement == nil {
		return nil
	}
	return f.FocusedElement.toRuntime()
}

func (f *recordingFrame) ClickPositionPointSlice() []runtime.Point {
	if f.ClickPosition != nil {
		if pt := f.ClickPosition.toRuntimePoint(); pt != nil {
			return []runtime.Point{*pt}
		}
	}
	return nil
}

func (f *recordingFrame) parseConsole() []runtime.ConsoleLog {
	if len(f.ConsoleLogsRaw) > 0 {
		var logs []runtime.ConsoleLog
		if err := json.Unmarshal(f.ConsoleLogsRaw, &logs); err == nil {
			return logs
		}
	}
	if len(f.ConsoleRaw) > 0 {
		var logs []runtime.ConsoleLog
		if err := json.Unmarshal(f.ConsoleRaw, &logs); err == nil {
			return logs
		}
	}
	return nil
}

func (f *recordingFrame) parseNetwork() []runtime.NetworkEvent {
	if len(f.NetworkRaw) == 0 {
		return nil
	}
	var events []runtime.NetworkEvent
	if err := json.Unmarshal(f.NetworkRaw, &events); err == nil {
		return events
	}
	return nil
}

func (p recordingPoint) toRuntimePoint() *runtime.Point {
	if p.X == 0 && p.Y == 0 {
		return nil
	}
	return &runtime.Point{X: p.X, Y: p.Y}
}

func (r recordingRegions) normalise() recordingRegions {
	sort.Slice(r, func(i, j int) bool {
		return r[i].Selector < r[j].Selector
	})
	return r
}

func (r recordingRegions) toRuntime() []runtime.HighlightRegion {
	result := make([]runtime.HighlightRegion, 0, len(r))
	for _, region := range r {
		runtimeRegion := runtime.HighlightRegion{Selector: region.Selector}
		if region.BoundingBox != nil {
			runtimeRegion.BoundingBox = region.BoundingBox.toRuntime()
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

func (b *recordingBoundingBox) toRuntime() *runtime.BoundingBox {
	if b == nil {
		return nil
	}
	return &runtime.BoundingBox{
		X:      b.X,
		Y:      b.Y,
		Width:  b.Width,
		Height: b.Height,
	}
}

func (e *recordingElementFocus) toRuntime() *runtime.ElementFocus {
	if e == nil {
		return nil
	}
	return &runtime.ElementFocus{
		Selector:    e.Selector,
		BoundingBox: e.BoundingBox.toRuntime(),
	}
}

var (
	_ = image.Rect(0, 0, 0, 0)
)

// Exported for testing
var (
	DecodeDimensions = decodeDimensions
)
