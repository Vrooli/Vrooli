// Package types provides shared types for workspace sandboxes.
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

// ConflictType identifies the type of path conflict.
type ConflictType string

const (
	ConflictTypeExact              ConflictType = "exact"
	ConflictTypeNewIsAncestor      ConflictType = "new_is_ancestor"
	ConflictTypeExistingIsAncestor ConflictType = "existing_is_ancestor"
)

// CheckPathOverlap checks if two paths have an ancestor/descendant relationship.
// Returns the conflict type if there's an overlap, or empty string if no conflict.
func CheckPathOverlap(path1, path2 string) ConflictType {
	// Normalize paths
	p1 := filepath.Clean(path1)
	p2 := filepath.Clean(path2)

	// Exact match
	if p1 == p2 {
		return ConflictTypeExact
	}

	// Check if p1 is ancestor of p2
	if strings.HasPrefix(p2, p1+string(filepath.Separator)) {
		return ConflictTypeNewIsAncestor
	}

	// Check if p2 is ancestor of p1
	if strings.HasPrefix(p1, p2+string(filepath.Separator)) {
		return ConflictTypeExistingIsAncestor
	}

	return ""
}
