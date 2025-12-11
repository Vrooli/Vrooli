package recording

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
)

type fakeRecordingRepo struct {
	projects          map[uuid.UUID]*database.Project
	projectsByName    map[string]*database.Project
	workflows         map[uuid.UUID]*database.Workflow
	workflowsByName   map[string]*database.Workflow
	executions        map[uuid.UUID]*database.Execution
	steps             []*database.ExecutionStep
	artifacts         []*database.ExecutionArtifact
	logs              []*database.ExecutionLog
	deletedProjects   []uuid.UUID
	deletedWorkflows  []uuid.UUID
	deletedExecutions []uuid.UUID

	createExecutionStepErr error
}

func newFakeRecordingRepo() *fakeRecordingRepo {
	return &fakeRecordingRepo{
		projects:        make(map[uuid.UUID]*database.Project),
		projectsByName:  make(map[string]*database.Project),
		workflows:       make(map[uuid.UUID]*database.Workflow),
		workflowsByName: make(map[string]*database.Workflow),
		executions:      make(map[uuid.UUID]*database.Execution),
	}
}

func (r *fakeRecordingRepo) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	if project, ok := r.projects[id]; ok {
		return project, nil
	}
	return nil, database.ErrNotFound
}

func (r *fakeRecordingRepo) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	if project, ok := r.projectsByName[name]; ok {
		return project, nil
	}
	return nil, database.ErrNotFound
}

func (r *fakeRecordingRepo) CreateProject(ctx context.Context, project *database.Project) error {
	copy := *project
	r.projects[project.ID] = &copy
	r.projectsByName[project.Name] = &copy
	return nil
}

func (r *fakeRecordingRepo) DeleteProject(ctx context.Context, id uuid.UUID) error {
	delete(r.projects, id)
	for name, project := range r.projectsByName {
		if project.ID == id {
			delete(r.projectsByName, name)
		}
	}
	r.deletedProjects = append(r.deletedProjects, id)
	return nil
}

func (r *fakeRecordingRepo) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	if wf, ok := r.workflows[id]; ok {
		return wf, nil
	}
	return nil, database.ErrNotFound
}

func (r *fakeRecordingRepo) GetWorkflowByName(ctx context.Context, name, folder string) (*database.Workflow, error) {
	if wf, ok := r.workflowsByName[name+"::"+folder]; ok {
		return wf, nil
	}
	return nil, database.ErrNotFound
}

func (r *fakeRecordingRepo) GetWorkflowByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*database.Workflow, error) {
	for _, wf := range r.workflows {
		if wf.ProjectID != nil && *wf.ProjectID == projectID && wf.Name == name {
			return wf, nil
		}
	}
	return nil, database.ErrNotFound
}

func (r *fakeRecordingRepo) CreateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	copy := *workflow
	r.workflows[workflow.ID] = &copy
	r.workflowsByName[workflow.Name+"::"+workflow.FolderPath] = &copy
	return nil
}

func (r *fakeRecordingRepo) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	if wf, ok := r.workflows[id]; ok {
		key := wf.Name + "::" + wf.FolderPath
		delete(r.workflowsByName, key)
	}
	delete(r.workflows, id)
	r.deletedWorkflows = append(r.deletedWorkflows, id)
	return nil
}

func (r *fakeRecordingRepo) CreateExecution(ctx context.Context, execution *database.Execution) error {
	copy := *execution
	r.executions[execution.ID] = &copy
	return nil
}

func (r *fakeRecordingRepo) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	copy := *execution
	r.executions[execution.ID] = &copy
	return nil
}

func (r *fakeRecordingRepo) DeleteExecution(ctx context.Context, id uuid.UUID) error {
	delete(r.executions, id)
	r.deletedExecutions = append(r.deletedExecutions, id)
	return nil
}

func (r *fakeRecordingRepo) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	if r.createExecutionStepErr != nil {
		return r.createExecutionStepErr
	}
	copy := *step
	r.steps = append(r.steps, &copy)
	return nil
}

func (r *fakeRecordingRepo) UpdateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	for idx, existing := range r.steps {
		if existing.ID == step.ID {
			copy := *step
			r.steps[idx] = &copy
			return nil
		}
	}
	return nil
}

