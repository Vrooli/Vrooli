// Package sandbox provides the sandbox provider interface for isolation.
//
// This package defines the SEAM for sandbox operations. The default
// implementation integrates with workspace-sandbox, but the interface
// allows for alternative isolation mechanisms or mocking in tests.
package sandbox

import (
	"context"
	"time"

	"agent-manager/internal/adapters/runner"
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

	// ApplyAtRunEnd is the agent-manager run-end apply path. It carries
	// run-context metadata (run id, conversation id, cost, run outcome) onto
	// the workspace-sandbox apply pipeline so provenance is recorded against
	// the originating run. The workspace-sandbox endpoint validates that
	// Source == SourceAgentManagerAutoApply; other sources are rejected.
	//
	// Apply behaviour is identical regardless of RunOutcome — outcome is
	// metadata only. Out-of-acceptance files are retained as
	// state=pending-review on the resulting provenance record.
	//
	// See scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md.
	ApplyAtRunEnd(ctx context.Context, req ApplyAtRunEndRequest) (*ApplyAtRunEndResult, error)

	// TurnCheckpoint applies accepted turn changes and parks the sandbox in a
	// resumable non-terminal state.
	TurnCheckpoint(ctx context.Context, req TurnCheckpointRequest) (*TurnCheckpointResult, error)

	// Stop suspends a sandbox (keeps data but releases mount).
	Stop(ctx context.Context, id uuid.UUID) error

	// Start resumes a stopped sandbox.
	Start(ctx context.Context, id uuid.UUID) error

	// Resume remounts a checkpointed sandbox for the next turn.
	Resume(ctx context.Context, id uuid.UUID) (*Sandbox, error)

	// IsAvailable checks if the sandbox provider is operational.
	IsAvailable(ctx context.Context) (bool, string)

	// ValidatePath checks whether a path exists, is a directory, and is within
	// the given project root. Used for early input validation in the UI.
	ValidatePath(ctx context.Context, path string, projectRoot string) (*PathValidationResult, error)

	// ExecProcess runs a single command synchronously inside a sandbox via
	// workspace-sandbox /exec. Honors the sandbox's protected-mode
	// guardrails (git allowlist enforcement, network-mode restrictions).
	// Used by protected-mode runners to launch the agent process through
	// workspace-sandbox rather than directly on the host.
	//
	// See execute/protected-sandbox-agent-launch.
	ExecProcess(ctx context.Context, req ExecProcessRequest) (*ExecProcessResult, error)
}

// -----------------------------------------------------------------------------
// Request/Response Types
// -----------------------------------------------------------------------------

// CreateRequest contains parameters for sandbox creation.
type CreateRequest struct {
	// Name is an optional human-readable name for the sandbox.
	Name string

	// ScopePath is the relative path within the project to scope the sandbox.
	ScopePath string

	// NoLock disables reserved-path locking for investigative sandboxes.
	// When nil, the workspace-sandbox server default applies (WORKSPACE_SANDBOX_DEFAULT_NO_LOCK).
	NoLock *bool

	// ProjectRoot is the root directory of the project.
	ProjectRoot    string
	AuxiliaryRoots []string

	// Owner identifies who owns this sandbox.
	Owner string

	// OwnerType is the type of owner ("user", "agent", "system").
	OwnerType string

	// IdempotencyKey enables safe retries of create requests.
	IdempotencyKey string

	// Metadata contains additional sandbox metadata.
	Metadata map[string]string

	// Behavior controls lifecycle and acceptance configuration.
	Behavior *domain.SandboxConfig
}

