// Package types defines the domain model for workspace-sandbox.
//
// # Domain Overview
//
// A sandbox is an isolated, copy-on-write workspace that allows agents or users
// to make changes to a project folder without modifying the original files.
// Changes are captured in an overlay filesystem and can be reviewed, approved,
// or rejected before being applied to the canonical repository.
//
// # Key Concepts
//
//   - Sandbox: An isolated workspace with a specific scope path within a project.
//     Each sandbox has a unique ID and tracks its lifecycle status.
//
//   - Scope Path: The directory within the project that the sandbox covers.
//     Sandboxes cannot have overlapping scopes (see mutual exclusion below).
//
//   - Status: Sandboxes progress through a state machine (see status.go).
//     Key states: creating → active → stopped → approved/rejected → deleted
//
//   - Overlay Layers: The driver creates:
//     - LowerDir: read-only view of the canonical repo
//     - UpperDir: writable layer capturing changes
//     - MergedDir: combined view where agents work
//
// # Mutual Exclusion Rule
//
// Two sandboxes cannot have scopes that overlap. This prevents:
//   - A child sandbox from being affected by changes in a parent scope
//   - A parent sandbox from overwriting changes made in a child scope
//
// The ConflictType enum describes the relationship when scopes overlap.
//
// # Safety Model
//
// This system provides SAFETY FROM ACCIDENTS, not security from adversaries.
// It prevents unintended damage and makes agent work reviewable/revertible,
// but does not create a hardened security boundary.
package types

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status types and state machine logic are in status.go

// OwnerType identifies what kind of entity owns a sandbox.
type OwnerType string

const (
	OwnerTypeAgent  OwnerType = "agent"
	OwnerTypeUser   OwnerType = "user"
	OwnerTypeTask   OwnerType = "task"
	OwnerTypeSystem OwnerType = "system"
)

// ChangeType represents the type of file change.
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeDeleted  ChangeType = "deleted"
)

// ApprovalStatus represents the approval state of a change.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

// Sandbox represents a workspace sandbox with all its metadata.
type Sandbox struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ScopePath   string     `json:"scopePath" db:"scope_path"`
	ProjectRoot string     `json:"projectRoot" db:"project_root"`
	Owner       string     `json:"owner,omitempty" db:"owner"`
	OwnerType   OwnerType  `json:"ownerType" db:"owner_type"`
	Status      Status     `json:"status" db:"status"`
	ErrorMsg    string     `json:"errorMessage,omitempty" db:"error_message"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	LastUsedAt  time.Time  `json:"lastUsedAt" db:"last_used_at"`
	StoppedAt   *time.Time `json:"stoppedAt,omitempty" db:"stopped_at"`
	ApprovedAt  *time.Time `json:"approvedAt,omitempty" db:"approved_at"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`

	// Driver configuration
	Driver        string `json:"driver" db:"driver"`
	DriverVersion string `json:"driverVersion" db:"driver_version"`

	// Mount paths
	LowerDir  string `json:"lowerDir,omitempty" db:"lower_dir"`
	UpperDir  string `json:"upperDir,omitempty" db:"upper_dir"`
	WorkDir   string `json:"workDir,omitempty" db:"work_dir"`
	MergedDir string `json:"mergedDir,omitempty" db:"merged_dir"`

	// Size accounting
	SizeBytes int64 `json:"sizeBytes" db:"size_bytes"`
	FileCount int   `json:"fileCount" db:"file_count"`

	// Session tracking
	ActivePIDs   []int `json:"activePids" db:"active_pids"`
	SessionCount int   `json:"sessionCount" db:"session_count"`

	// Metadata
	Tags     []string               `json:"tags,omitempty" db:"tags"`
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// WorkspacePath returns the path where sandbox operations should occur.
func (s *Sandbox) WorkspacePath() string {
	if s.Status == StatusActive && s.MergedDir != "" {
		return s.MergedDir
	}
	return ""
}

// FileChange represents a single file change in a sandbox.
type FileChange struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	SandboxID      uuid.UUID      `json:"sandboxId" db:"sandbox_id"`
	FilePath       string         `json:"filePath" db:"file_path"`
	ChangeType     ChangeType     `json:"changeType" db:"change_type"`
	FileSize       int64          `json:"fileSize" db:"file_size"`
	FileMode       int            `json:"fileMode" db:"file_mode"`
	DetectedAt     time.Time      `json:"detectedAt" db:"detected_at"`
	ApprovalStatus ApprovalStatus `json:"approvalStatus" db:"approval_status"`
	ApprovedAt     *time.Time     `json:"approvedAt,omitempty" db:"approved_at"`
	ApprovedBy     string         `json:"approvedBy,omitempty" db:"approved_by"`
}

// AuditEvent represents a logged sandbox operation.
type AuditEvent struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	SandboxID    *uuid.UUID             `json:"sandboxId,omitempty" db:"sandbox_id"`
	EventType    string                 `json:"eventType" db:"event_type"`
	EventTime    time.Time              `json:"eventTime" db:"event_time"`
	Actor        string                 `json:"actor,omitempty" db:"actor"`
	ActorType    string                 `json:"actorType" db:"actor_type"`
	Details      map[string]interface{} `json:"details,omitempty" db:"details"`
	SandboxState map[string]interface{} `json:"sandboxState,omitempty" db:"sandbox_state"`
}

// CreateRequest contains the parameters for creating a new sandbox.
type CreateRequest struct {
	ScopePath   string                 `json:"scopePath"`
	ProjectRoot string                 `json:"projectRoot,omitempty"`
	Owner       string                 `json:"owner,omitempty"`
	OwnerType   OwnerType              `json:"ownerType,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ListFilter contains filters for listing sandboxes.
