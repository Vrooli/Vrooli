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
//     Key states: creating → active → checkpointed/stopped → approved/rejected → deleted
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
// Two sandboxes cannot have overlapping reserved directories. This prevents:
//   - Two agents working on the same subtree at once (conflicts/collisions)
//   - Approval ambiguity when multiple sandboxes propose changes to the same area
//
// NoLock disables this mutual exclusion behavior when explicitly requested.
//
// Note: ScopePath controls what is mounted copy-on-write. ReservedPath(s) controls:
//
//	(1) mutual exclusion/locking, and (2) the default approval allowlist.
//
// The ConflictType enum describes the relationship when reserved paths overlap.
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

// ViewMode specifies how to display file content in the diff viewer.
type ViewMode string

const (
	ViewModeDiff     ViewMode = "diff"      // Show only changed lines (hunks)
	ViewModeFullDiff ViewMode = "full_diff" // Show full file with changes highlighted
	ViewModeSource   ViewMode = "source"    // Show file content only (no diff markers)
)

// LineChange indicates what type of change occurred on a line.
type LineChange string

const (
	LineChangeNone    LineChange = ""        // Context line (unchanged)
	LineChangeAdded   LineChange = "added"   // Line was added
	LineChangeDeleted LineChange = "deleted" // Line was deleted
)

// AnnotatedLine represents a single line with its change status.
// Used in full_diff mode to show the complete file with inline change markers.
type AnnotatedLine struct {
	Number    int        `json:"number"`              // Current line number (0 for deleted lines)
	Content   string     `json:"content"`             // Line content
	Change    LineChange `json:"change,omitempty"`    // Type of change
	OldNumber int        `json:"oldNumber,omitempty"` // Original line number (for deleted lines)
}

// FileViewData holds per-file content for full_diff and source view modes.
type FileViewData struct {
	FullContent    string          `json:"fullContent,omitempty"`    // Complete file content
	AnnotatedLines []AnnotatedLine `json:"annotatedLines,omitempty"` // Lines with change annotations
}

// AcceptanceStatus describes how a file change maps to acceptance rules.
type AcceptanceStatus string

const (
	AcceptanceStatusAccepted      AcceptanceStatus = "accepted"
	AcceptanceStatusIgnored       AcceptanceStatus = "ignored"
	AcceptanceStatusDenied        AcceptanceStatus = "denied"
	AcceptanceStatusBinaryIgnored AcceptanceStatus = "binary_ignored"
)

// FileCriteria defines allow/deny matchers for file acceptance.
type FileCriteria struct {
	PathGlobs  []string `json:"pathGlobs,omitempty"`
	Extensions []string `json:"extensions,omitempty"`
}

// AcceptanceConfig controls which files are eligible for approval.
type AcceptanceConfig struct {
	Mode         string       `json:"mode,omitempty"` // "allowlist"
	Allow        FileCriteria `json:"allow,omitempty"`
	Deny         FileCriteria `json:"deny,omitempty"`
	IgnoreBinary bool         `json:"ignoreBinary,omitempty"`
}

// LifecycleEvent represents lifecycle triggers for sandbox cleanup.
type LifecycleEvent string

const (
	LifecycleEventRunCompleted LifecycleEvent = "run_completed"
	LifecycleEventRunFailed    LifecycleEvent = "run_failed"
	LifecycleEventRunCancelled LifecycleEvent = "run_cancelled"
	LifecycleEventApproved     LifecycleEvent = "approved"
	LifecycleEventRejected     LifecycleEvent = "rejected"
	LifecycleEventTerminal     LifecycleEvent = "terminal"
)

// LifecycleConfig controls sandbox stop/delete behavior.
type LifecycleConfig struct {
	StopOn      []LifecycleEvent `json:"stopOn,omitempty"`
	DeleteOn    []LifecycleEvent `json:"deleteOn,omitempty"`
	TTL         time.Duration    `json:"ttl,omitempty"`
	IdleTimeout time.Duration    `json:"idleTimeout,omitempty"`
}

// SandboxBehavior configures lifecycle and acceptance policies plus the
// auditability-contract apply levers (Phase 4 of
// agent-sandbox-audit-foundation).
type SandboxBehavior struct {
	Lifecycle  LifecycleConfig  `json:"lifecycle,omitempty"`
	Acceptance AcceptanceConfig `json:"acceptance,omitempty"`

	// ManualReview defers apply until an operator explicitly approves via
	// one of the three viewing surfaces (GCT, agent-manager,
	// workspace-sandbox). When true, the LifecycleReconciler enforces
	// LifecycleConfig.ManualReviewTTL: abandoned sandboxes auto-deny on
	// expiry with Source=SourceWorkspaceSandboxGC. Default false.
	ManualReview bool `json:"manualReview,omitempty"`

	// Protected configures protected-mode runtime guardrails. Empty struct
	// means no enforcement (tracking-mode behaviour). Per the
	// protected-agent-sandboxing initiative, the agent-manager side sets
	// these when SandboxConfig.Mode == protected.
	Protected ProtectedConfig `json:"protected,omitempty"`
}