func (r *fakeRecordingRepo) CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error {
	copy := *artifact
	r.artifacts = append(r.artifacts, &copy)
	return nil
}

func (r *fakeRecordingRepo) CreateExecutionLog(ctx context.Context, logEntry *database.ExecutionLog) error {
	copy := *logEntry
	r.logs = append(r.logs, &copy)
	return nil
}

func (r *fakeRecordingRepo) UpdateExecutionCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, progress int) error {
	return nil
}

func TestRecordingServiceImportArchive(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-TIMELINE-PERSISTENCE] imports recording archive and creates timeline artifacts", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRecordingRepo()
		demoProject := &database.Project{
			ID:         uuid.New(),
			Name:       "Demo Browser Automations",
			FolderPath: t.TempDir(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		repo.CreateProject(context.Background(), demoProject)

		recordingsRoot := t.TempDir()
		log := logrus.New()
		log.SetOutput(io.Discard)

		service := NewRecordingService(repo, nil, nil, log, recordingsRoot)

		archivePath := filepath.Join(t.TempDir(), "recording.zip")
		createTestRecordingArchive(t, archivePath)

		result, err := service.ImportArchive(context.Background(), archivePath, RecordingImportOptions{})
		if err != nil {
			t.Fatalf("ImportArchive returned error: %v", err)
		}

		if result.FrameCount != 2 {
			t.Fatalf("expected frame count 2, got %d", result.FrameCount)
		}
		if result.AssetCount != 2 {
			t.Fatalf("expected asset count 2, got %d", result.AssetCount)
		}
		if result.Execution.TriggerType != "extension" {
			t.Fatalf("expected trigger type 'extension', got %s", result.Execution.TriggerType)
		}

		if len(repo.steps) != 2 {
			t.Fatalf("expected 2 execution steps, got %d", len(repo.steps))
		}

		hasTimeline := false
		hasScreenshot := false
		for _, artifact := range repo.artifacts {
			switch artifact.ArtifactType {
			case "timeline_frame":
				hasTimeline = true
				if _, ok := artifact.Payload["screenshotArtifactId"]; !ok {
					t.Fatalf("timeline frame missing screenshot artifact reference")
				}
			case "screenshot":
				hasScreenshot = true
				if !strings.Contains(artifact.StorageURL, "/api/v1/recordings/assets/") {
					t.Fatalf("unexpected storage URL: %s", artifact.StorageURL)
				}
			}
		}

		if !hasTimeline {
			t.Fatalf("expected at least one timeline artifact")
		}
		if !hasScreenshot {
			t.Fatalf("expected at least one screenshot artifact")
		}

		// Verify files persisted to the recordings root
		frameGlob := filepath.Join(recordingsRoot, result.Execution.ID.String(), "frames", "*.png")
		matches, err := filepath.Glob(frameGlob)
		if err != nil || len(matches) == 0 {
			t.Fatalf("expected persisted frame asset, got error: %v", err)
		}
	})
}

func TestRecordingServiceImportArchiveRejectsEmptyManifest(t *testing.T) {
	t.Run("[REQ:BAS-RECORDING-INGEST-VALIDATION] rejects archives without frames", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRecordingRepo()
		log := logrus.New()
		log.SetOutput(io.Discard)
		service := NewRecordingService(repo, nil, nil, log, t.TempDir())

		archivePath := filepath.Join(t.TempDir(), "empty-recording.zip")
		manifest := map[string]any{
			"runId":  "empty-run",
			"frames": []map[string]any{},
			"viewport": map[string]int{
				"width":  1280,
				"height": 720,
			},
		}
		writeRecordingArchive(t, archivePath, manifest, nil)

		_, err := service.ImportArchive(context.Background(), archivePath, RecordingImportOptions{})
		if err == nil || !errors.Is(err, ErrRecordingManifestMissingFrames) {
			t.Fatalf("expected ErrRecordingManifestMissingFrames, got %v", err)
		}
		if len(repo.executions) != 0 {
			t.Fatalf("expected no executions to be created, found %d", len(repo.executions))
		}
		if len(repo.projects) != 0 {
			t.Fatalf("expected no projects to be created, found %d", len(repo.projects))
		}
	})
}

