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
//
//   - LowerDir: read-only view of the canonical repo
//
//   - UpperDir: writable layer capturing changes
//
//   - MergedDir: combined view where agents work
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

	// IdempotencyKey is a client-provided key for request deduplication.
	// If set, subsequent create requests with the same key return this sandbox.
	IdempotencyKey string `json:"idempotencyKey,omitempty" db:"idempotency_key"`

	// UpdatedAt tracks the last modification time for optimistic concurrency control.
	// Operations that update the sandbox should check this value to detect
	// concurrent modifications.
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`

	// Version is incremented on each update for optimistic locking.
	// Callers can include this in update requests to detect concurrent modifications.
	Version int64 `json:"version" db:"version"`

	// BaseCommitHash stores the canonical repo's commit hash at sandbox creation time.
	// Used for conflict detection (OT-P2-002): if the canonical repo has diverged,
	// patch application may fail or produce unexpected results.
	BaseCommitHash string `json:"baseCommitHash,omitempty" db:"base_commit_hash"`
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
//
// # Idempotency
//
// The IdempotencyKey field enables safe retries. If a request with the same
// IdempotencyKey was already processed, the same result is returned without
// creating a duplicate sandbox. Keys should be unique per logical operation
// (e.g., a UUID generated by the caller).
//
// If no IdempotencyKey is provided, each request creates a new sandbox.
type CreateRequest struct {
	ScopePath   string                 `json:"scopePath"`
	ProjectRoot string                 `json:"projectRoot,omitempty"`
	Owner       string                 `json:"owner,omitempty"`
	OwnerType   OwnerType              `json:"ownerType,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	// IdempotencyKey is an optional client-provided key for request deduplication.
	// If provided and a sandbox was already created with this key, that sandbox
	// is returned instead of creating a new one. This enables safe retries.
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
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

	// Force bypasses conflict detection and applies changes even if the
	// canonical repo has changed since sandbox creation. Use with caution.
	// [OT-P2-002] Conflict Detection
	Force bool `json:"force,omitempty"`

	// CreateCommit controls whether to create a git commit after applying changes.
	// Default is false - changes are applied to the working tree only.
	// When true and CommitMsg is provided, a commit is created.
	CreateCommit bool `json:"createCommit,omitempty"`
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
	Remaining  int       `json:"remaining"` // [OT-P1-002] Number of unapproved changes still in sandbox
	IsPartial  bool      `json:"isPartial"` // [OT-P1-002] True if sandbox preserved for follow-up approvals
	CommitHash string    `json:"commitHash,omitempty"`
	ErrorMsg   string    `json:"error,omitempty"`
	AppliedAt  time.Time `json:"appliedAt"`

	// ConflictInfo contains information about detected conflicts if any.
	// [OT-P2-002] Conflict Detection
	ConflictInfo *ConflictInfo `json:"conflictInfo,omitempty"`
}

// ConflictInfo contains information about detected repo conflicts.
// [OT-P2-002] Conflict Detection
type ConflictInfo struct {
	// HasConflict is true if the canonical repo has changed since sandbox creation
	HasConflict bool `json:"hasConflict"`

	// BaseCommitHash is the commit hash at sandbox creation
	BaseCommitHash string `json:"baseCommitHash,omitempty"`

	// CurrentHash is the current commit hash in the canonical repo
	CurrentHash string `json:"currentHash,omitempty"`

	// ConflictingFiles lists files modified in both sandbox and canonical repo
	ConflictingFiles []string `json:"conflictingFiles,omitempty"`

	// RepoChangedFiles lists all files changed in repo since sandbox creation
	RepoChangedFiles []string `json:"repoChangedFiles,omitempty"`
}

// DiscardRequest contains the parameters for discarding specific changes.
// This allows removing files from the sandbox without applying them.
type DiscardRequest struct {
	SandboxID uuid.UUID   `json:"sandboxId"`
	FileIDs   []uuid.UUID `json:"fileIds"`             // Files to discard
	FilePaths []string    `json:"filePaths,omitempty"` // Alternative: paths instead of IDs
	Actor     string      `json:"actor,omitempty"`
}