// ProtectedConfig configures runtime guardrails for protected-mode sandboxes.
// See execute/protected-sandbox-git-and-network-guardrails.
type ProtectedConfig struct {
	// GitAllowlist names the only `git` verbs that may be invoked via /exec
	// when this is non-empty. Default for protected mode is the read-only
	// set: status, diff, log, show, rev-parse. Mutating verbs (commit,
	// branch, checkout, reset, rebase, merge, push, pull, clean) are
	// blocked with a structured 403 — agents must go through GCT for
	// side-effecting git operations (wrap-not-use principle).
	//
	// Empty slice means "no allowlist enforcement" (tracking mode). To
	// allow ALL git verbs, set the allowlist to ["*"] explicitly.
	GitAllowlist []string `json:"gitAllowlist,omitempty"`
}

// DefaultProtectedGitAllowlist returns the locked default allowlist for
// protected-mode sandboxes per the auditability contract. Intended for
// agent-manager to populate when constructing the apply-at-run-end payload
// for a protected-mode sandbox.
func DefaultProtectedGitAllowlist() []string {
	return []string{"status", "diff", "log", "show", "rev-parse"}
}

// ApprovalStatus represents the approval state of a change.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

// HomeOverlayState describes whether a sandbox has a working home
// overlay (host $HOME visible inside the namespace via per-sandbox CoW).
//
// DOC: home-overlay seam. See docs/internal/SEAMS.md. Tied to:
//   - driver.DriverCapabilities.HomeOverlay (does the driver provide it?)
//   - config.IsolationProfile.HomeOverlayRequirement (does this profile need it?)
//   - handlers.process exec (loud failure when required and Absent).
type HomeOverlayState string

const (
	// HomeOverlayPresent means the home overlay was mounted and verified.
	HomeOverlayPresent HomeOverlayState = "present"
	// HomeOverlayAbsent means the driver supports the overlay but the
	// mount failed (verify failure, mount-cmd error, layout invalid, etc.).
	// Sandbox is still usable for profiles that don't need the overlay.
	HomeOverlayAbsent HomeOverlayState = "absent"
	// HomeOverlayNotRequested means no host $HOME was configured (rare —
	// usually only in tests).
	HomeOverlayNotRequested HomeOverlayState = "not_requested"
	// HomeOverlayUnsupported means the driver doesn't provide a home
	// overlay at all (copy driver).
	HomeOverlayUnsupported HomeOverlayState = "unsupported"
)

// HomeOverlayRequirement is the profile-side declaration of how
// strongly the profile depends on the per-sandbox host-$HOME overlay.
// Three-valued so profiles can express "uses $HOME if present, falls
// back if not" rather than a binary required/off shape that would
// force every absent-overlay case into a refusal.
//
// DOC: home-overlay seam — profile-side requirement declaration.
// See docs/internal/SEAMS.md.
type HomeOverlayRequirement string

const (
	// HomeOverlayNotNeeded means the profile does not depend on the
	// host $HOME overlay. This is the default for new/unset profiles.
	HomeOverlayNotNeeded HomeOverlayRequirement = "not_needed"
	// HomeOverlayOptional means the profile uses $HOME-relative paths
	// when the overlay is Present but functions correctly without it.
	// Callers MUST treat absence as a soft fallback (HOME_OVERLAY_FALLBACK
	// audit code) rather than a refusal.
	HomeOverlayOptional HomeOverlayRequirement = "optional"
	// HomeOverlayRequired means the profile cannot function without the
	// host $HOME overlay. Handlers refuse exec with HTTP 409 when the
	// sandbox's HomeOverlayState is anything other than Present.
	HomeOverlayRequired HomeOverlayRequirement = "required"
)

// IsValid reports whether the requirement holds one of the three
// canonical values.
func (r HomeOverlayRequirement) IsValid() bool {
	switch r {
	case HomeOverlayNotNeeded, HomeOverlayOptional, HomeOverlayRequired:
		return true
	}
	return false
}