func TestRecordingServiceImportArchiveCleansUpOnPersistenceFailure(t *testing.T) {
	t.Run("[REQ:BAS-RECORDING-INGEST-RECOVERY] cleans up partial resources when frame persistence fails", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRecordingRepo()
		repo.createExecutionStepErr = errors.New("step persistence failed")

		log := logrus.New()
		log.SetOutput(io.Discard)

		recordingsRoot := filepath.Join(t.TempDir(), "recordings-root")
		service := NewRecordingService(repo, nil, nil, log, recordingsRoot)

		archivePath := filepath.Join(t.TempDir(), "recording.zip")
		createTestRecordingArchive(t, archivePath)

		_, err := service.ImportArchive(context.Background(), archivePath, RecordingImportOptions{})
		if err == nil {
			t.Fatalf("expected import to fail due to recorder error")
		}

		if len(repo.projects) != 0 {
			t.Fatalf("expected temporary project to be cleaned up, found %d entries", len(repo.projects))
		}
		if len(repo.workflows) != 0 {
			t.Fatalf("expected temporary workflow to be cleaned up, found %d entries", len(repo.workflows))
		}
		if len(repo.executions) != 0 {
			t.Fatalf("expected execution to be deleted, found %d entries", len(repo.executions))
		}

		if len(repo.deletedProjects) != 1 {
			t.Fatalf("expected project deletion to be recorded, got %d", len(repo.deletedProjects))
		}
		if len(repo.deletedWorkflows) != 1 {
			t.Fatalf("expected workflow deletion to be recorded, got %d", len(repo.deletedWorkflows))
		}
		if len(repo.deletedExecutions) != 1 {
			t.Fatalf("expected execution deletion to be recorded, got %d", len(repo.deletedExecutions))
		}

		entries, err := os.ReadDir(recordingsRoot)
		if err != nil {
			t.Fatalf("failed to read recordings root: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected recordings root to be empty after cleanup, found %d entries", len(entries))
		}
	})
}

func createTestRecordingArchive(t *testing.T, archivePath string) {
	t.Helper()

	manifest := map[string]any{
		"runId":    "test-run",
		"viewport": map[string]int{"width": 1280, "height": 720},
		"frames": []map[string]any{
			{
				"index":      0,
				"timestamp":  0,
				"durationMs": 1200,
				"event":      "navigate",
				"stepType":   "navigate",
				"title":      "Open example",
				"url":        "https://example.com",
				"screenshot": "frames/0001.png",
				"cursorTrail": []map[string]float64{
					{"x": 10, "y": 20},
					{"x": 30, "y": 40},
				},
				"consoleLogs": []map[string]any{
					{"type": "log", "text": "Console message", "timestamp": 1},
				},
				"network": []map[string]any{
					{"type": "request", "url": "https://example.com", "timestamp": 2},
				},
			},
			{
				"index":      1,
				"timestamp":  1500,
				"durationMs": 1500,
				"event":      "click",
				"stepType":   "click",
				"title":      "Click CTA",
				"url":        "https://example.com/cta",
				"screenshot": "frames/0002.png",
				"focusedElement": map[string]any{
					"selector": "#cta",
					"boundingBox": map[string]any{
						"x":      100,
						"y":      200,
						"width":  150,
						"height": 40,
					},
				},
			},
		},
	}

	screenshots := map[string][]byte{
		"frames/0001.png": makeTestPNG(t, 180),
		"frames/0002.png": makeTestPNG(t, 220),
	}

	writeRecordingArchive(t, archivePath, manifest, screenshots)
}

