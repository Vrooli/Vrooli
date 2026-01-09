// Package routines provides the routine/workflow import usecase.
package routines

// InspectRoutineRequest is the request for inspecting a workflow file before import.
type InspectRoutineRequest struct {
	// FilePath is the absolute path to the workflow file
	FilePath string `json:"file_path"`
}

// InspectRoutineResponse contains the result of inspecting a workflow file.
type InspectRoutineResponse struct {
	// FilePath is the normalized absolute path
	FilePath string `json:"file_path"`
	// Exists indicates if the file exists
	Exists bool `json:"exists"`
	// IsValid indicates if the file is a valid workflow
	IsValid bool `json:"is_valid"`
	// ValidationError contains the error message if validation failed
	ValidationError string `json:"validation_error,omitempty"`
	// AlreadyIndexed indicates if this workflow is already in the database
	AlreadyIndexed bool `json:"already_indexed"`
	// IndexedID is the workflow ID if already indexed
	IndexedID string `json:"indexed_id,omitempty"`
	// Preview contains workflow metadata for preview
	Preview *WorkflowPreview `json:"preview,omitempty"`
}

// WorkflowPreview contains workflow metadata for display before import.
type WorkflowPreview struct {
	// ID from the workflow file (if present)
	ID string `json:"id,omitempty"`
	// Name of the workflow
	Name string `json:"name"`
	// Description of the workflow
	Description string `json:"description,omitempty"`
	// NodeCount is the number of nodes in the workflow
	NodeCount int `json:"node_count"`
	// EdgeCount is the number of edges in the workflow
	EdgeCount int `json:"edge_count"`
	// Tags associated with the workflow
	Tags []string `json:"tags,omitempty"`
	// Version of the workflow
	Version int `json:"version"`
	// HasStartNode indicates if the workflow has a start node
	HasStartNode bool `json:"has_start_node"`
	// HasEndNode indicates if the workflow has an end node
	HasEndNode bool `json:"has_end_node"`
}

// ImportRoutineRequest is the request for importing a workflow file.
type ImportRoutineRequest struct {
	// FilePath is the absolute path to the source workflow file
	FilePath string `json:"file_path"`
	// DestPath is the optional destination path within the project's workflows directory
	DestPath string `json:"dest_path,omitempty"`
	// Name is an optional override for the workflow name
	Name string `json:"name,omitempty"`
	// OverwriteIfExists allows overwriting an existing workflow at the destination
	OverwriteIfExists bool `json:"overwrite_if_exists"`
}

// ImportRoutineResponse contains the result of importing a workflow.
type ImportRoutineResponse struct {
	// WorkflowID is the ID of the imported workflow
	WorkflowID string `json:"workflow_id"`
	// Name of the imported workflow
	Name string `json:"name"`
	// Path is the relative path within the project
	Path string `json:"path"`
	// Warnings contains any non-fatal issues encountered during import
	Warnings []string `json:"warnings,omitempty"`
}

// ScanRoutinesRequest is the request for scanning a directory for workflow files.
type ScanRoutinesRequest struct {
	// Path is the directory to scan (defaults to project's workflows directory)
	Path string `json:"path,omitempty"`
	// Depth is the scan depth (1 or 2)
	Depth int `json:"depth,omitempty"`
}

// ScanRoutinesResponse contains the result of scanning for workflow files.
type ScanRoutinesResponse struct {
	// Path is the scanned directory
	Path string `json:"path"`
	// Parent is the parent directory (null if at root)
	Parent *string `json:"parent"`
	// Entries contains the found workflow files and directories
	Entries []RoutineEntry `json:"entries"`
}

// RoutineEntry represents a workflow file or directory in scan results.
type RoutineEntry struct {
	// Name is the file/directory name
	Name string `json:"name"`
	// Path is the absolute path
	Path string `json:"path"`
	// IsValid indicates if this is a valid workflow file
	IsValid bool `json:"is_valid"`
	// IsRegistered indicates if this workflow is already indexed
	IsRegistered bool `json:"is_registered"`
	// WorkflowID is the ID if already indexed
	WorkflowID string `json:"workflow_id,omitempty"`
	// PreviewName is the workflow name from the file
	PreviewName string `json:"preview_name,omitempty"`
}