// Sandbox represents a workspace sandbox with all its metadata.
type Sandbox struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	Name           string     `json:"name,omitempty" db:"name"`
	ScopePath      string     `json:"scopePath" db:"scope_path"`
	ReservedPath   string     `json:"reservedPath" db:"reserved_path"`
	ReservedPaths  []string   `json:"reservedPaths,omitempty" db:"reserved_paths"`
	NoLock         bool       `json:"noLock,omitempty" db:"no_lock"`
	ProjectRoot    string     `json:"projectRoot" db:"project_root"`
	AuxiliaryRoots []string   `json:"auxiliaryRoots,omitempty" db:"auxiliary_roots"`
	Owner          string     `json:"owner,omitempty" db:"owner"`
	OwnerType      OwnerType  `json:"ownerType" db:"owner_type"`
	Status         Status     `json:"status" db:"status"`
	ErrorMsg       string     `json:"errorMessage,omitempty" db:"error_message"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
	LastUsedAt     time.Time  `json:"lastUsedAt" db:"last_used_at"`
	StoppedAt      *time.Time `json:"stoppedAt,omitempty" db:"stopped_at"`
	ApprovedAt     *time.Time `json:"approvedAt,omitempty" db:"approved_at"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`

	// Driver configuration. DriverID is the canonical driver identifier
	// (overlayfs-userns, overlayfs-root, fuse-overlayfs, copy). The DB
	// column is `driver_id`; an older `driver` column (with the legacy
	// `overlayfs` value) is migrated at startup by main.go.
	DriverID      string `json:"driverId" db:"driver_id"`
	DriverVersion string `json:"driverVersion" db:"driver_version"`

	// Mount paths
	LowerDir  string `json:"lowerDir,omitempty" db:"lower_dir"`
	UpperDir  string `json:"upperDir,omitempty" db:"upper_dir"`
	WorkDir   string `json:"workDir,omitempty" db:"work_dir"`
	MergedDir string `json:"mergedDir,omitempty" db:"merged_dir"`

	// Home overlay paths (transient — recreated on every Mount, not
	// persisted to the DB). The home overlay is a per-sandbox
	// overlay mount whose lower layer is the host $HOME and whose upper
	// layer lives under HomeOverlayBaseDir (outside $HOME — see
	// config.ResolveStoragePaths().Transient). bwrap binds HomeMergedDir at
	// /home/<user> inside the namespace so agent CLIs (claude, codex,
	// etc.) find their host configuration while writes go to the per-run
	// upper layer.
	HomeLowerDir  string `json:"-" db:"-"`
	HomeUpperDir  string `json:"-" db:"-"`
	HomeWorkDir   string `json:"-" db:"-"`
	HomeMergedDir string `json:"homeMergedDir,omitempty" db:"-"`

	// HomeOverlayState is the canonical answer to "did this sandbox get
	// a home overlay?" Persisted in sandboxes.home_overlay_state.
	// DOC: home-overlay seam. See docs/internal/SEAMS.md.
	//
	// Set during Mount: Present on success; Absent when the driver
	// supports overlay but the mount failed; NotRequested when no
	// $HOME was set; Unsupported on the copy driver. Handlers refuse
	// vrooli-aware exec when this is anything other than Present.
	HomeOverlayState HomeOverlayState `json:"homeOverlayState" db:"home_overlay_state"`

	// Size accounting
	SizeBytes int64 `json:"sizeBytes" db:"size_bytes"`
	FileCount int   `json:"fileCount" db:"file_count"`

	// Session tracking
	ActivePIDs   []int `json:"activePids" db:"active_pids"`
	SessionCount int   `json:"sessionCount" db:"session_count"`

	// Metadata
	Tags     []string               `json:"tags,omitempty" db:"tags"`
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`

	// Behavior captures lifecycle and acceptance configuration.
	Behavior SandboxBehavior `json:"behavior,omitempty" db:"behavior"`

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

	// --- Computed, never persisted (db:"-"). Populated by handlers from the
	// active driver's required containment plus the host containment probe
	// before the sandbox is serialized, so every sandbox-returning endpoint
	// carries the same negotiated workspace contract. See the single
	// derivation function driver.DeriveWorkspaceLayout. ---

	// WorkspacePath is the path the agent process sees for its workspace.
	// "/workspace" when exec containment bind-mounts the merged dir under a
	// path illusion (bwrap on Linux); otherwise the host MergedDir (identity
	// layout — copy driver, or any backend without path illusion). Consumers
	// MUST take the agent-visible path from here, never infer it from GOOS or
	// driver id.
	WorkspacePath string `json:"workspacePath" db:"-"`

	// PathIllusion is true when WorkspacePath differs from the host MergedDir
	// because the containment backend translates host paths onto an illusory
	// mount point. false means WorkspacePath == MergedDir and path
	// translation is identity.
	PathIllusion bool `json:"pathIllusion" db:"-"`

	// Containment reports the process-containment that is actually in effect
	// for execs against this sandbox (level, backend, enforcement list).
	Containment *SandboxContainment `json:"containment,omitempty" db:"-"`
}

// SandboxContainment describes the process-containment in effect: the
// required level, the backend that carries it out ("bwrap" on Linux,
// "none" when execution falls through to the direct path), and the
// platform-neutral enforcement guarantees that backend provides. Reported
// on sandbox responses (predicted from the driver) and stamped on
// exec/process responses (the backend that actually ran the launch).
type SandboxContainment struct {
	Level        string   `json:"level"`
	Backend      string   `json:"backend"`
	Enforcements []string `json:"enforcements"`
	Mode         string   `json:"mode"`
	Gaps         []string `json:"gaps,omitempty"`
}

// HostWorkspacePath returns the host-side path where sandbox operations
// occur (the overlay merged dir), or "" when the sandbox is not active.
// Distinct from the wire field WorkspacePath, which is the agent-visible
// path inside the containment namespace.
func (s *Sandbox) HostWorkspacePath() string {
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

	// Acceptance describes how this change maps to acceptance rules.
	Acceptance *AcceptanceInfo `json:"acceptance,omitempty" db:"-"`
}

// AcceptanceInfo captures acceptance evaluation for a file change.
type AcceptanceInfo struct {
	Status AcceptanceStatus `json:"status"`
	Reason string           `json:"reason,omitempty"`
	Rule   string           `json:"rule,omitempty"`
}