// Sandbox represents an active or stopped sandbox.
type Sandbox struct {
	ID          uuid.UUID     `json:"id"`
	ScopePath   string        `json:"scopePath"`
	ProjectRoot string        `json:"projectRoot"`
	Status      SandboxStatus `json:"status"`

	// WorkDir is the host-side overlay merged dir. It is what host-routed
	// (tracking-mode) launches chdir into and the base host path the
	// SandboxLauncher translates onto the agent-visible WorkspacePath. Kept
	// distinct from WorkspacePath so host-side consumers (diff/apply, host
	// launches) never accidentally receive the in-namespace illusion path.
	WorkDir string `json:"workDir"`

	// WorkspacePath is the agent-visible workspace path reported by
	// workspace-sandbox: "/workspace" when exec containment binds under a
	// path illusion, else the host merged dir (identity). The single source
	// of truth for the in-namespace working directory — never inferred from
	// GOOS or driver id.
	WorkspacePath string `json:"workspacePath"`

	// PathIllusion is true when WorkspacePath differs from WorkDir because
	// the containment backend rewrites host paths onto an illusory mount.
	// false means path translation is identity.
	PathIllusion bool `json:"pathIllusion"`

	// Containment is the process-containment the sandbox actually enforces
	// (level, backend, enforcements). Consumed by the launcher selector to
	// surface degraded protected-mode runs. Nil when the server did not
	// report it (older workspace-sandbox).
	Containment *runner.Containment `json:"containment,omitempty"`

	CreatedAt        time.Time         `json:"createdAt"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	HomeOverlayState HomeOverlayState  `json:"homeOverlayState"`
}

// HomeOverlayState mirrors the workspace-sandbox enum
// (api/internal/types/types.go::HomeOverlayState). Single source of
// truth for "did this sandbox get a host-$HOME overlay?". Used by the
// SandboxLauncher to refuse $HOME/.local/... commands when the overlay
// is missing — without this, a missing overlay manifests as
// `env: …/claude: No such file or directory` at exec time.
//
// DOC: home-overlay seam — agent-manager mirror. See docs/internal/SEAMS.md.
type HomeOverlayState string

const (
	HomeOverlayPresent      HomeOverlayState = "present"
	HomeOverlayAbsent       HomeOverlayState = "absent"
	HomeOverlayNotRequested HomeOverlayState = "not_requested"
	HomeOverlayUnsupported  HomeOverlayState = "unsupported"
)

// IsHomeOverlayPresent reports whether the sandbox's home overlay is
// usable for $HOME-relative operations. Mirrors the workspace-sandbox
// policy.IsHomeOverlayPresent helper — kept synchronized via the
// parity test in home_overlay_policy_test.go.
//
// DOC: home-overlay seam — single predicate for state==Present.
func IsHomeOverlayPresent(state HomeOverlayState) bool {
	return state == HomeOverlayPresent
}

// SandboxStatus represents the sandbox lifecycle state.
type SandboxStatus string

const (
	SandboxStatusCreating     SandboxStatus = "creating"
	SandboxStatusActive       SandboxStatus = "active"
	SandboxStatusStopped      SandboxStatus = "stopped"
	SandboxStatusCheckpointed SandboxStatus = "checkpointed"
	SandboxStatusApproved     SandboxStatus = "approved"
	SandboxStatusRejected     SandboxStatus = "rejected"
	SandboxStatusDeleted      SandboxStatus = "deleted"
	SandboxStatusError        SandboxStatus = "error"
)

// DiffResult contains the generated diff for a sandbox.
type DiffResult struct {
	SandboxID   uuid.UUID    `json:"sandboxId"`
	Files       []FileChange `json:"files"`
	UnifiedDiff string       `json:"unifiedDiff"`
	Generated   time.Time    `json:"generated"`
	Stats       DiffStats    `json:"stats"`

	// ArchiveState distinguishes archive-sourced responses from live
	// responses. Empty (zero value) means the diff was served from the
	// live overlay; non-empty values only appear when the response came
	// from a durable archive captured at terminal-status transition.
	//   - empty          → live overlay
	//   - "complete"     → archived snapshot with content blobs
	//   - "not_captured" → archived row, no blobs (e.g. Error → Deleted)
	ArchiveState ArchiveState `json:"archiveState,omitempty"`
}

// ArchiveState mirrors workspace-sandbox's ArchiveState taxonomy.
type ArchiveState string

const (
	// ArchiveStateComplete: archived snapshot with content blobs durable
	// on disk. Live diffs leave ArchiveState empty rather than using
	// this value.
	ArchiveStateComplete ArchiveState = "complete"
	// ArchiveStateNotCaptured: terminal-state sandbox whose snapshot was
	// deliberately skipped (e.g. Error → Deleted). UI renders "no diff
	// captured" rather than treating an empty file list as a bug.
	ArchiveStateNotCaptured ArchiveState = "not_captured"
)

// FileChange represents a single file modification.
type FileChange struct {
	ID           uuid.UUID      `json:"id"`
	FilePath     string         `json:"filePath"`
	ChangeType   FileChangeType `json:"changeType"`
	FileSize     int64          `json:"fileSize"`
	LinesAdded   int            `json:"linesAdded,omitempty"`
	LinesRemoved int            `json:"linesRemoved,omitempty"`
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
	Success        bool      `json:"success"`
	Applied        int       `json:"applied"`
	Remaining      int       `json:"remaining"`
	IsPartial      bool      `json:"isPartial"`
	CommitHash     string    `json:"commitHash,omitempty"`
	AppliedAt      time.Time `json:"appliedAt"`
	ErrorMsg       string    `json:"errorMsg,omitempty"`
	TotalSizeBytes int64     `json:"appliedSizeBytes,omitempty"`
	DiffPath       string    `json:"diffPath,omitempty"`
}

// ApplyAtRunEndRequest carries run-context metadata onto the workspace-sandbox
// run-end apply call. The agent-manager seam mirrors the workspace-sandbox
// types.ApplyAtRunEndRequest shape but does not import workspace-sandbox
// directly (the adapter owns the wire translation).
type ApplyAtRunEndRequest struct {
	SandboxID uuid.UUID

	// RunID is the agent-manager run that produced these changes.
	RunID string

	// ConversationID is the agent-thread identifier (Decision D7). When
	// empty, the workspace-sandbox endpoint records nothing for the field.
	ConversationID string

	// Cost is the total USD cost of the run, recorded on provenance.
	Cost float64

	// RunOutcome is the contract-canonical outcome ∈ {success, failure,
	// cancelled, timeout}. Apply behaviour is identical regardless of
	// outcome (lossy by design); the value is metadata only.
	RunOutcome string

	// Actor mirrors ApprovalRequest.Actor for attribution policies. Defaults
	// to "applyAtRunEnd" when empty.
	Actor string

	// CommitMsg / CreateCommit / Force are forwarded to the apply pipeline.
	CommitMsg    string
	CreateCommit bool
	Force        bool
}

// ApplyAtRunEndResult mirrors the workspace-sandbox ApprovalResult fields the
// run executor cares about. IsPartial=true means out-of-acceptance files
// remain as state=pending-review on the provenance record; the sandbox
// persists for operator review.
type ApplyAtRunEndResult struct {
	Success        bool
	Applied        int
	Failed         int
	Remaining      int
	IsPartial      bool
	CommitHash     string
	AppliedAt      time.Time
	ErrorMsg       string
	TotalSizeBytes int64
	DiffPath       string
}

type TurnCheckpointRequest struct {
	SandboxID      uuid.UUID
	RunID          string
	ConversationID string
	TurnID         string
	TurnSequence   int
	Cost           float64
	RunOutcome     string
	Actor          string
	CommitMsg      string
	CreateCommit   bool
	Force          bool
}

type TurnCheckpointResult struct {
	SandboxID      uuid.UUID
	Status         SandboxStatus
	Success        bool
	Applied        int
	Failed         int
	Remaining      int
	IsPartial      bool
	CommitHash     string
	BaseCommitHash string
	CheckpointID   string
	AppliedAt      time.Time
	ErrorMsg       string
	TotalSizeBytes int64
	DiffPath       string
}

// PathValidationResult contains the result of a path validation check.
type PathValidationResult struct {
	Path              string `json:"path"`
	ProjectRoot       string `json:"projectRoot,omitempty"`
	Valid             bool   `json:"valid"`
	Exists            bool   `json:"exists,omitempty"`
	IsDirectory       bool   `json:"isDirectory,omitempty"`
	WithinProjectRoot bool   `json:"withinProjectRoot,omitempty"`
	Error             string `json:"error,omitempty"`
}

// ExecProcessRequest contains parameters for ExecProcess.
//
// Protected-mode runners (claude_code, codex, opencode) populate this when
// SandboxConfig.Mode == protected so the agent process runs through
// workspace-sandbox containment rather than directly on the host. The
// workspace-sandbox handler enforces:
//   - git verb allowlist (Behavior.Protected.GitAllowlist)
//   - network mode (none / localhost / full) via bwrap
//   - resource limits (memory, CPU, processes, open files, timeout)
type ExecProcessRequest struct {
	SandboxID      uuid.UUID
	Command        string
	Args           []string
	Env            map[string]string
	WorkingDir     string
	WritableMounts []WritableMount
	NetworkMode    string // "none" | "localhost" | "full"; "" → workspace-sandbox default

	MemoryLimitMB int
	CPUTimeSec    int
	TimeoutSec    int
	MaxProcesses  int
	MaxOpenFiles  int
}

// WritableMount declares a directory that the sandbox may bind read-write.
type WritableMount struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
}

// ExecProcessResult mirrors the workspace-sandbox /exec response.
type ExecProcessResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	PID      int
	TimedOut bool

	// Blocked reports a structured guardrail denial (e.g. git verb blocked).
	// When non-nil, the workspace-sandbox returned 403 with a typed body.
	Blocked *ExecBlocked
}

// ExecBlocked describes a structured workspace-sandbox guardrail denial.
type ExecBlocked struct {
	Error   string // e.g. "git_verb_blocked"
	Verb    string // e.g. "commit"
	Message string
}
