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
// Isolated unit tests can provide an in-memory repository implementation.
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