// AuditEvent represents a logged sandbox operation.
type AuditEvent struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	SandboxID *uuid.UUID `json:"sandboxId,omitempty" db:"sandbox_id"`
	EventType string     `json:"eventType" db:"event_type"`
	EventTime time.Time  `json:"eventTime" db:"event_time"`
	Actor     string     `json:"actor,omitempty" db:"actor"`
	ActorType string     `json:"actorType" db:"actor_type"`

	// Source identifies the originating approval surface (per Decision D8 in
	// scenarios/swarm-manager/execute/agent-manager-sandbox-auto-apply-defaults/plan.md).
	// Empty on legacy events emitted before the auditability rollout; otherwise
	// one of the ApprovalSource values. Stored alongside Actor so audit queries
	// can both group by surface and identify the requesting principal.
	Source ApprovalSource `json:"source,omitempty" db:"source"`

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
	Name           string                 `json:"name,omitempty"`
	ScopePath      string                 `json:"scopePath"`
	ReservedPath   string                 `json:"reservedPath,omitempty"`
	ReservedPaths  []string               `json:"reservedPaths,omitempty"`
	NoLock         *bool                  `json:"noLock,omitempty"`
	ProjectRoot    string                 `json:"projectRoot,omitempty"`
	AuxiliaryRoots []string               `json:"auxiliaryRoots,omitempty"`
	Owner          string                 `json:"owner,omitempty"`
	OwnerType      OwnerType              `json:"ownerType,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Behavior       SandboxBehavior        `json:"behavior,omitempty"`

	// IdempotencyKey is an optional client-provided key for request deduplication.
	// If provided and a sandbox was already created with this key, that sandbox
	// is returned instead of creating a new one. This enables safe retries.
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
}

// ListFilter contains filters for listing sandboxes.
type ListFilter struct {
	Name        string    `json:"name,omitempty"`
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
	SandboxID   uuid.UUID     `json:"sandboxId"`
	Files       []*FileChange `json:"files"`
	UnifiedDiff string        `json:"unifiedDiff"`
	Generated   time.Time     `json:"generated"`
	Stats       DiffStats     `json:"stats"`

	// View mode support for full_diff and source modes
	Mode         ViewMode                `json:"mode,omitempty"`         // Requested view mode
	FileContents map[string]FileViewData `json:"fileContents,omitempty"` // Per-file content, keyed by file path

	// ArchiveState distinguishes archive-sourced responses from live
	// responses. Empty (zero value) means the diff was served from the
	// live overlay; the value is only set when the response originated
	// from a durable archive captured at terminal-status transition.
	//
	//   - empty       → live overlay (Active or Stopped sandbox; or
	//                   Creating with no overlay yet; or Error with a
	//                   missing upper dir). The diff Files list reflects
	//                   real-time data (or is empty for the no-overlay
	//                   cases).
	//   - "complete"  → archive row exists and per-file blobs are durable
	//                   on disk. Files list is the snapshot taken at
	//                   terminal transition.
	//   - "not_captured" → archive row exists but no blobs were written
	//                   (e.g. Error → Deleted transitions deliberately
	//                   skip blob capture). Files list is empty.
	//
	// Consumers (agent-manager adapter, UI components, CLI) can therefore
	// render three distinct states without inference: live data, archived
	// snapshot, or "no diff captured". See
	// docs/internal/ARCHIVE_DESIGN.md §3.
	ArchiveState ArchiveState `json:"archiveState,omitempty"`
}

// ArchiveState is the durability state of a diff snapshot. Two values
// only — see docs/internal/ARCHIVE_DESIGN.md §3 for the full taxonomy.
type ArchiveState string

const (
	// ArchiveStateComplete indicates the response came from a durable
	// archive row whose per-file blobs are present on disk. Live diffs
	// leave ArchiveState empty (zero value) rather than using this.
	ArchiveStateComplete ArchiveState = "complete"

	// ArchiveStateNotCaptured indicates the snapshot was deliberately
	// skipped at terminal transition (e.g. Error → Deleted). The
	// archive metadata row exists for History UI rendering; no blobs
	// exist on disk; the diff Files list is empty.
	ArchiveStateNotCaptured ArchiveState = "not_captured"
)

// IsValid reports whether s is one of the recognized ArchiveState values.
func (s ArchiveState) IsValid() bool {
	switch s {
	case ArchiveStateComplete, ArchiveStateNotCaptured:
		return true
	default:
		return false
	}
}

// DiffArchive is a durable record of a sandbox's diff at the moment it
// transitioned to a terminal status (Approved, Rejected, or Deleted).
// One row per sandbox; written transactionally with the status flip.
//
// Storage shape: metadata in SQLite (sandbox_diff_archives), content
// in per-sandbox content-addressed blobs on disk (api-core/storage
// ClassData). See docs/internal/ARCHIVE_DESIGN.md.
type DiffArchive struct {
	// SandboxID is the primary key — one archive per sandbox.
	SandboxID uuid.UUID `json:"sandboxId"`

	// SnapshotAt is the wall-clock time the snapshot was committed.
	SnapshotAt time.Time `json:"snapshotAt"`

	// ArchiveState distinguishes captured-with-content from deliberately
	// not-captured rows.
	ArchiveState ArchiveState `json:"archiveState"`

	// SandboxStatus is the terminal status the sandbox transitioned to
	// when the snapshot was taken — one of Approved, Rejected, Deleted.
	// Denormalized so retention queries can filter by status without
	// joining sandboxes.
	SandboxStatus Status `json:"sandboxStatus"`

	// Files is the per-file index. Empty when ArchiveState is
	// NotCaptured. Each entry references a content blob by SHA-256;
	// callers fetch the blob through the blobstore on demand.
	Files []ArchivedFileEntry `json:"files"`

	// Stats is the aggregate diff stats at snapshot time, identical in
	// shape to DiffResult.Stats.
	Stats DiffStats `json:"stats"`

	// UnifiedDiffSHA256 is the content-address of the unified-diff
	// text blob for this archive. Empty when NotCaptured.
	UnifiedDiffSHA256 string `json:"unifiedDiffSha256,omitempty"`

	// TotalBlobBytes is the sum of on-disk (gzipped) sizes of every
	// blob this archive owns. Used by retention to enforce size budgets.
	TotalBlobBytes int64 `json:"totalBlobBytes"`

	// ProjectRoot is denormalized from the sandbox at snapshot time so
	// retention can scope by project without rejoining sandboxes.
	ProjectRoot string `json:"projectRoot"`

	// Owner is denormalized for History tab filtering.
	Owner string `json:"owner,omitempty"`

	// AgentManagerRunID is denormalized for fast lookup of "find the
	// archive for this run" without joining applied_changes.
	AgentManagerRunID string `json:"agentManagerRunId,omitempty"`
}

// ArchivedFileEntry describes one file in a DiffArchive index. The
// per-file content is stored as a separate blob keyed by BlobSHA256;
// callers fetch the blob through the blobstore.
type ArchivedFileEntry struct {
	Path           string         `json:"path"`
	ChangeType     ChangeType     `json:"changeType"`
	Size           int64          `json:"size"`
	FileMode       int            `json:"fileMode,omitempty"`
	ApprovalStatus ApprovalStatus `json:"approvalStatus,omitempty"`

	// BlobSHA256 is the content-address of this file's blob in the
	// blobstore. Empty for change types that have no content (e.g.
	// some "deleted" entries where the prior content was not
	// retained, or empty added files).
	BlobSHA256 string `json:"blobSha256,omitempty"`
}

// ArchiveListFilter selects archives for the history listing
// endpoint. All fields are optional; zero values mean "no filter."
type ArchiveListFilter struct {
	// Statuses is a subset of {Approved, Rejected, Deleted}. Empty
	// means all three.
	Statuses []Status `json:"statuses,omitempty"`

	// ProjectRoot, Owner, AgentManagerRunID are exact-match filters.
	ProjectRoot       string `json:"projectRoot,omitempty"`
	Owner             string `json:"owner,omitempty"`
	AgentManagerRunID string `json:"agentManagerRunId,omitempty"`

	// Search is a free-text substring matched against owner,
	// agent_manager_run_id, and sandbox_id. Case-sensitive (SQLite
	// LIKE without COLLATE NOCASE) for predictable behavior.
	Search string `json:"search,omitempty"`

	// SnapshotAtFrom / SnapshotAtTo bound the snapshot_at column.
	// Zero values disable the respective bound.
	SnapshotAtFrom time.Time `json:"snapshotAtFrom,omitempty"`
	SnapshotAtTo   time.Time `json:"snapshotAtTo,omitempty"`

	// SortBy selects the order column. Allowed values:
	//   "snapshot_at"     (default)
	//   "total_blob_bytes"
	// Anything else is rejected at the repository layer.
	SortBy string `json:"sortBy,omitempty"`

	// SortDesc toggles descending order. Default is descending for
	// snapshot_at (newest first) — repository normalizes.
	SortDesc bool `json:"sortDesc,omitempty"`

	// Limit caps the result set. The repository clamps to a sane
	// upper bound; 0 means "use the default."
	Limit int `json:"limit,omitempty"`

	// Offset is the page offset. Page-based for now; cursor-based
	// pagination is a follow-up if the listing becomes large.
	Offset int `json:"offset,omitempty"`
}

// DiffStats summarizes the aggregate impact of a sandbox diff.
// FilesChanged is always FilesAdded + FilesModified + FilesDeleted.
type DiffStats struct {
	FilesChanged  int   `json:"filesChanged"`
	FilesAdded    int   `json:"filesAdded"`
	FilesModified int   `json:"filesModified"`
	FilesDeleted  int   `json:"filesDeleted"`
	LinesAdded    int   `json:"linesAdded"`
	LinesRemoved  int   `json:"linesRemoved"`
	TotalBytes    int64 `json:"totalBytes"`
}

// ApprovalSource is a typed enum identifying which surface originated an
// approve / reject / apply action. The value is recorded on the resulting
// state transition (audit events, provenance records) so reviewers can tell
// agent-manager auto-apply, GCT operator approval, the workspace-sandbox UI,
// and CLI invocations apart at a glance.
//
// SourceWorkspaceSandboxGC is system-initiated and is emitted only by the
// GC auto-deny path for expired manualReview=true sandboxes. Decode-time
// validation rejects it on inbound requests (agents/operators may not
// claim system identity). See Decision D8 in
// scenarios/swarm-manager/execute/agent-manager-sandbox-auto-apply-defaults/plan.md.
type ApprovalSource string

const (
	// SourceUnspecified means the caller did not set a source. Treated as
	// invalid for inbound requests.
	SourceUnspecified ApprovalSource = ""

	// SourceAgentManagerAutoApply marks an apply triggered by agent-manager
	// at run end (manualReview=false default).
	SourceAgentManagerAutoApply ApprovalSource = "agent-manager-auto-apply"

	// SourceGitControlTower marks an approval coming from the GCT
	// AI Changes review queue.
	SourceGitControlTower ApprovalSource = "git-control-tower"

	// SourceAgentManagerUI marks an approval from the agent-manager run-detail
	// diff view.
	SourceAgentManagerUI ApprovalSource = "agent-manager-ui"

	// SourceWorkspaceSandboxUI marks an approval from the workspace-sandbox
	// sandbox-detail diff view.
	SourceWorkspaceSandboxUI ApprovalSource = "workspace-sandbox-ui"

	// SourceCLI marks an approval/apply driven from a CLI surface.
	SourceCLI ApprovalSource = "cli"

	// SourceWorkspaceSandboxGC is system-initiated and is set ONLY by the
	// GC auto-deny path for manualReview-TTL expiry. Inbound requests
	// carrying this value are rejected — operators may not claim system
	// identity.
	SourceWorkspaceSandboxGC ApprovalSource = "workspace-sandbox-gc"
)

// IsValid reports whether s is a recognised source value (including the
// system-initiated value, which IsValidInbound rejects separately).
func (s ApprovalSource) IsValid() bool {
	switch s {
	case SourceUnspecified,
		SourceAgentManagerAutoApply,
		SourceGitControlTower,
		SourceAgentManagerUI,
		SourceWorkspaceSandboxUI,
		SourceCLI,
		SourceWorkspaceSandboxGC:
		return true
	default:
		return false
	}
}

// IsValidInbound reports whether s is acceptable on an inbound request
// (operator approval, agent-manager apply call, etc.). The system-initiated
// SourceWorkspaceSandboxGC value is rejected — only the GC reconciler may
// emit it.
func (s ApprovalSource) IsValidInbound() bool {
	switch s {
	case SourceAgentManagerAutoApply,
		SourceGitControlTower,
		SourceAgentManagerUI,
		SourceWorkspaceSandboxUI,
		SourceCLI:
		return true
	default:
		return false
	}
}

// ApprovalRequest contains the parameters for approving changes.
type ApprovalRequest struct {
	SandboxID  uuid.UUID   `json:"sandboxId"`
	Mode       string      `json:"mode"` // "all", "files", "hunks"
	FileIDs    []uuid.UUID `json:"fileIds,omitempty"`
	HunkRanges []HunkRange `json:"hunkRanges,omitempty"`
	Actor      string      `json:"actor,omitempty"`
	CommitMsg  string      `json:"commitMessage,omitempty"`

	// Source identifies the originating approval surface for audit. See
	// ApprovalSource. Zero-value is permitted on legacy callers during the
	// migration window; the GC-only value is rejected at decode time.
	Source ApprovalSource `json:"source,omitempty"`

	// OverrideAcceptance bypasses acceptance filtering and applies all changes.
	OverrideAcceptance bool `json:"overrideAcceptance,omitempty"`

	// Force bypasses conflict detection and applies changes even if the
	// canonical repo has changed since sandbox creation. Use with caution.
	// [OT-P2-002] Conflict Detection
	Force bool `json:"force,omitempty"`

	// CreateCommit controls whether to create a git commit after applying changes.
	// Default is false - changes are applied to the working tree only.
	// When true and CommitMsg is provided, a commit is created.
	CreateCommit bool `json:"createCommit,omitempty"`

	// Auditability-contract metadata forwarded from apply-at-run-end. These
	// are stamped onto the resulting AppliedChange rows so readers (web-console,
	// GCT) can group by run / conversation. Empty for operator-driven approvals.
	// Canonical enum values live in
	// packages/proto/schemas/workspace-sandbox/v1/domain/applied_change.proto.
	AgentManagerRunID string  `json:"agentManagerRunId,omitempty"`
	ConversationID    string  `json:"conversationId,omitempty"`
	Cost              float64 `json:"cost,omitempty"`
	RunOutcome        string  `json:"runOutcome,omitempty"`
}

// ApplyAtRunEndRequest carries agent-manager run-context metadata onto the
// run-end apply call. Per Decision D6, this is its own request type (not an
// extension of ApprovalRequest) so the agent-manager seam stays narrow and
// the operator approve / reject paths are not perturbed. The handler
// translates this into the same internal apply path used by ApprovalRequest;
// the per-file state-machine logic is shared.
//
// See scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md Findings 1
// and 2 for the contract this type encodes.
type ApplyAtRunEndRequest struct {
	SandboxID uuid.UUID `json:"sandboxId"`

	// AgentManagerRunID is the agent-manager run that produced these changes.
	// Recorded on the resulting ProvenanceRunGroup.
	AgentManagerRunID string `json:"agentManagerRunId"`

	// ConversationID is the agent-thread identifier (per Decision D7).
	// Spawner-supplied or inherited from a parent run; never reused
	// from Run.SessionID (runner-specific resume token).
	ConversationID string `json:"conversationId,omitempty"`

	// Cost is the total USD cost of the run, recorded on provenance.
	Cost float64 `json:"cost,omitempty"`

	// RunOutcome is the contract-canonical outcome ∈ {success, failure,
	// cancelled, timeout}. Apply behaviour is identical regardless of
	// outcome (Decision D5 mapping is lossy by design); the value is
	// metadata only.
	RunOutcome string `json:"runOutcome,omitempty"`

	// Source must be SourceAgentManagerAutoApply for apply-at-run-end calls.
	// The handler validates this; other sources are rejected.
	Source ApprovalSource `json:"source"`

	// Actor mirrors ApprovalRequest.Actor for attribution policies.
	Actor string `json:"actor,omitempty"`

	// CommitMsg / CreateCommit are forwarded to the underlying apply path.
	CommitMsg    string `json:"commitMessage,omitempty"`
	CreateCommit bool   `json:"createCommit,omitempty"`

	// Force mirrors ApprovalRequest.Force; default off.
	Force bool `json:"force,omitempty"`
}

// TurnCheckpointRequest carries the agent-manager turn context for the
// resumable post-turn sandbox lifecycle. Unlike approval, a turn checkpoint
// applies accepted changes, records provenance, releases mounts, and parks the
// same logical sandbox in StatusCheckpointed for later resume.
type TurnCheckpointRequest struct {
	SandboxID uuid.UUID `json:"sandboxId"`

	AgentManagerRunID string `json:"agentManagerRunId"`
	ConversationID    string `json:"conversationId,omitempty"`
	TurnID            string `json:"turnId,omitempty"`
	TurnSequence      int    `json:"turnSequence,omitempty"`

	Source     ApprovalSource `json:"source"`
	Actor      string         `json:"actor,omitempty"`
	RunOutcome string         `json:"runOutcome,omitempty"`
	Cost       float64        `json:"cost,omitempty"`

	CommitMsg    string `json:"commitMessage,omitempty"`
	CreateCommit bool   `json:"createCommit,omitempty"`
	Force        bool   `json:"force,omitempty"`
}

// TurnCheckpointResult reports the outcome of a turn checkpoint.
type TurnCheckpointResult struct {
	SandboxID        uuid.UUID `json:"sandboxId"`
	Status           Status    `json:"status"`
	Success          bool      `json:"success"`
	Applied          int       `json:"applied"`
	Failed           int       `json:"failed"`
	Remaining        int       `json:"remaining"`
	IsPartial        bool      `json:"isPartial"`
	CommitHash       string    `json:"commitHash,omitempty"`
	BaseCommitHash   string    `json:"baseCommitHash,omitempty"`
	CheckpointID     string    `json:"checkpointId,omitempty"`
	ErrorMsg         string    `json:"error,omitempty"`
	AppliedAt        time.Time `json:"appliedAt"`
	AppliedSizeBytes int64     `json:"appliedSizeBytes,omitempty"`
	DiffPath         string    `json:"diffPath,omitempty"`
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
	// AppliedSizeBytes is the authoritative total size of the files applied.
	AppliedSizeBytes int64 `json:"appliedSizeBytes,omitempty"`
	// DiffPath names the durable diff endpoint for this sandbox's archive.
	DiffPath string `json:"diffPath,omitempty"`

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
//
// The auditability fields (RunOutcome, ProvenanceState, ConversationID,
// CostUSD) follow the contract defined in
// packages/proto/schemas/workspace-sandbox/v1/domain/applied_change.proto
// and are read by the GCT pending-AI-provenance-hardening initiative and
// the web-console review queue. The proto package path is the version;
// breaking changes ship as v2, not in-place mutation of the v1 enums.
type AppliedChange struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	SandboxID         uuid.UUID  `json:"sandboxId" db:"sandbox_id"`
	SandboxOwner      string     `json:"sandboxOwner" db:"sandbox_owner"`
	SandboxOwnerType  string     `json:"sandboxOwnerType" db:"sandbox_owner_type"`
	FilePath          string     `json:"filePath" db:"file_path"`
	ProjectRoot       string     `json:"projectRoot" db:"project_root"`
	ChangeType        string     `json:"changeType" db:"change_type"`
	FileSize          int64      `json:"fileSize" db:"file_size"`
	AgentManagerRunID string     `json:"agentManagerRunId,omitempty" db:"agent_manager_run_id"`
	AppliedAt         time.Time  `json:"appliedAt" db:"applied_at"`
	CommittedAt       *time.Time `json:"committedAt,omitempty" db:"committed_at"`
	CommitHash        string     `json:"commitHash,omitempty" db:"commit_hash"`
	CommitMessage     string     `json:"commitMessage,omitempty" db:"commit_message"`

	RunOutcome         string     `json:"runOutcome,omitempty" db:"run_outcome"`
	ProvenanceState    string     `json:"state,omitempty" db:"provenance_state"`
	ConversationID     string     `json:"conversationId,omitempty" db:"conversation_id"`
	CostUSD            float64    `json:"costUsd,omitempty" db:"cost_usd"`
	ResolutionAttempts int        `json:"resolutionAttempts" db:"resolution_attempts"`
	UnresolvableAt     *time.Time `json:"unresolvableAt,omitempty" db:"unresolvable_at"`
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
	FilePath          string    `json:"filePath"`
	RelativePath      string    `json:"relativePath"`
	ChangeType        string    `json:"changeType"`
	SandboxID         uuid.UUID `json:"sandboxId"`
	SandboxOwner      string    `json:"sandboxOwner"`
	AgentManagerRunID string    `json:"agentManagerRunId,omitempty"`
	AppliedAt         time.Time `json:"appliedAt"`
	// Status indicates the file's current state relative to git
	// "pending" = still uncommitted, "already_committed" = committed externally
	Status string `json:"status"`
}

// CommitPreviewRequest contains parameters for the commit preview endpoint.
type CommitPreviewRequest struct {
	ProjectRoot string      `json:"projectRoot,omitempty"`
	SandboxIDs  []uuid.UUID `json:"sandboxIds,omitempty"`
	FilePaths   []string    `json:"filePaths,omitempty"`
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

// --- External Commit Notification Types ---

// MarkCommittedRequest is sent by external tools (e.g., git-control-tower) to notify
// workspace-sandbox that files have been committed outside its own commit flow.
type MarkCommittedRequest struct {
	ProjectRoot   string   `json:"projectRoot"`
	FilePaths     []string `json:"filePaths"`
	CommitHash    string   `json:"commitHash"`
	CommitMessage string   `json:"commitMessage"`
}

// MarkCommittedResult reports the outcome of marking files as committed.
type MarkCommittedResult struct {
	MarkedCount   int `json:"markedCount"`
	NotFoundCount int `json:"notFoundCount"`
}

// --- Provenance By Run Types ---

// ProvenanceRunGroup groups pending applied changes by agent-manager run ID.
//
// The RunOutcome / ConversationID / CostUSD fields are part of the
// auditability contract (Findings 1–2 in
// scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md). They are
// written on the wire by agent-manager's apply-at-run-end call and
// surfaced through this response shape. Canonical enum values live in
// packages/proto/schemas/workspace-sandbox/v1/domain/applied_change.proto.
type ProvenanceRunGroup struct {
	RunID           string           `json:"runId"`
	SandboxID       string           `json:"sandboxId"`
	SandboxOwner    string           `json:"sandboxOwner"`
	Files           []ProvenanceFile `json:"files"`
	LatestAppliedAt time.Time        `json:"latestAppliedAt"`

	// RunOutcome ∈ {success, failure, cancelled, timeout}. Empty for legacy
	// run groups recorded before the auditability rollout.
	RunOutcome string `json:"runOutcome,omitempty"`

	// ConversationID groups runs that belong to the same agent thread.
	// See Run.ConversationID on the agent-manager side.
	ConversationID string `json:"conversationId,omitempty"`

	// CostUSD is the total cost of the originating run, in USD.
	CostUSD float64 `json:"costUsd,omitempty"`
}

// ProvenanceFile represents a single file within a provenance run group.
//
// State is the per-file lifecycle state from the auditability contract
// (Finding 2): "applied" (in-acceptance, persisted to canonical repo),
// "pending-review" (out-of-acceptance, sandbox-resident until operator
// action), or "denied" (operator declined or auto-deny on TTL expiry).
// Empty for legacy records.
type ProvenanceFile struct {
	FilePath     string    `json:"filePath"`
	RelativePath string    `json:"relativePath"`
	ChangeType   string    `json:"changeType"`
	AppliedAt    time.Time `json:"appliedAt"`

	// State ∈ {applied, pending-review, denied}; empty on legacy rows.
	State ProvenanceFileState `json:"state,omitempty"`
}

// ProvenanceFileState is the per-file lifecycle state from the auditability
// contract. The canonical wire-value definition lives in
// packages/proto/schemas/workspace-sandbox/v1/domain/applied_change.proto
// (workspace_sandbox.v1.FileState). The Go strings here mirror the
// kebab-cased projection of that enum and are pinned by
// TestProvenanceFileStateMatchesProto in this package.
type ProvenanceFileState string

const (
	// ProvenanceFileStateApplied means the change has been applied to the
	// canonical repository.
	ProvenanceFileStateApplied ProvenanceFileState = "applied"

	// ProvenanceFileStatePendingReview means the change is held in the
	// sandbox awaiting operator approval (out-of-acceptance changes when
	// AutoApply runs, or all changes when ManualReview=true).
	ProvenanceFileStatePendingReview ProvenanceFileState = "pending-review"

	// ProvenanceFileStateDenied means the operator declined the change, or
	// the GC reconciler auto-denied it on manualReview-TTL expiry.
	ProvenanceFileStateDenied ProvenanceFileState = "denied"
)

// IsValid reports whether s is a recognised state value (empty allowed for
// legacy records that pre-date the auditability rollout).
func (s ProvenanceFileState) IsValid() bool {
	switch s {
	case "", ProvenanceFileStateApplied, ProvenanceFileStatePendingReview, ProvenanceFileStateDenied:
		return true
	default:
		return false
	}
}
