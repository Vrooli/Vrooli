// Package sandbox. service.go: the Service struct + its construction
// surface. Operation methods (Create/Stop/Approve/etc.) live in
// service_*.go alongside the helpers they use.
//
// File map:
//
//	service.go             ServiceAPI interface, Service struct, options, NewService
//	service_create.go      Create + idempotency + validation + mount
//	service_lifecycle.go   Get/List/Start/Stop/Delete + lifecycle policy
//	service_review.go      Diff/Approve/ApplyAtRunEnd/Reject/Discard/Conflicts/Rebase
//	service_pending.go     Pending changes, commit preview, mark-committed, provenance
//	service_acceptance.go  Per-file acceptance evaluation + glob matching
//	service_paths.go       Mount-path bookkeeping + ValidatePath
//	service_audit.go       Audit logging + agent-manager notification
package sandbox

import (
	"context"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/metrics"
	"workspace-sandbox/internal/policy"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/types"
)

// --- Service Interface ---
// The ServiceAPI interface defines the sandbox service contract.
// This interface documents all operations available to handlers and enables
// testing with mock implementations.

// ServiceAPI defines the contract for sandbox service operations.
type ServiceAPI interface {
	// Create creates a new sandbox for the specified scope path.
	Create(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error)

	// Get retrieves a sandbox by ID. Returns NotFoundError if not found.
	Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)

	// List retrieves sandboxes matching the filter.
	List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error)

	// Stop unmounts a sandbox but preserves its data for later review.
	// Returns StateError if sandbox cannot be stopped.
	Stop(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)

	// Start remounts a stopped sandbox to resume work.
	// Returns StateError if sandbox cannot be started.
	Start(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)

	// Delete removes a sandbox and all its data.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetDiff generates a diff for the sandbox changes.
	GetDiff(ctx context.Context, id uuid.UUID) (*types.DiffResult, error)

	// Approve applies sandbox changes to the canonical repo.
	// Returns StateError if sandbox cannot be approved.
	Approve(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error)

	// ApplyAtRunEnd is the agent-manager run-end apply path from the
	// auditability contract (Decision D6). It carries run-context
	// metadata onto the apply call and routes through the same internal
	// apply path as Approve to guarantee no per-file state-machine drift
	// between the operator and auto-apply surfaces.
	//
	// The Source field must be SourceAgentManagerAutoApply; other inbound
	// values are rejected.
	ApplyAtRunEnd(ctx context.Context, req *types.ApplyAtRunEndRequest) (*types.ApprovalResult, error)

	// Reject marks sandbox changes as rejected.
	// Returns StateError if sandbox cannot be rejected.
	Reject(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error)

	// Discard removes specific files from a sandbox without applying them.
	// This allows rejecting individual files while keeping others pending.
	Discard(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error)

	// GetWorkspacePath returns the path where sandbox operations should occur.
	// Returns error if sandbox is not mounted.
	GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error)

	// CheckConflicts checks if the canonical repo has changed since
	// sandbox creation and identifies any conflicting files.
	CheckConflicts(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error)

	// Rebase updates the sandbox's BaseCommitHash to the current repo state.
	Rebase(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error)

	// ValidatePath checks if a path is valid for use as a sandbox scope.
	ValidatePath(ctx context.Context, path, projectRoot string) (*types.PathValidationResult, error)

	// GetPendingChanges returns pending (uncommitted) changes grouped by sandbox.
	GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error)

	// GetFileProvenance returns the history of changes for a specific file.
	GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error)

	// GetCommitPreview returns a preview of what would be committed.
	GetCommitPreview(ctx context.Context, req *types.CommitPreviewRequest) (*types.CommitPreviewResult, error)

	// CommitPending commits pending changes to git and updates provenance records.
	CommitPending(ctx context.Context, req *types.CommitPendingRequest) (*types.CommitPendingResult, error)

	// MarkCommitted marks pending changes as committed for files that
	// were committed by an external tool (e.g., git-control-tower).
	MarkCommitted(ctx context.Context, req *types.MarkCommittedRequest) (*types.MarkCommittedResult, error)

	// GetProvenanceByRun returns pending applied changes grouped by
	// agent-manager run ID.
	GetProvenanceByRun(ctx context.Context, projectRoot string) ([]types.ProvenanceRunGroup, error)
}

// Compile-time guarantee that Service implements ServiceAPI.
var _ ServiceAPI = (*Service)(nil)

// --- Service Implementation ---

