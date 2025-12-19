// Package sandbox provides the sandbox provider interface for isolation.
//
// This package defines the SEAM for sandbox operations. The default
// implementation integrates with workspace-sandbox, but the interface
// allows for alternative isolation mechanisms or mocking in tests.
package sandbox

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// SandboxProvider Interface - The primary seam for sandbox operations
// -----------------------------------------------------------------------------

// Provider is the interface for sandbox creation and management.
// This abstracts the workspace-sandbox integration, enabling:
// - Alternative sandbox implementations
// - Testing without actual sandbox creation
// - Future support for different isolation mechanisms
type Provider interface {
	// Create creates a new sandbox for the given scope.
	Create(ctx context.Context, req CreateRequest) (*Sandbox, error)

	// Get retrieves a sandbox by ID.
	Get(ctx context.Context, id uuid.UUID) (*Sandbox, error)

	// Delete removes a sandbox and its resources.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetWorkspacePath returns the path where agents should execute.
	GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error)

	// GetDiff generates a diff of changes made in the sandbox.
	GetDiff(ctx context.Context, id uuid.UUID) (*DiffResult, error)

	// Approve applies sandbox changes to the canonical repository.
	Approve(ctx context.Context, req ApproveRequest) (*ApproveResult, error)

	// Reject marks sandbox changes as rejected without applying.
	Reject(ctx context.Context, id uuid.UUID, actor string) error

	// PartialApprove approves only selected files from the sandbox.
	PartialApprove(ctx context.Context, req PartialApproveRequest) (*ApproveResult, error)

	// Stop suspends a sandbox (keeps data but releases mount).
	Stop(ctx context.Context, id uuid.UUID) error

	// Start resumes a stopped sandbox.
	Start(ctx context.Context, id uuid.UUID) error

	// IsAvailable checks if the sandbox provider is operational.
	IsAvailable(ctx context.Context) (bool, string)
}

// -----------------------------------------------------------------------------
// Request/Response Types
// -----------------------------------------------------------------------------

// CreateRequest contains parameters for sandbox creation.
type CreateRequest struct {
	// ScopePath is the relative path within the project to scope the sandbox.
	ScopePath string

	// ProjectRoot is the root directory of the project.
	ProjectRoot string

	// Owner identifies who owns this sandbox.
	Owner string

	// OwnerType is the type of owner ("user", "agent", "system").
	OwnerType string

	// IdempotencyKey enables safe retries of create requests.
	IdempotencyKey string

	// Metadata contains additional sandbox metadata.
	Metadata map[string]string
}

// Sandbox represents an active or stopped sandbox.
type Sandbox struct {
	ID          uuid.UUID         `json:"id"`
	ScopePath   string            `json:"scopePath"`
	ProjectRoot string            `json:"projectRoot"`
	Status      SandboxStatus     `json:"status"`
	WorkDir     string            `json:"workDir"`
	CreatedAt   time.Time         `json:"createdAt"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SandboxStatus represents the sandbox lifecycle state.
type SandboxStatus string

const (
	SandboxStatusCreating SandboxStatus = "creating"
	SandboxStatusActive   SandboxStatus = "active"
	SandboxStatusStopped  SandboxStatus = "stopped"
	SandboxStatusApproved SandboxStatus = "approved"
	SandboxStatusRejected SandboxStatus = "rejected"
	SandboxStatusDeleted  SandboxStatus = "deleted"
	SandboxStatusError    SandboxStatus = "error"
)

// DiffResult contains the generated diff for a sandbox.
type DiffResult struct {
	SandboxID   uuid.UUID     `json:"sandboxId"`
	Files       []FileChange  `json:"files"`
	UnifiedDiff string        `json:"unifiedDiff"`
	Generated   time.Time     `json:"generated"`
	Stats       DiffStats     `json:"stats"`
}

// FileChange represents a single file modification.
type FileChange struct {
	ID         uuid.UUID      `json:"id"`
	FilePath   string         `json:"filePath"`
	ChangeType FileChangeType `json:"changeType"`
	FileSize   int64          `json:"fileSize"`
	LinesAdded int            `json:"linesAdded,omitempty"`
	LinesRemoved int          `json:"linesRemoved,omitempty"`
}

// FileChangeType indicates how a file was modified.
type FileChangeType string

const (
	FileChangeAdded    FileChangeType = "added"
	FileChangeModified FileChangeType = "modified"
	FileChangeDeleted  FileChangeType = "deleted"
)

// DiffStats contains summary statistics for a diff.
type DiffStats struct {
	FilesChanged  int   `json:"filesChanged"`
	FilesAdded    int   `json:"filesAdded"`
	FilesModified int   `json:"filesModified"`
	FilesDeleted  int   `json:"filesDeleted"`
	TotalLines    int   `json:"totalLines"`
	LinesAdded    int   `json:"linesAdded"`
	LinesRemoved  int   `json:"linesRemoved"`
	TotalBytes    int64 `json:"totalBytes"`
}

// ApproveRequest contains parameters for approving sandbox changes.
type ApproveRequest struct {
	SandboxID uuid.UUID `json:"sandboxId"`
	Actor     string    `json:"actor"`
	CommitMsg string    `json:"commitMsg,omitempty"`
	Force     bool      `json:"force,omitempty"` // Force despite conflicts
}

// PartialApproveRequest approves only selected files.
type PartialApproveRequest struct {
	SandboxID uuid.UUID   `json:"sandboxId"`
	FileIDs   []uuid.UUID `json:"fileIds"`
	Actor     string      `json:"actor"`
	CommitMsg string      `json:"commitMsg,omitempty"`
}

// ApproveResult contains the outcome of an approval.
type ApproveResult struct {
	Success    bool      `json:"success"`
	Applied    int       `json:"applied"`
	Remaining  int       `json:"remaining"`
	IsPartial  bool      `json:"isPartial"`
	CommitHash string    `json:"commitHash,omitempty"`
	AppliedAt  time.Time `json:"appliedAt"`
	ErrorMsg   string    `json:"errorMsg,omitempty"`
}

// -----------------------------------------------------------------------------
// Scope Lock Interface
// -----------------------------------------------------------------------------

// LockManager handles scope-based locking for concurrent runs.
type LockManager interface {
	// Acquire attempts to acquire a lock for the given scope.
	// Returns an error if the scope overlaps with an existing lock.
	Acquire(ctx context.Context, req LockRequest) (*domain.ScopeLock, error)

	// Release releases a previously acquired lock.
	Release(ctx context.Context, lockID uuid.UUID) error

	// Check verifies if a scope can be locked without acquiring.
	Check(ctx context.Context, scopePath, projectRoot string) (bool, []domain.ScopeConflict, error)

	// Refresh extends the expiration time of a lock.
	Refresh(ctx context.Context, lockID uuid.UUID, extension time.Duration) error
}

// LockRequest contains parameters for acquiring a scope lock.
type LockRequest struct {
	RunID       uuid.UUID
	ScopePath   string
	ProjectRoot string
	TTL         time.Duration
}
