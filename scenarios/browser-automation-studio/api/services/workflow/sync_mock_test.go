package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// MockWorkflowSyncRepository is a test double for WorkflowSyncRepository.
// It tracks method calls and allows error injection for testing error paths.
type MockWorkflowSyncRepository struct {
	// Storage
	Projects  map[uuid.UUID]*database.ProjectIndex
	Workflows map[uuid.UUID]*database.WorkflowIndex
	Assets    map[uuid.UUID]*database.AssetIndex

	// Error injection
	GetProjectError             error
	ListWorkflowsByProjectError error
	CreateWorkflowError         error
	UpdateWorkflowError         error
	DeleteWorkflowError         error
	ListAssetsByProjectError    error
	CreateAssetError            error
	UpdateAssetError            error
	DeleteAssetError            error

	// Call tracking
	CreateWorkflowCalls []database.WorkflowIndex
	UpdateWorkflowCalls []database.WorkflowIndex
	DeleteWorkflowCalls []uuid.UUID
	CreateAssetCalls    []database.AssetIndex
	UpdateAssetCalls    []database.AssetIndex
	DeleteAssetCalls    []uuid.UUID
}

// NewMockWorkflowSyncRepository creates a new mock repository for testing.
func NewMockWorkflowSyncRepository() *MockWorkflowSyncRepository {
	return &MockWorkflowSyncRepository{
		Projects:  make(map[uuid.UUID]*database.ProjectIndex),
		Workflows: make(map[uuid.UUID]*database.WorkflowIndex),
		Assets:    make(map[uuid.UUID]*database.AssetIndex),
	}
}

// Compile-time check that MockWorkflowSyncRepository implements WorkflowSyncRepository
var _ WorkflowSyncRepository = (*MockWorkflowSyncRepository)(nil)

func (m *MockWorkflowSyncRepository) GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error) {
	if m.GetProjectError != nil {
		return nil, m.GetProjectError
	}
	project, ok := m.Projects[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	copy := *project
	return &copy, nil
}

func (m *MockWorkflowSyncRepository) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error) {
	if m.ListWorkflowsByProjectError != nil {
		return nil, m.ListWorkflowsByProjectError
	}

	var result []*database.WorkflowIndex
	for _, w := range m.Workflows {
		if w.ProjectID != nil && *w.ProjectID == projectID {
			copy := *w
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *MockWorkflowSyncRepository) GetWorkflowByNameInProject(ctx context.Context, projectID uuid.UUID, name, folderPath string) (*database.WorkflowIndex, error) {
	for _, w := range m.Workflows {
		if w.ProjectID != nil && *w.ProjectID == projectID && w.Name == name && w.FolderPath == folderPath {
			copy := *w
			return &copy, nil
		}
	}
	return nil, database.ErrNotFound
}

func (m *MockWorkflowSyncRepository) CreateWorkflow(ctx context.Context, workflow *database.WorkflowIndex) error {
	if m.CreateWorkflowError != nil {
		return m.CreateWorkflowError
	}
	m.CreateWorkflowCalls = append(m.CreateWorkflowCalls, *workflow)
	copy := *workflow
	m.Workflows[workflow.ID] = &copy
	return nil
}

func (m *MockWorkflowSyncRepository) UpdateWorkflow(ctx context.Context, workflow *database.WorkflowIndex) error {
	if m.UpdateWorkflowError != nil {
		return m.UpdateWorkflowError
	}
	m.UpdateWorkflowCalls = append(m.UpdateWorkflowCalls, *workflow)
	copy := *workflow
	m.Workflows[workflow.ID] = &copy
	return nil
}

func (m *MockWorkflowSyncRepository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	if m.DeleteWorkflowError != nil {
		return m.DeleteWorkflowError
	}
	m.DeleteWorkflowCalls = append(m.DeleteWorkflowCalls, id)
	delete(m.Workflows, id)
	return nil
}

func (m *MockWorkflowSyncRepository) ListAssetsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.AssetIndex, error) {
	if m.ListAssetsByProjectError != nil {
		return nil, m.ListAssetsByProjectError
	}

	var result []*database.AssetIndex
	for _, a := range m.Assets {
		if a.ProjectID == projectID {
			copy := *a
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *MockWorkflowSyncRepository) CreateAsset(ctx context.Context, asset *database.AssetIndex) error {
	if m.CreateAssetError != nil {
		return m.CreateAssetError
	}
	m.CreateAssetCalls = append(m.CreateAssetCalls, *asset)
	copy := *asset
	m.Assets[asset.ID] = &copy
	return nil
}

func (m *MockWorkflowSyncRepository) UpdateAsset(ctx context.Context, asset *database.AssetIndex) error {
	if m.UpdateAssetError != nil {
		return m.UpdateAssetError
	}
	m.UpdateAssetCalls = append(m.UpdateAssetCalls, *asset)
	copy := *asset
	m.Assets[asset.ID] = &copy
	return nil
}

func (m *MockWorkflowSyncRepository) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	if m.DeleteAssetError != nil {
		return m.DeleteAssetError
	}
	m.DeleteAssetCalls = append(m.DeleteAssetCalls, id)
	delete(m.Assets, id)
	return nil
}

// Test helpers

// AddProject adds a project to the mock repository.
func (m *MockWorkflowSyncRepository) AddProject(project *database.ProjectIndex) {
	copy := *project
	m.Projects[project.ID] = &copy
}

// AddWorkflow adds a workflow to the mock repository.
func (m *MockWorkflowSyncRepository) AddWorkflow(workflow *database.WorkflowIndex) {
	copy := *workflow
	m.Workflows[workflow.ID] = &copy
}

// AddAsset adds an asset to the mock repository.
func (m *MockWorkflowSyncRepository) AddAsset(asset *database.AssetIndex) {
	copy := *asset
	m.Assets[asset.ID] = &copy
}