func TestRecordingServiceResolveProject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	manifest := &recordingManifest{}

	t.Run("prefers explicit project ID", func(t *testing.T) {
		repo := newFakeRecordingRepo()
		service := newTestRecordingService(t, repo)

		project := &database.Project{
			ID:         uuid.New(),
			Name:       "Explicit Project",
			FolderPath: t.TempDir(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("failed to seed project: %v", err)
		}

		opts := RecordingImportOptions{ProjectID: &project.ID}

		resolved, created, err := service.resolveProject(ctx, manifest, opts)
		if err != nil {
			t.Fatalf("resolveProject returned error: %v", err)
		}
		if created {
			t.Fatalf("expected existing project to be reused")
		}
		if resolved.ID != project.ID {
			t.Fatalf("expected project %s, got %s", project.ID, resolved.ID)
		}
	})

	t.Run("matches by provided name", func(t *testing.T) {
		repo := newFakeRecordingRepo()
		service := newTestRecordingService(t, repo)

		project := &database.Project{
			ID:         uuid.New(),
			Name:       "Named Project",
			FolderPath: t.TempDir(),
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("failed to seed project: %v", err)
		}

		opts := RecordingImportOptions{ProjectName: project.Name}

		resolved, created, err := service.resolveProject(ctx, manifest, opts)
		if err != nil {
			t.Fatalf("resolveProject returned error: %v", err)
		}
		if created {
			t.Fatalf("expected name match to reuse project")
		}
		if resolved.ID != project.ID {
			t.Fatalf("expected project %s, got %s", project.ID, resolved.ID)
		}
	})

	t.Run("creates default extension recordings project", func(t *testing.T) {
		repo := newFakeRecordingRepo()
		service := newTestRecordingService(t, repo)

		resolved, created, err := service.resolveProject(ctx, manifest, RecordingImportOptions{})
		if err != nil {
			t.Fatalf("resolveProject returned error: %v", err)
		}
		if !created {
			t.Fatalf("expected project to be created")
		}
		if resolved.Name != "Extension Recordings" {
			t.Fatalf("expected default extension recordings project, got %s", resolved.Name)
		}
		if _, err := os.Stat(resolved.FolderPath); err != nil {
			t.Fatalf("expected project directory to exist, got error: %v", err)
		}
	})
}

func TestRecordingServiceResolveWorkflow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	project := &database.Project{
		ID:         uuid.New(),
		Name:       "Workflow Project",
		FolderPath: "projects/workflow",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	t.Run("prefers explicit workflow ID", func(t *testing.T) {
		repo := newFakeRecordingRepo()
		service := newTestRecordingService(t, repo)
		_ = repo.CreateProject(ctx, project)

		workflow := &database.Workflow{
			ID:         uuid.New(),
			ProjectID:  &project.ID,
			Name:       "Existing Workflow",
			FolderPath: defaultRecordingWorkflowFolder,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("failed to seed workflow: %v", err)
		}

		opts := RecordingImportOptions{WorkflowID: &workflow.ID}

		resolved, created, err := service.resolveWorkflow(ctx, &recordingManifest{}, project, opts)
		if err != nil {
			t.Fatalf("resolveWorkflow returned error: %v", err)
		}
		if created {
			t.Fatalf("expected workflow reuse")
		}
		if resolved.ID != workflow.ID {
			t.Fatalf("expected workflow %s, got %s", workflow.ID, resolved.ID)
		}
	})

	t.Run("matches by workflow name", func(t *testing.T) {
		repo := newFakeRecordingRepo()
		service := newTestRecordingService(t, repo)
		_ = repo.CreateProject(ctx, project)

		workflow := &database.Workflow{
			ID:         uuid.New(),
			ProjectID:  &project.ID,
			Name:       "Manifest Workflow",
			FolderPath: defaultRecordingWorkflowFolder,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("failed to seed workflow: %v", err)
		}

		opts := RecordingImportOptions{WorkflowName: workflow.Name}
		resolved, created, err := service.resolveWorkflow(ctx, &recordingManifest{}, project, opts)
		if err != nil {
			t.Fatalf("resolveWorkflow returned error: %v", err)
		}
		if created {
			t.Fatalf("expected named workflow reuse")
		}
		if resolved.ID != workflow.ID {
			t.Fatalf("expected workflow %s, got %s", workflow.ID, resolved.ID)
		}
	})

	t.Run("creates workflow using manifest metadata", func(t *testing.T) {
		repo := newFakeRecordingRepo()
		service := newTestRecordingService(t, repo)
		_ = repo.CreateProject(ctx, project)

		manifest := &recordingManifest{
			RunID:        "run-123",
			WorkflowName: "Recording Workflow",
		}

		resolved, created, err := service.resolveWorkflow(ctx, manifest, project, RecordingImportOptions{})
		if err != nil {
			t.Fatalf("resolveWorkflow returned error: %v", err)
		}
		if !created {
			t.Fatalf("expected workflow to be created")
		}
		if resolved.Name != manifest.WorkflowName {
			t.Fatalf("expected workflow name %q, got %q", manifest.WorkflowName, resolved.Name)
		}
		if resolved.ProjectID == nil || *resolved.ProjectID != project.ID {
			t.Fatalf("expected workflow project %s, got %v", project.ID, resolved.ProjectID)
		}
		if resolved.FolderPath != defaultRecordingWorkflowFolder {
			t.Fatalf("expected folder %s, got %s", defaultRecordingWorkflowFolder, resolved.FolderPath)
		}
		if _, ok := repo.workflows[resolved.ID]; !ok {
			t.Fatalf("expected workflow to be persisted")
		}
	})
}