// DiscardResult contains the outcome of a discard operation.
type DiscardResult struct {
	Success   bool     `json:"success"`
	Discarded int      `json:"discarded"` // Number of files discarded
	Remaining int      `json:"remaining"` // Number of changes still pending
	ErrorMsg  string   `json:"error,omitempty"`
	Files     []string `json:"files,omitempty"` // Paths of discarded files
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

// --- GC (Garbage Collection) Types [OT-P1-003] ---

// GCPolicy specifies which sandboxes should be garbage collected.
// Multiple policies can be combined - sandboxes matching ANY policy are collected.
type GCPolicy struct {
	// MaxAge is the maximum age of sandboxes. Sandboxes older than this are collected.
	// Zero means no age limit.
	MaxAge time.Duration `json:"maxAge,omitempty"`

	// IdleTimeout is how long a sandbox can be unused before collection.
	// Based on LastUsedAt timestamp.
	// Zero means no idle timeout.
	IdleTimeout time.Duration `json:"idleTimeout,omitempty"`

	// MaxTotalSizeBytes is the maximum total size of all sandboxes.
	// When exceeded, oldest sandboxes are collected until under limit.
	// Zero means no size limit.
	MaxTotalSizeBytes int64 `json:"maxTotalSizeBytes,omitempty"`

	// IncludeTerminal specifies whether to collect approved/rejected sandboxes.
	// If true, collects sandboxes in terminal states (approved, rejected) after
	// TerminalDelay has passed.
	IncludeTerminal bool `json:"includeTerminal,omitempty"`

	// TerminalDelay is how long to wait before collecting terminal sandboxes.
	// Only applies if IncludeTerminal is true.
	// Default: 1 hour
	TerminalDelay time.Duration `json:"terminalDelay,omitempty"`

	// Statuses limits collection to sandboxes in these states.
	// Empty means: stopped, error (never touches active sandboxes).
	Statuses []Status `json:"statuses,omitempty"`
}

// DefaultGCPolicy returns a sensible default GC policy.
func DefaultGCPolicy() GCPolicy {
	return GCPolicy{
		MaxAge:          24 * time.Hour,
		IdleTimeout:     4 * time.Hour,
		IncludeTerminal: true,
		TerminalDelay:   1 * time.Hour,
		// Only collect non-active sandboxes by default
		Statuses: []Status{StatusStopped, StatusError, StatusApproved, StatusRejected},
	}
}

// GCRequest contains parameters for a garbage collection run.
type GCRequest struct {
	// Policy specifies the GC criteria. If nil, uses DefaultGCPolicy.
	Policy *GCPolicy `json:"policy,omitempty"`

	// DryRun if true, reports what would be collected without actually deleting.
	DryRun bool `json:"dryRun,omitempty"`

	// Limit is the maximum number of sandboxes to collect in this run.
	// Zero means no limit.
	Limit int `json:"limit,omitempty"`

	// Actor identifies who/what initiated the GC run.
	Actor string `json:"actor,omitempty"`
}

// GCResult contains the outcome of a garbage collection run.
type GCResult struct {
	// Collected is the list of sandboxes that were (or would be) collected.
	Collected []*GCCollectedSandbox `json:"collected"`

	// TotalCollected is the count of sandboxes collected.
	TotalCollected int `json:"totalCollected"`

	// TotalBytesReclaimed is the total size of collected sandboxes.
	TotalBytesReclaimed int64 `json:"totalBytesReclaimed"`

	// Errors contains any errors encountered during collection.
	Errors []GCError `json:"errors,omitempty"`

	// DryRun indicates if this was a dry run (no actual deletion).
	DryRun bool `json:"dryRun"`

	// StartedAt is when the GC run started.
	StartedAt time.Time `json:"startedAt"`

	// CompletedAt is when the GC run finished.
	CompletedAt time.Time `json:"completedAt"`

	// Reason describes why each sandbox was collected.
	Reasons map[string][]string `json:"reasons,omitempty"`
}

// GCCollectedSandbox contains info about a collected sandbox.
type GCCollectedSandbox struct {
	ID        uuid.UUID `json:"id"`
	ScopePath string    `json:"scopePath"`
	Status    Status    `json:"status"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
	Reason    string    `json:"reason"`
}

// GCError represents an error during garbage collection.
type GCError struct {
	SandboxID uuid.UUID `json:"sandboxId"`
	Error     string    `json:"error"`
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

// --- Retry/Rebase Workflow Types (OT-P2-003) ---

// RebaseRequest contains parameters for rebasing a sandbox against the current repo state.
type RebaseRequest struct {
	SandboxID uuid.UUID `json:"sandboxId"`

	// Strategy determines how to handle conflicts during rebase.
	// "regenerate": Regenerate diff only without merging (just update BaseCommitHash)
	// This is the safest and most common option.
	Strategy RebaseStrategy `json:"strategy"`

	// Actor identifies who/what initiated the rebase.
	Actor string `json:"actor,omitempty"`
}

// RebaseStrategy determines how conflicts are handled during rebase.
type RebaseStrategy string

const (
	// RebaseStrategyRegenerate only updates BaseCommitHash without merging.
	// The sandbox changes remain intact, but the diff will be regenerated
	// against the new canonical repo state for accurate conflict detection.
	RebaseStrategyRegenerate RebaseStrategy = "regenerate"
)

// RebaseResult contains the outcome of a rebase operation.
type RebaseResult struct {
	Success bool `json:"success"`

	// PreviousBaseHash is the commit hash before rebase.
	PreviousBaseHash string `json:"previousBaseHash,omitempty"`

	// NewBaseHash is the commit hash after rebase.
	NewBaseHash string `json:"newBaseHash,omitempty"`

	// ConflictingFiles lists files with potential conflicts (changed in both sandbox and repo).
	ConflictingFiles []string `json:"conflictingFiles,omitempty"`

	// RepoChangedFiles lists files changed in repo since original sandbox creation.
	RepoChangedFiles []string `json:"repoChangedFiles,omitempty"`

	// Strategy used for the rebase.
	Strategy RebaseStrategy `json:"strategy"`

	// ErrorMsg contains error details if rebase failed.
	ErrorMsg string `json:"error,omitempty"`

	// RebasedAt is when the rebase was performed.
	RebasedAt time.Time `json:"rebasedAt"`
}

// ConflictCheckRequest contains parameters for checking conflicts.
type ConflictCheckRequest struct {
	SandboxID uuid.UUID `json:"sandboxId"`
}

// PathValidationResult contains the result of validating a filesystem path.
// Used by the UI to check paths before creating sandboxes.
type PathValidationResult struct {
	// Path is the validated path (echoed back).
	Path string `json:"path"`

	// ProjectRoot is the project root used for validation (echoed back).
	ProjectRoot string `json:"projectRoot,omitempty"`

	// Valid is true if the path passes all validation checks.
	Valid bool `json:"valid"`

	// Exists is true if the path exists on the filesystem.
	Exists bool `json:"exists,omitempty"`

	// IsDirectory is true if the path is a directory.
	IsDirectory bool `json:"isDirectory,omitempty"`

	// WithinProjectRoot is true if the path is within the project root.
	WithinProjectRoot bool `json:"withinProjectRoot,omitempty"`

	// Error contains a human-readable error message if validation failed.
	Error string `json:"error,omitempty"`
}

// ConflictCheckResponse contains the result of a conflict check.
type ConflictCheckResponse struct {
	// HasConflict is true if the canonical repo has changed since sandbox creation.
	HasConflict bool `json:"hasConflict"`

	// BaseCommitHash is the commit hash at sandbox creation.
	BaseCommitHash string `json:"baseCommitHash,omitempty"`

	// CurrentHash is the current commit hash in the canonical repo.
	CurrentHash string `json:"currentHash,omitempty"`

	// RepoChangedFiles lists all files changed in repo since sandbox creation.
	RepoChangedFiles []string `json:"repoChangedFiles,omitempty"`

	// SandboxChangedFiles lists all files changed in the sandbox.
	SandboxChangedFiles []string `json:"sandboxChangedFiles,omitempty"`

	// ConflictingFiles lists files modified in both sandbox and canonical repo.
	ConflictingFiles []string `json:"conflictingFiles,omitempty"`

	// CheckedAt is when the check was performed.
	CheckedAt time.Time `json:"checkedAt"`
}

// --- Provenance Tracking Types ---

// AppliedChange represents a file change that was applied from a sandbox.
// Used for provenance tracking - knowing which sandbox modified which files.
type AppliedChange struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	SandboxID        uuid.UUID  `json:"sandboxId" db:"sandbox_id"`
	SandboxOwner     string     `json:"sandboxOwner" db:"sandbox_owner"`
	SandboxOwnerType string     `json:"sandboxOwnerType" db:"sandbox_owner_type"`
	FilePath         string     `json:"filePath" db:"file_path"`
	ProjectRoot      string     `json:"projectRoot" db:"project_root"`
	ChangeType       string     `json:"changeType" db:"change_type"`
	FileSize         int64      `json:"fileSize" db:"file_size"`
	AppliedAt        time.Time  `json:"appliedAt" db:"applied_at"`
	CommittedAt      *time.Time `json:"committedAt,omitempty" db:"committed_at"`
	CommitHash       string     `json:"commitHash,omitempty" db:"commit_hash"`
	CommitMessage    string     `json:"commitMessage,omitempty" db:"commit_message"`
}

// PendingChangesSummary summarizes pending changes from a single sandbox.
type PendingChangesSummary struct {
	SandboxID     uuid.UUID `json:"sandboxId"`
	SandboxOwner  string    `json:"sandboxOwner"`
	FileCount     int       `json:"fileCount"`
	LatestApplied time.Time `json:"latestApplied"`
}

// PendingChangesResult contains the result of querying pending changes.
type PendingChangesResult struct {
	Summaries  []PendingChangesSummary `json:"summaries"`
	TotalFiles int                     `json:"totalFiles"`
}

// CommitPendingRequest contains parameters for committing pending changes.
type CommitPendingRequest struct {
	// ProjectRoot filters to changes in a specific project.
	ProjectRoot string `json:"projectRoot,omitempty"`

	// SandboxIDs filters to changes from specific sandboxes.
	// If empty, all pending changes (optionally filtered by ProjectRoot) are committed.
	SandboxIDs []uuid.UUID `json:"sandboxIds,omitempty"`

	// CommitMessage is the message for the git commit.
	CommitMessage string `json:"commitMessage,omitempty"`

	// Actor identifies who initiated the commit.
	Actor string `json:"actor,omitempty"`
}

// CommitPendingResult contains the outcome of committing pending changes.
type CommitPendingResult struct {
	Success        bool   `json:"success"`
	FilesCommitted int    `json:"filesCommitted"`
	CommitHash     string `json:"commitHash,omitempty"`
	ErrorMsg       string `json:"error,omitempty"`
}

// --- Commit Preview Types ---

// CommitPreviewFile represents a single file in the commit preview.
type CommitPreviewFile struct {
	FilePath     string    `json:"filePath"`
	RelativePath string    `json:"relativePath"`
	ChangeType   string    `json:"changeType"`
	SandboxID    uuid.UUID `json:"sandboxId"`
	SandboxOwner string    `json:"sandboxOwner"`
	AppliedAt    time.Time `json:"appliedAt"`
	// Status indicates the file's current state relative to git
	// "pending" = still uncommitted, "already_committed" = committed externally
	Status string `json:"status"`
}

// CommitPreviewRequest contains parameters for the commit preview endpoint.
type CommitPreviewRequest struct {
	ProjectRoot string      `json:"projectRoot,omitempty"`
	SandboxIDs  []uuid.UUID `json:"sandboxIds,omitempty"`
}

// CommitPreviewResult contains the preview of what would be committed.
type CommitPreviewResult struct {
	// Files contains all pending files with their status
	Files []CommitPreviewFile `json:"files"`

	// CommittableFiles is the count of files that can actually be committed
	// (still uncommitted in git)
	CommittableFiles int `json:"committableFiles"`

	// AlreadyCommittedFiles is the count of files that were committed externally
	// These will be marked as reconciled but not included in the new commit
	AlreadyCommittedFiles int `json:"alreadyCommittedFiles"`

	// SuggestedMessage is an auto-generated commit message
	SuggestedMessage string `json:"suggestedMessage"`

	// GroupedBySandbox provides a summary grouped by sandbox owner
	GroupedBySandbox []CommitPreviewSandboxGroup `json:"groupedBySandbox"`
}

// CommitPreviewSandboxGroup summarizes changes from a single sandbox.
type CommitPreviewSandboxGroup struct {
	SandboxID    uuid.UUID `json:"sandboxId"`
	SandboxOwner string    `json:"sandboxOwner"`
	FileCount    int       `json:"fileCount"`
	Added        int       `json:"added"`
	Modified     int       `json:"modified"`
	Deleted      int       `json:"deleted"`
}
