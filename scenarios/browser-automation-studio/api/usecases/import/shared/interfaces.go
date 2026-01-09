// Package shared provides common interfaces and utilities for import usecases.
// These interfaces enable dependency injection and testing seams.
package shared

import (
	"context"
	"os"

	"github.com/google/uuid"
)

// DirectoryScanner abstracts filesystem operations for scanning directories.
// This interface allows mocking filesystem operations in tests.
type DirectoryScanner interface {
	// ScanDirectory lists entries in a directory
	ScanDirectory(ctx context.Context, path string) ([]FileEntry, error)

	// ScanForPattern recursively finds files matching a pattern
	ScanForPattern(ctx context.Context, root string, pattern string, maxDepth int) ([]FileEntry, error)

	// ReadFile reads a file's contents
	ReadFile(ctx context.Context, path string) ([]byte, error)

	// WriteFile writes content to a file, creating parent directories if needed
	WriteFile(ctx context.Context, path string, content []byte, perm os.FileMode) error

	// CopyFile copies a file from src to dst
	CopyFile(ctx context.Context, src, dst string) error

	// Exists checks if a path exists
	Exists(ctx context.Context, path string) (bool, error)

	// IsDir checks if a path is a directory
	IsDir(ctx context.Context, path string) (bool, error)

	// Stat returns file info for a path
	Stat(ctx context.Context, path string) (os.FileInfo, error)
}

// FileEntry represents a file or directory entry from scanning.
type FileEntry struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
}

// WorkflowIndexer abstracts database operations for workflow indexing.
// This interface allows mocking database operations in tests.
type WorkflowIndexer interface {
	// CreateWorkflowIndex creates a new workflow index entry
	CreateWorkflowIndex(ctx context.Context, projectID uuid.UUID, workflow *WorkflowIndexData) error

	// GetWorkflowByFilePath retrieves a workflow by its file path within a project
	GetWorkflowByFilePath(ctx context.Context, projectID uuid.UUID, filePath string) (*WorkflowIndexData, error)

	// GetWorkflowByID retrieves a workflow by ID
	GetWorkflowByID(ctx context.Context, id uuid.UUID) (*WorkflowIndexData, error)

	// UpdateWorkflowIndex updates an existing workflow index entry
	UpdateWorkflowIndex(ctx context.Context, workflow *WorkflowIndexData) error

	// ListWorkflowsByProject lists all workflows for a project
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID) ([]*WorkflowIndexData, error)

	// DeleteWorkflowIndex deletes a workflow index entry
	DeleteWorkflowIndex(ctx context.Context, id uuid.UUID) error
}

// WorkflowIndexData represents workflow index information.
type WorkflowIndexData struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Description string
	FilePath    string // Relative path within project
	Version     int
	NodeCount   int
	EdgeCount   int
	Tags        []string
}

// ProjectIndexer abstracts database operations for project indexing.
type ProjectIndexer interface {
	// GetProjectByID retrieves a project by ID
	GetProjectByID(ctx context.Context, id uuid.UUID) (*ProjectIndexData, error)

	// GetProjectByFolderPath retrieves a project by its folder path
	GetProjectByFolderPath(ctx context.Context, folderPath string) (*ProjectIndexData, error)

	// ListProjects lists all projects with pagination
	ListProjects(ctx context.Context, limit, offset int) ([]*ProjectIndexData, error)

	// CreateProject creates a new project
	CreateProject(ctx context.Context, project *ProjectIndexData) error
}

// ProjectIndexData represents project index information.
type ProjectIndexData struct {
	ID          uuid.UUID
	Name        string
	Description string
	FolderPath  string
}

// WorkflowSyncer abstracts workflow synchronization operations.
// This interface allows triggering workflow sync after project import.
type WorkflowSyncer interface {
	// SyncProjectWorkflows synchronizes the workflow DB index for a project from the filesystem
	SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error
}

// WorkflowValidator abstracts workflow validation operations.
type WorkflowValidator interface {
	// ValidateWorkflowJSON validates workflow JSON content
	ValidateWorkflowJSON(content []byte) (*WorkflowValidationResult, error)
}

// WorkflowValidationResult contains the result of workflow validation.
type WorkflowValidationResult struct {
	IsValid      bool
	Errors       []ValidationIssue
	Warnings     []ValidationIssue
	NodeCount    int
	EdgeCount    int
	HasStartNode bool
	HasEndNode   bool
}

// ValidationIssue represents a validation error or warning.
type ValidationIssue struct {
	Code    string
	Message string
	Path    string // JSON path to the issue
}
