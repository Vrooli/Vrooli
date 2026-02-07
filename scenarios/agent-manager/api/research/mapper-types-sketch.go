// Package research contains sketches for the mapper feature.
// This file is NOT production code - it's a design reference.
package research

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// MAPPER DOMAIN TYPES (SKETCH)
// =============================================================================

// ExplorationStatus indicates how much of a scope area has been explored.
type ExplorationStatus string

const (
	// StatusFog indicates the area has not been explored at all.
	StatusFog ExplorationStatus = "fog"
	// StatusPartial indicates some but not all children have been explored.
	StatusPartial ExplorationStatus = "partial"
	// StatusExplored indicates the area has been fully explored.
	StatusExplored ExplorationStatus = "explored"
	// StatusModified indicates files were modified (implies explored).
	StatusModified ExplorationStatus = "modified"
)

// FileAccess records a single file access by the agent.
type FileAccess struct {
	Path       string    // Absolute or scope-relative path
	AccessType string    // "read", "search", "write", "delete"
	Timestamp  time.Time // When access occurred
	EventID    uuid.UUID // RunEvent that triggered this access
	ToolName   string    // Tool used (Read, Grep, Write, etc.)
	Success    bool      // Whether access succeeded
}

// MapperState holds the current exploration state for a run.
type MapperState struct {
	RunID uuid.UUID `json:"runId"`

	// Files tracks individual file accesses
	// Key is normalized path relative to scope
	Files map[string]FileAccess `json:"files"`

	// Directories tracks aggregated directory exploration
	// Key is directory path relative to scope
	Directories map[string]DirectorySummary `json:"directories"`

	// Scope defines the boundaries of the map
	Scope MapScope `json:"scope"`

	// Progress metrics
	Progress MapperProgress `json:"progress"`

	// LastUpdated is when the map was last modified
	LastUpdated time.Time `json:"lastUpdated"`
}

// DirectorySummary aggregates exploration status for a directory.
type DirectorySummary struct {
	Path            string            `json:"path"`
	Status          ExplorationStatus `json:"status"`
	TotalFiles      int               `json:"totalFiles"`      // Files in this dir (not recursive)
	ExploredFiles   int               `json:"exploredFiles"`   // Files accessed
	ModifiedFiles   int               `json:"modifiedFiles"`   // Files changed
	TotalSubdirs    int               `json:"totalSubdirs"`    // Direct child directories
	ExploredSubdirs int               `json:"exploredSubdirs"` // Subdirs with any exploration
}

// MapScope defines the boundaries of exploration tracking.
type MapScope struct {
	// RootPath is the base path for the map (usually task.ScopePath)
	RootPath string `json:"rootPath"`

	// TotalFiles is the count of files in scope (may be computed lazily)
	TotalFiles int `json:"totalFiles"`

	// TotalDirectories is the count of directories in scope
	TotalDirectories int `json:"totalDirectories"`

	// ComputedAt is when scope stats were calculated
	ComputedAt *time.Time `json:"computedAt,omitempty"`

	// IncludePatterns limits which paths are in scope (globs)
	IncludePatterns []string `json:"includePatterns,omitempty"`

	// ExcludePatterns removes paths from scope (globs)
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
}

// MapperProgress holds computed progress metrics.
type MapperProgress struct {
	// ExploredPercent is (explored files / total files) * 100
	ExploredPercent float64 `json:"exploredPercent"`

	// FogPercent is 100 - ExploredPercent (unexplored areas)
	FogPercent float64 `json:"fogPercent"`

	// ModifiedPercent is (modified files / total files) * 100
	ModifiedPercent float64 `json:"modifiedPercent"`

	// UniqueFilesExplored is count of distinct files accessed
	UniqueFilesExplored int `json:"uniqueFilesExplored"`

	// UniqueFilesModified is count of distinct files changed
	UniqueFilesModified int `json:"uniqueFilesModified"`

	// DirectoriesExplored is count of directories with any access
	DirectoriesExplored int `json:"directoriesExplored"`
}

// =============================================================================
// MAPPER PROCESSOR INTERFACE (SKETCH)
// =============================================================================

// MapperProcessor tracks exploration from event streams.
type MapperProcessor interface {
	// OnToolCall processes a tool invocation event.
	OnToolCall(event ToolCallEventData)

	// OnToolResult processes a tool result event.
	OnToolResult(event ToolResultEventData)

	// GetState returns the current mapper state.
	GetState() *MapperState

	// GetProgress returns computed progress metrics.
	GetProgress() MapperProgress

	// RecordAccess manually records a file access.
	RecordAccess(access FileAccess)

	// SetScope configures the scope boundaries.
	SetScope(scope MapScope)

	// ComputeScopeStats calculates total files in scope (may be slow).
	ComputeScopeStats() error
}

// Placeholder types to make this compile (these exist in domain package)
type ToolCallEventData struct {
	ToolName string
	Input    map[string]interface{}
}

type ToolResultEventData struct {
	ToolName string
	Output   string
	Success  bool
}

// =============================================================================
// TOOL EVENT PARSING (SKETCH)
// =============================================================================

// ExtractPathsFromToolCall extracts file paths from a tool call.
// Returns paths that would be accessed if the call succeeds.
func ExtractPathsFromToolCall(tc ToolCallEventData) []string {
	var paths []string

	switch tc.ToolName {
	case "Read":
		if path, ok := tc.Input["file_path"].(string); ok {
			paths = append(paths, path)
		}
	case "Write":
		if path, ok := tc.Input["file_path"].(string); ok {
			paths = append(paths, path)
		}
	case "Edit":
		if path, ok := tc.Input["file_path"].(string); ok {
			paths = append(paths, path)
		}
	case "Glob":
		// Glob searches a directory - record the search path
		if path, ok := tc.Input["path"].(string); ok {
			paths = append(paths, path)
		}
	case "Grep":
		// Grep searches files - record the search path
		if path, ok := tc.Input["path"].(string); ok {
			paths = append(paths, path)
		}
	}

	return paths
}

// ClassifyToolAccess determines the access type from tool name.
func ClassifyToolAccess(toolName string) string {
	switch toolName {
	case "Read":
		return "read"
	case "Glob", "Grep":
		return "search"
	case "Write":
		return "write"
	case "Edit":
		return "write"
	default:
		return "unknown"
	}
}
