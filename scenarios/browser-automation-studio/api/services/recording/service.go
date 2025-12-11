package recording

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

const (
	defaultRecordingWorkflowFolder = "/recordings"
	defaultRecordingWorkflowName   = "Extension Recording"
)

// maxRecordingFrames returns the configured maximum frame count.
// Configurable via BAS_RECORDING_MAX_FRAMES (default: 400)
func maxRecordingFrames() int {
	return config.Load().Recording.MaxFrames
}

// maxRecordingArchiveBytes returns the configured maximum archive size.
// Configurable via BAS_RECORDING_MAX_ARCHIVE_BYTES (default: 200MB)
func maxRecordingArchiveBytes() int64 {
	return config.Load().Recording.MaxArchiveBytes
}

// maxRecordingAssetBytes returns the configured maximum per-frame size.
// Configurable via BAS_RECORDING_MAX_FRAME_BYTES (default: 12MB)
func maxRecordingAssetBytes() int64 {
	return config.Load().Recording.MaxFrameBytes
}

// recordingDefaultFrameDurationMs returns the configured default frame duration.
// Configurable via BAS_RECORDING_DEFAULT_FRAME_DURATION_MS (default: 1600)
func recordingDefaultFrameDurationMs() int {
	return config.Load().Recording.DefaultFrameDurationMs
}

var (
	errManifestMissingFrames = errors.New("recording manifest did not include any frames")
	errManifestTooLarge      = errors.New("recording archive exceeds maximum allowed size")
)

// errTooManyFrames returns an error with the current max frame limit.
func errTooManyFrames() error {
	return fmt.Errorf("recording exceeds maximum frame count (%d)", maxRecordingFrames())
}

// Exported error aliases for handler detection.
var (
	ErrRecordingManifestMissingFrames = errManifestMissingFrames
	ErrRecordingArchiveTooLarge       = errManifestTooLarge
)

// ErrRecordingTooManyFrames is checked via errors.Is by using a custom type.
var ErrRecordingTooManyFrames = errors.New("recording exceeds maximum frame count")

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
	UpdateExecutionCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, progress int) error
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
//
// This service is split across multiple files for better organization:
//   - recording_service.go: Core service and main ImportArchive method
//   - recording_types.go: Manifest types and their methods (recordingManifest, recordingFrame, etc.)
//   - recording_resolution.go: Project and workflow resolution logic
//   - recording_persistence.go: Frame persistence and cleanup operations
//   - recording_adapter.go: Conversion from recording frames to automation contracts
//   - recording_helpers.go: Utility functions (manifest loading, storage selection, etc.)
//   - recording_file_store.go: Local filesystem storage implementation
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

	if info.Size() > maxRecordingArchiveBytes() {
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

	if len(manifest.Frames) > maxRecordingFrames() {
		return nil, errTooManyFrames()
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