type ListFilter struct {
	Status      []Status  `json:"status,omitempty"`
	Owner       string    `json:"owner,omitempty"`
	ProjectRoot string    `json:"projectRoot,omitempty"`
	ScopePath   string    `json:"scopePath,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedFrom time.Time `json:"createdFrom,omitempty"`
	CreatedTo   time.Time `json:"createdTo,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	Offset      int       `json:"offset,omitempty"`
}

// ListResult contains the result of a sandbox list operation.
type ListResult struct {
	Sandboxes  []*Sandbox `json:"sandboxes"`
	TotalCount int        `json:"totalCount"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
}

// DiffResult contains the diff output for a sandbox.
type DiffResult struct {
	SandboxID     uuid.UUID     `json:"sandboxId"`
	Files         []*FileChange `json:"files"`
	UnifiedDiff   string        `json:"unifiedDiff"`
	Generated     time.Time     `json:"generated"`
	TotalAdded    int           `json:"totalAdded"`
	TotalDeleted  int           `json:"totalDeleted"`
	TotalModified int           `json:"totalModified"`
}

// ApprovalRequest contains the parameters for approving changes.
type ApprovalRequest struct {
	SandboxID  uuid.UUID   `json:"sandboxId"`
	Mode       string      `json:"mode"` // "all", "files", "hunks"
	FileIDs    []uuid.UUID `json:"fileIds,omitempty"`
	HunkRanges []HunkRange `json:"hunkRanges,omitempty"`
	Actor      string      `json:"actor,omitempty"`
	CommitMsg  string      `json:"commitMessage,omitempty"`
}

// HunkRange specifies a range of lines to approve within a file.
type HunkRange struct {
	FileID    uuid.UUID `json:"fileId"`
	StartLine int       `json:"startLine"`
	EndLine   int       `json:"endLine"`
}

// ApprovalResult contains the outcome of an approval operation.
type ApprovalResult struct {
	Success    bool      `json:"success"`
	Applied    int       `json:"applied"`
	Failed     int       `json:"failed"`
	CommitHash string    `json:"commitHash,omitempty"`
	ErrorMsg   string    `json:"error,omitempty"`
	AppliedAt  time.Time `json:"appliedAt"`
}

// PathConflict represents a scope path conflict between sandboxes.
type PathConflict struct {
	ExistingID    string
	ExistingScope string
	NewScope      string
	ConflictType  ConflictType
}

// ConflictType identifies how two sandbox scope paths overlap.
// This is critical for the mutual exclusion rule: sandboxes cannot have
// overlapping scopes because changes in one could affect the other.
type ConflictType string

const (
	// ConflictTypeExact means the new and existing scopes are identical paths.
	// Example: new="/project/src" and existing="/project/src"
	ConflictTypeExact ConflictType = "exact"

	// ConflictTypeNewContainsExisting means the new scope is a parent of the existing scope.
	// If we allow this, the new sandbox could modify files that the existing sandbox
	// is also working on.
	// Example: new="/project" contains existing="/project/src"
	ConflictTypeNewContainsExisting ConflictType = "new_contains_existing"

	// ConflictTypeExistingContainsNew means the existing scope is a parent of the new scope.
	// The existing sandbox could modify files that the new sandbox wants to work on.
	// Example: existing="/project" contains new="/project/src"
	ConflictTypeExistingContainsNew ConflictType = "existing_contains_new"
)

// SandboxStats contains aggregate statistics for all sandboxes.
// Used for dashboard metrics and monitoring.
type SandboxStats struct {
	TotalCount     int64   `json:"totalCount"`
	ActiveCount    int64   `json:"activeCount"`
	StoppedCount   int64   `json:"stoppedCount"`
	ErrorCount     int64   `json:"errorCount"`
	ApprovedCount  int64   `json:"approvedCount"`
	RejectedCount  int64   `json:"rejectedCount"`
	DeletedCount   int64   `json:"deletedCount"`
	TotalSizeBytes int64   `json:"totalSizeBytes"`
	AvgSizeBytes   float64 `json:"avgSizeBytes"`
}

// CheckPathOverlap checks if an existing sandbox scope and a proposed new scope overlap.
// Returns the conflict type if there's an overlap, or empty string if no conflict.
//
// Parameters:
//   - existingScope: the scope path of an existing active sandbox
//   - newScope: the scope path being requested for a new sandbox
//
// The result indicates who "contains" whom:
//   - ConflictTypeExact: same path
//   - ConflictTypeExistingContainsNew: existing is parent of new
//   - ConflictTypeNewContainsExisting: new is parent of existing
func CheckPathOverlap(existingScope, newScope string) ConflictType {
	// Normalize paths to ensure consistent comparison
	existing := filepath.Clean(existingScope)
	proposed := filepath.Clean(newScope)

	// Exact match - same scope
	if existing == proposed {
		return ConflictTypeExact
	}

	// Check if existing scope contains (is parent of) the new scope
	// Example: existing="/project" contains new="/project/src"
	if strings.HasPrefix(proposed, existing+string(filepath.Separator)) {
		return ConflictTypeExistingContainsNew
	}

	// Check if new scope contains (is parent of) the existing scope
	// Example: new="/project" contains existing="/project/src"
	if strings.HasPrefix(existing, proposed+string(filepath.Separator)) {
		return ConflictTypeNewContainsExisting
	}

	return "" // No overlap - paths are independent
}
