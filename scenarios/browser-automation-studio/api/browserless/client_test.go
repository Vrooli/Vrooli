package browserless

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/database"
)

type mockRepository struct {
	folders           map[string]*database.WorkflowFolder
	executionLogs     []*database.ExecutionLog
	screenshots       []*database.Screenshot
	updatedExecutions []*database.Execution
}

func newMockRepository() *mockRepository {
	return &mockRepository{folders: make(map[string]*database.WorkflowFolder)}
}

// Project operations
type noopProject struct{}

func (m *mockRepository) CreateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *mockRepository) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) UpdateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *mockRepository) DeleteProject(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockRepository) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, nil
}
func (m *mockRepository) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]interface{}, error) {
	return nil, nil
}

// Workflow operations
func (m *mockRepository) CreateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *mockRepository) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) GetWorkflowByName(ctx context.Context, name, folderPath string) (*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) UpdateWorkflow(ctx context.Context, workflow *database.Workflow) error {
	return nil
}
func (m *mockRepository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockRepository) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *mockRepository) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}

// Execution operations
func (m *mockRepository) CreateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}
func (m *mockRepository) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return nil, nil
}
func (m *mockRepository) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	copyExec := *execution
	m.updatedExecutions = append(m.updatedExecutions, &copyExec)
	return nil
}
func (m *mockRepository) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}

// Artifact operations
func (m *mockRepository) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	copyShot := *screenshot
	m.screenshots = append(m.screenshots, &copyShot)
	return nil
}
func (m *mockRepository) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}

func (m *mockRepository) CreateExecutionLog(ctx context.Context, logEntry *database.ExecutionLog) error {
	copyLog := *logEntry
	m.executionLogs = append(m.executionLogs, &copyLog)
	return nil
}
func (m *mockRepository) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionLog, error) {
	return nil, nil
}

func (m *mockRepository) CreateExtractedData(ctx context.Context, data *database.ExtractedData) error {
	return nil
}
func (m *mockRepository) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*database.ExtractedData, error) {
	return nil, nil
}

// Folder operations
func (m *mockRepository) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	m.folders[folder.Path] = folder
	return nil
}
func (m *mockRepository) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	if folder, ok := m.folders[path]; ok {
		return folder, nil
	}
	return nil, database.ErrNotFound
}
func (m *mockRepository) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	result := make([]*database.WorkflowFolder, 0, len(m.folders))
	for _, folder := range m.folders {
		result = append(result, folder)
	}
	return result, nil
}

func newTestClient() (*Client, *mockRepository) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	repo := newMockRepository()
	client := &Client{
		log:         log,
		repo:        repo,
		browserless: "http://localhost:0",
		httpClient:  &http.Client{Timeout: 2 * time.Second},
	}
	return client, repo
}

func TestBuildInstructions(t *testing.T) {
	client, _ := newTestClient()

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []interface{}{
				map[string]interface{}{
					"id":   "node-1",
					"type": "navigate",
					"data": map[string]interface{}{
						"url":       "https://example.com",
						"waitUntil": "domcontentloaded",
						"timeoutMs": 10000,
					},
				},
				map[string]interface{}{
					"id":   "node-2",
					"type": "wait",
					"data": map[string]interface{}{
						"type":     "time",
						"duration": 1500,
					},
				},
				map[string]interface{}{
					"id":   "node-3",
					"type": "screenshot",
					"data": map[string]interface{}{
						"name":           "after",
						"viewportWidth":  1280,
						"viewportHeight": 720,
					},
				},
			},
		},
	}

	instr, err := client.buildInstructions(workflow)
	if err != nil {
		t.Fatalf("buildInstructions failed: %v", err)
	}

	if len(instr) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(instr))
	}

	if instr[0].Params.URL != "https://example.com" {
		t.Errorf("navigate url not parsed: %+v", instr[0].Params)
	}

	if instr[1].Params.DurationMs != 1500 {
		t.Errorf("wait duration mismatch: %+v", instr[1].Params)
	}

	if instr[2].Params.Name != "after" || instr[2].Params.ViewportWidth != 1280 {
		t.Errorf("screenshot params mismatch: %+v", instr[2].Params)
	}
}

func TestBuildInstructionsUnsupportedNode(t *testing.T) {
	client, _ := newTestClient()

	workflow := &database.Workflow{
		ID: uuid.New(),
		FlowDefinition: database.JSONMap{
			"nodes": []interface{}{
				map[string]interface{}{
					"id":   "node-1",
					"type": "custom",
					"data": map[string]interface{}{},
				},
			},
		},
	}

	if _, err := client.buildInstructions(workflow); err == nil {
		t.Fatalf("expected error for unsupported node type")
	}
}

func TestBuildWorkflowScript(t *testing.T) {
	instr := []runtime.Instruction{
		{Index: 0, NodeID: "node-1", Type: "navigate", Params: runtime.InstructionParam{URL: "https://example.com"}},
	}

	script, err := runtime.BuildWorkflowScript(instr)
	if err != nil {
		t.Fatalf("buildWorkflowScript failed: %v", err)
	}

	if !strings.Contains(script, "https://example.com") {
		t.Errorf("script does not contain instruction payload: %s", script)
	}

	if strings.Contains(script, "__INSTRUCTIONS__") {
		t.Errorf("placeholder not replaced in script: %s", script)
	}
}

func TestStoreScreenshotFallback(t *testing.T) {
	client, repo := newTestClient()
	client.storage = nil

	exec := &database.Execution{ID: uuid.New()}
	step := runtime.StepResult{Index: 0, Type: "screenshot", StepName: "test"}

	t.Setenv("TMPDIR", t.TempDir())

	encoded := base64.StdEncoding.EncodeToString([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	step.ScreenshotBase64 = encoded

	record, err := client.persistScreenshot(context.Background(), exec, step)
	if err != nil {
		t.Fatalf("persistScreenshot failed: %v", err)
	}
	if record == nil {
		t.Fatalf("expected screenshot record")
	}

	if len(repo.screenshots) != 1 {
		t.Fatalf("expected screenshot record to be persisted")
	}

	path := repo.screenshots[0].StorageURL
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fallback screenshot not written: %v", err)
	}
}

func TestCheckBrowserlessHealth(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/pressure" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := newTestClient()
		client.browserless = server.URL
		client.httpClient = server.Client()

		if err := client.CheckBrowserlessHealth(); err != nil {
			t.Fatalf("expected healthy status, got error: %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		defer server.Close()

		client, _ := newTestClient()
		client.browserless = server.URL
		client.httpClient = server.Client()

		if err := client.CheckBrowserlessHealth(); err == nil {
			t.Fatalf("expected error for failing health check")
		}
	})
}

func TestNewClient(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	repo := newMockRepository()
	t.Setenv("BROWSERLESS_URL", "http://localhost:3000")

	client := NewClient(log, repo)
	if client == nil {
		t.Fatalf("expected client instance")
	}

	if client.httpClient == nil {
		t.Fatalf("expected http client to be configured")
	}
}