// Service provides high-level sandbox operations. Methods are
// distributed across service_*.go files by responsibility; the struct
// itself only declares fields here.
type Service struct {
	repo    repository.Repository
	driver  driver.Driver
	config  ServiceConfig
	clock   clock.Clock
	audit   audit.Emitter
	starter process.Starter

	// Policies — volatile decision points wired via ServiceOption.
	attributionPolicy policy.AttributionPolicy
	validationPolicy  policy.ValidationPolicy
	teardownPolicy    policy.TeardownPolicy

	// gitOps is the seam for git operations, enabling test isolation.
	// When nil, the default production GitOps is used; tests inject a
	// MockGitOps via WithGitOps to avoid touching real repos.
	gitOps diff.GitOperations

	// metrics is the optional observability sink. Production wires
	// metrics.Default(); tests typically leave it nil so atomic counters
	// don't bleed across cases. Always nil-checked by callers.
	metrics *metrics.Collector

	// procFS lets tests inject a synthetic /proc for the deterministic
	// Delete-time daemon kill. nil → production scans the real /proc
	// via NewRealProcFS().
	procFS ProcFS
}

// ServiceConfig holds service configuration.
type ServiceConfig struct {
	DefaultProjectRoot      string
	MaxSandboxes            int
	DefaultTTL              time.Duration
	DefaultNoLock           bool
	AgentManagerURL         string
	AgentManagerSyncEnabled bool
	AgentManagerSyncTimeout time.Duration
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		MaxSandboxes:            1000,
		DefaultTTL:              24 * time.Hour,
		AgentManagerSyncEnabled: true,
		AgentManagerSyncTimeout: 5 * time.Second,
	}
}

// SetDefaultNoLock updates the default NoLock setting at runtime.
// Called by the config update handler when the user toggles scope locking.
func (s *Service) SetDefaultNoLock(v bool) {
	s.config.DefaultNoLock = v
}

// resolveNoLock applies the config default for NoLock when the caller
// didn't specify. If req.NoLock is nil (not set in JSON), uses
// s.config.DefaultNoLock; if explicitly set, respects the caller's choice.
func (s *Service) resolveNoLock(req *types.CreateRequest) {
	if req.NoLock == nil {
		v := s.config.DefaultNoLock
		req.NoLock = &v
	}
}

// ServiceOption configures the service.
type ServiceOption func(*Service)

// WithAttributionPolicy sets the attribution policy.
func WithAttributionPolicy(p policy.AttributionPolicy) ServiceOption {
	return func(s *Service) {
		s.attributionPolicy = p
	}
}

// WithValidationPolicy sets the validation policy.
func WithValidationPolicy(p policy.ValidationPolicy) ServiceOption {
	return func(s *Service) {
		s.validationPolicy = p
	}
}

// WithTeardownPolicy sets the teardown policy. Pre-teardown hooks
// allow external systems to gracefully evacuate processes from the
// sandbox's merged directory before the filesystem disappears.
func WithTeardownPolicy(p policy.TeardownPolicy) ServiceOption {
	return func(s *Service) {
		s.teardownPolicy = p
	}
}

// WithMetrics wires a metrics collector. Production wires
// metrics.Default(); tests usually leave it nil.
func WithMetrics(m *metrics.Collector) ServiceOption {
	return func(s *Service) {
		s.metrics = m
	}
}

// WithProcFS overrides the /proc view used by the deterministic
// Delete-time daemon kill. Tests inject sandboxiface.FakeProcFS;
// production leaves this unset so the real /proc is scanned.
func WithProcFS(p ProcFS) ServiceOption {
	return func(s *Service) {
		s.procFS = p
	}
}

// WithGitOps sets the git operations implementation. Primary seam
// for test isolation of git-related functionality.
func WithGitOps(g diff.GitOperations) ServiceOption {
	return func(s *Service) {
		s.gitOps = g
	}
}

// NewService creates a new sandbox service. clk and emitter are
// required:
//
//   - clk: idle timeouts, audit timestamps, manual-review TTL
//     evaluation, and the per-sandbox auto-heal clock all flow
//     through it. Production wires clock.System{}; tests wire
//     FakeClock so time-dependent behavior is deterministic.
//   - emitter: every audit event (created, approved, rejected,
//     auto-heal-failed, manual-review-ttl-expired, etc.) goes
//     through it. Production wires audit.NewRepoEmitter(repo.LogAuditEvent, clk);
//     tests wire mocks.NewFakeEmitter(clk) and assert via
//     assertx.AssertAuditEvents.
func NewService(repo repository.Repository, drv driver.Driver, cfg ServiceConfig, clk clock.Clock, emitter audit.Emitter, starter process.Starter, opts ...ServiceOption) *Service {
	if clk == nil {
		panic("sandbox.NewService: clock is required")
	}
	if emitter == nil {
		panic("sandbox.NewService: audit emitter is required")
	}
	if starter == nil {
		panic("sandbox.NewService: starter is required")
	}
	s := &Service{
		repo:    repo,
		driver:  drv,
		config:  cfg,
		clock:   clk,
		audit:   emitter,
		starter: starter,
		// Defaults: no-op policies + production GitOps backed by starter.
		validationPolicy: policy.NewNoOpValidationPolicy(),
		teardownPolicy:   policy.NewNoOpTeardownPolicy(),
		gitOps:           diff.NewGitOps(starter),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}
