package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

type fakeRecordingRepo struct {
	projects        map[uuid.UUID]*database.Project
	projectsByName  map[string]*database.Project
	workflows       map[uuid.UUID]*database.Workflow
	workflowsByName map[string]*database.Workflow
	executions      map[uuid.UUID]*database.Execution
	steps           []*database.ExecutionStep
	artifacts       []*database.ExecutionArtifact
	logs            []*database.ExecutionLog
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
	return nil
}

func (r *fakeRecordingRepo) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
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
		framePath := filepath.Join(recordingsRoot, result.Execution.ID.String(), "frames", "frame-0001.png")
		if _, err := os.Stat(framePath); err != nil {
			t.Fatalf("expected persisted frame asset, got error: %v", err)
		}
	})
}

func createTestRecordingArchive(t *testing.T, archivePath string) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)

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

	pngData := func(colorValue uint8) []byte {
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

	screenshots := map[string][]byte{
		"frames/0001.png": pngData(180),
		"frames/0002.png": pngData(220),
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
