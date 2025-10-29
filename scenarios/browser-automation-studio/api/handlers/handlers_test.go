package handlers

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless"
	"github.com/vrooli/browser-automation-studio/database"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// mockRepository implements database.Repository for testing
type mockRepository struct{}

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
func (m *mockRepository) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
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
func (m *mockRepository) CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *mockRepository) UpdateExecutionStep(ctx context.Context, step *database.ExecutionStep) error {
	return nil
}
func (m *mockRepository) ListExecutionSteps(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionStep, error) {
	return nil, nil
}
func (m *mockRepository) CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error {
	return nil
}
func (m *mockRepository) ListExecutionArtifacts(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionArtifact, error) {
	return nil, nil
}
func (m *mockRepository) CreateExecutionLog(ctx context.Context, log *database.ExecutionLog) error {
	return nil
}
func (m *mockRepository) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*database.ExecutionLog, error) {
	return nil, nil
}
func (m *mockRepository) CreateScreenshot(ctx context.Context, screenshot *database.Screenshot) error {
	return nil
}
func (m *mockRepository) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}
func (m *mockRepository) CreateExtractedData(ctx context.Context, data *database.ExtractedData) error {
	return nil
}
func (m *mockRepository) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*database.ExtractedData, error) {
	return nil, nil
}
func (m *mockRepository) CreateFolder(ctx context.Context, folder *database.WorkflowFolder) error {
	return nil
}
func (m *mockRepository) GetFolder(ctx context.Context, path string) (*database.WorkflowFolder, error) {
	return nil, nil
}
func (m *mockRepository) ListFolders(ctx context.Context) ([]*database.WorkflowFolder, error) {
	return nil, nil
}

// Ensure mockRepository implements the interface at compile time
var _ database.Repository = (*mockRepository)(nil)

// TestNewHandler verifies handler initialization with all dependencies
func TestNewHandler(t *testing.T) {
	// Set required environment variables for testing
	t.Setenv("BROWSERLESS_URL", "http://localhost:3000")

	log := logrus.New()
	log.SetOutput(io.Discard) // Suppress output during tests

	// Create mock dependencies
	repo := &mockRepository{}
	browserlessClient := browserless.NewClient(log, repo)
	hub := wsHub.NewHub(log)

	// Initialize handler
	handler := NewHandler(repo, browserlessClient, hub, log)

	if handler == nil {
		t.Fatal("Expected handler to be initialized, got nil")
	}

	if handler.log == nil {
		t.Error("Expected logger to be set")
	}

	if handler.repo == nil {
		t.Error("Expected repository to be set")
	}

	if handler.browserless == nil {
		t.Error("Expected browserless client to be set")
	}

	if handler.wsHub == nil {
		t.Error("Expected WebSocket hub to be set")
	}

	if handler.workflowService == nil {
		t.Error("Expected workflow service to be initialized")
	}
}

// TestHandlerUpgraderConfiguration verifies WebSocket upgrader settings
func TestHandlerUpgraderConfiguration(t *testing.T) {
	t.Setenv("BROWSERLESS_URL", "http://localhost:3000")

	log := logrus.New()
	log.SetOutput(io.Discard)

	repo := &mockRepository{}
	browserlessClient := browserless.NewClient(log, repo)
	hub := wsHub.NewHub(log)

	handler := NewHandler(repo, browserlessClient, hub, log)

	// Verify upgrader allows cross-origin by default (needed for iframe embedding)
	if handler.upgrader.CheckOrigin == nil {
		t.Error("Expected CheckOrigin function to be set")
	}
}

// TestHandlerDependencies verifies all critical dependencies are present
func TestHandlerDependencies(t *testing.T) {
	t.Setenv("BROWSERLESS_URL", "http://localhost:3000")

	log := logrus.New()
	log.SetOutput(io.Discard)

	tests := []struct {
		name        string
		repo        database.Repository
		browserless *browserless.Client
		wsHub       *wsHub.Hub
		shouldPanic bool
	}{
		{
			name:        "all dependencies present",
			repo:        &mockRepository{},
			browserless: browserless.NewClient(log, &mockRepository{}),
			wsHub:       wsHub.NewHub(log),
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.shouldPanic {
					t.Errorf("Expected panic=%v, got panic=%v", tt.shouldPanic, r != nil)
				}
			}()

			handler := NewHandler(tt.repo, tt.browserless, tt.wsHub, log)
			if !tt.shouldPanic && handler == nil {
				t.Error("Expected handler to be created")
			}
		})
	}
}
