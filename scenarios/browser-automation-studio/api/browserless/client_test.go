package browserless

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// Mock repository for testing
type mockRepository struct {
	folders map[string]*database.WorkflowFolder
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		folders: make(map[string]*database.WorkflowFolder),
	}
}

func (m *mockRepository) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	folder, ok := m.folders[path]
	if !ok {
		return nil, database.ErrNotFound
	}
	return folder, nil
}

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

func (m *mockRepository) DeleteProject(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockRepository) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, nil
}

func (m *mockRepository) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]interface{}, error) {
	return nil, nil
}

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

func (m *mockRepository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockRepository) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}

func (m *mockRepository) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}

func (m *mockRepository) CreateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}

func (m *mockRepository) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return nil, nil
}

func (m *mockRepository) UpdateExecution(ctx context.Context, execution *database.Execution) error {
	return nil
}

func (m *mockRepository) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}

func (m *mockRepository) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	return nil
}

func (m *mockRepository) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}

func (m *mockRepository) CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error {
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

func (m *mockRepository) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	m.folders[folder.Path] = folder
	return nil
}

func (m *mockRepository) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	result := make([]*database.WorkflowFolder, 0, len(m.folders))
	for _, folder := range m.folders {
		result = append(result, folder)
	}
	return result, nil
}

func setupTestClient() (*Client, *mockRepository) {
	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := newMockRepository()

	// Set environment variable to prevent client from failing
	os.Setenv("BROWSERLESS_URL", "http://localhost:3000")
	defer os.Unsetenv("BROWSERLESS_URL")

	client := NewClient(log, repo)

	return client, repo
}

func TestNewClient(t *testing.T) {
	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := newMockRepository()

	t.Run("Success", func(t *testing.T) {
		os.Setenv("BROWSERLESS_URL", "http://localhost:3000")
		defer os.Unsetenv("BROWSERLESS_URL")

		client := NewClient(log, repo)
		if client == nil {
			t.Fatal("Expected client to be created")
		}

		if client.log == nil {
			t.Error("Expected logger to be set")
		}

		if client.repo == nil {
			t.Error("Expected repository to be set")
		}
	})
}

func TestCheckBrowserlessHealth(t *testing.T) {
	client, _ := setupTestClient()

	t.Run("HealthCheck", func(t *testing.T) {
		// Test that health check can be called
		err := client.CheckBrowserlessHealth()
		if err != nil {
			t.Logf("Health check failed as expected (browserless not running): %v", err)
		} else {
			t.Log("Health check passed (browserless is running)")
		}
	})
}

func TestClientConfiguration(t *testing.T) {
	client, _ := setupTestClient()

	t.Run("HasLogger", func(t *testing.T) {
		if client.log == nil {
			t.Error("Client should have a logger")
		}
	})

	t.Run("HasRepository", func(t *testing.T) {
		if client.repo == nil {
			t.Error("Client should have a repository")
		}
	})
}

func TestBrowserSession(t *testing.T) {
	t.Run("CreateSession", func(t *testing.T) {
		session := &BrowserSession{
			ID:          uuid.New().String(),
			ExecutionID: uuid.New(),
			URL:         "https://example.com",
		}

		if session.ID == "" {
			t.Error("Session ID should not be empty")
		}

		if session.ExecutionID == uuid.Nil {
			t.Error("Execution ID should not be nil")
		}

		if session.URL == "" {
			t.Error("URL should not be empty")
		}
	})
}