func TestSelectRecordingStorage(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("forces local store when BAS_RECORDING_STORAGE=local", func(t *testing.T) {
		t.Setenv("BAS_RECORDING_STORAGE", "local")
		provided := &stubStorage{}

		store := selectRecordingStorage(provided, t.TempDir(), log)
		if _, ok := store.(*recordingFileStore); !ok {
			t.Fatalf("expected recordingFileStore when forcing local mode, got %T", store)
		}
	})

	t.Run("uses provided store when env unset", func(t *testing.T) {
		t.Setenv("BAS_RECORDING_STORAGE", "")
		provided := &stubStorage{}

		store := selectRecordingStorage(provided, t.TempDir(), log)
		if store != provided {
			t.Fatalf("expected provided storage to be used")
		}
	})

	t.Run("falls back to local store when none provided", func(t *testing.T) {
		t.Setenv("BAS_RECORDING_STORAGE", "")

		store := selectRecordingStorage(nil, t.TempDir(), log)
		if _, ok := store.(*recordingFileStore); !ok {
			t.Fatalf("expected recordingFileStore fallback, got %T", store)
		}
	})
}

func writeRecordingArchive(t *testing.T, archivePath string, manifest map[string]any, screenshots map[string][]byte) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	manifestWriter, err := writer.Create("manifest.json")
	if err != nil {
		t.Fatalf("failed to create manifest entry: %v", err)
	}
	if _, err := manifestWriter.Write(manifestBytes); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	for name, data := range screenshots {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("failed to create entry %s: %v", name, err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatalf("failed to write entry %s: %v", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close archive: %v", err)
	}
}

func makeTestPNG(t *testing.T, colorValue uint8) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for x := 0; x < 4; x++ {
		for y := 0; y < 4; y++ {
			img.Set(x, y, color.RGBA{R: colorValue, G: colorValue, B: colorValue, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
	return buf.Bytes()
}

func newTestRecordingService(t *testing.T, repo RecordingRepository) *RecordingService {
	t.Helper()

	log := logrus.New()
	log.SetOutput(io.Discard)
	return NewRecordingService(repo, nil, nil, log, t.TempDir())
}

type stubStorage struct{}

func (*stubStorage) GetScreenshot(context.Context, string) (io.ReadCloser, *minio.ObjectInfo, error) {
	return nil, nil, nil
}

func (*stubStorage) StoreScreenshot(context.Context, uuid.UUID, string, []byte, string) (*storage.ScreenshotInfo, error) {
	return nil, nil
}

func (*stubStorage) StoreScreenshotFromFile(context.Context, uuid.UUID, string, string) (*storage.ScreenshotInfo, error) {
	return nil, nil
}

func (*stubStorage) DeleteScreenshot(context.Context, string) error { return nil }

func (*stubStorage) ListExecutionScreenshots(context.Context, uuid.UUID) ([]string, error) {
	return nil, nil
}

func (*stubStorage) HealthCheck(context.Context) error { return nil }

func (*stubStorage) GetBucketName() string { return "stub" }
