package workflow

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// WorkflowSyncRepository provides database operations for workflow synchronization.
// This interface enables testing sync logic without a real database.
//
// ## Design Rationale
//
// The sync system needs to:
// 1. Load existing database state into memory for O(1) lookups
// 2. Upsert records to match filesystem state
// 3. Delete stale records (garbage collection)
//
// By injecting this interface, we can test sync logic with a mock repository
// that tracks calls and returns controlled responses.
//
// ## Usage
//
// Production code uses database.Repository (which implements this interface).
// Tests can use MockWorkflowSyncRepository for isolated unit testing.
type WorkflowSyncRepository interface {
	// Project operations
	GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error)

	// Workflow operations
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error)
	GetWorkflowByNameInProject(ctx context.Context, projectID uuid.UUID, name, folderPath string) (*database.WorkflowIndex, error)
	CreateWorkflow(ctx context.Context, workflow *database.WorkflowIndex) error
	UpdateWorkflow(ctx context.Context, workflow *database.WorkflowIndex) error
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error

	// Asset operations
	ListAssetsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.AssetIndex, error)
	CreateAsset(ctx context.Context, asset *database.AssetIndex) error
	UpdateAsset(ctx context.Context, asset *database.AssetIndex) error
	DeleteAsset(ctx context.Context, id uuid.UUID) error
}

// Note: database.Repository is a superset of WorkflowSyncRepository.
// Any concrete implementation of database.Repository (like database.repository)
// will automatically satisfy WorkflowSyncRepository.

// DBState holds loaded database records for O(1) lookup during sync.
// This intermediate type reduces repeated database queries during filesystem walks.
type DBState struct {
	// WorkflowsByID maps workflow UUID to database record for fast existence checks
	WorkflowsByID map[uuid.UUID]*database.WorkflowIndex

	// WorkflowsByNameKey maps "projectID:name:folderPath" to workflow for deduplication.
	// This is used to detect when an external workflow with the same name and folder
	// already exists in the database, allowing us to reuse its ID instead of creating duplicates.
	WorkflowsByNameKey map[string]*database.WorkflowIndex

	// AssetsByPath maps relative file path to asset record for deduplication
	AssetsByPath map[string]*database.AssetIndex
}

// WorkflowNameKey generates a lookup key for workflow deduplication.
// The key format is "projectID:name:folderPath" to ensure uniqueness within a project.
func WorkflowNameKey(projectID uuid.UUID, name, folderPath string) string {
	return fmt.Sprintf("%s:%s:%s", projectID.String(), name, folderPath)
}

// NewDBState creates an empty DBState for sync operations.
func NewDBState() *DBState {
	return &DBState{
		WorkflowsByID:      make(map[uuid.UUID]*database.WorkflowIndex),
		WorkflowsByNameKey: make(map[string]*database.WorkflowIndex),
		AssetsByPath:       make(map[string]*database.AssetIndex),
	}
}

// SeenState tracks what was found on filesystem during a sync operation.
// After walking the filesystem, anything in DBState but NOT in SeenState is stale.
type SeenState struct {
	// WorkflowIDs tracks which workflows were found on disk
	WorkflowIDs map[uuid.UUID]bool

	// AssetPaths tracks which assets were found on disk
	AssetPaths map[string]bool
}

// NewSeenState creates an empty SeenState for sync operations.
func NewSeenState() *SeenState {
	return &SeenState{
		WorkflowIDs: make(map[uuid.UUID]bool),
		AssetPaths:  make(map[string]bool),
	}
}

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
