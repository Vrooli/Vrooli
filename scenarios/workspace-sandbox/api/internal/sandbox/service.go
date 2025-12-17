package sandbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/policy"
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

	// Delete removes a sandbox and all its data.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetDiff generates a diff for the sandbox changes.
	GetDiff(ctx context.Context, id uuid.UUID) (*types.DiffResult, error)

	// Approve applies sandbox changes to the canonical repo.
	// Returns StateError if sandbox cannot be approved.
	Approve(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error)

	// Reject marks sandbox changes as rejected.
	// Returns StateError if sandbox cannot be rejected.
	Reject(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error)

	// GetWorkspacePath returns the path where sandbox operations should occur.
	// Returns error if sandbox is not mounted.
	GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error)
}

// Verify Service implements ServiceAPI interface at compile time.
var _ ServiceAPI = (*Service)(nil)

// --- Service Implementation ---

// Service provides high-level sandbox operations.
type Service struct {
	repo   repository.Repository
	driver driver.Driver
	config ServiceConfig

	// Policies - these define the volatile behavior of the service.
	// Using interfaces allows behavior to be configured without code changes.
	approvalPolicy    policy.ApprovalPolicy
	attributionPolicy policy.AttributionPolicy
	validationPolicy  policy.ValidationPolicy
}

// ServiceConfig holds service configuration.
type ServiceConfig struct {
	DefaultProjectRoot string
	MaxSandboxes       int
	DefaultTTL         time.Duration
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		MaxSandboxes: 1000,
		DefaultTTL:   24 * time.Hour,
	}
}

// ServiceOption configures the service.
type ServiceOption func(*Service)

// WithApprovalPolicy sets the approval policy.
func WithApprovalPolicy(p policy.ApprovalPolicy) ServiceOption {
	return func(s *Service) {
		s.approvalPolicy = p
	}
}

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

// NewService creates a new sandbox service.
func NewService(repo repository.Repository, drv driver.Driver, cfg ServiceConfig, opts ...ServiceOption) *Service {
	s := &Service{
		repo:   repo,
		driver: drv,
		config: cfg,
		// Default policies (no-op implementations for backwards compatibility)
		validationPolicy: policy.NewNoOpValidationPolicy(),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Create creates a new sandbox for the specified scope path.
//
// # Required Fields
//
// Either req.ProjectRoot must be set, or ServiceConfig.DefaultProjectRoot must
// be configured. ScopePath is optional; if empty, defaults to the project root.
//
// # Idempotency
//
// If req.IdempotencyKey is provided, the system first checks if a sandbox was
// already created with that key. If found, the existing sandbox is returned
// without creating a duplicate. This enables safe retries of create requests.
//
// # Assumptions Made
//
//   - The project root directory exists and is accessible
//   - The driver is available (overlayfs on Linux)
//   - The database is reachable for metadata storage
//
// # Errors
//
// Returns ValidationError for invalid input, ScopeConflictError if the scope
// overlaps with an existing sandbox, or DriverError if mounting fails.
func (s *Service) Create(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
	// --- Idempotency Check ---
	// If an idempotency key is provided, check if we already processed this request.
	// This enables safe retries: if a client retries a create request, they get
	// the same sandbox back instead of an error or a duplicate.
	if req.IdempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("failed to check idempotency key: %w", err)
		}
		if existing != nil {
			// Found existing sandbox with this key - return it (idempotent success)
			// Note: We don't verify parameters match because the idempotency key
			// is the authoritative indicator that this is the "same" request.
			return existing, nil
		}
	}

	// ASSUMPTION: Either request or config provides project root
	// GUARD: Check this explicitly and provide helpful error message
	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		return nil, types.NewValidationErrorWithHint(
			"projectRoot",
			"project root is required but not provided",
			"Set projectRoot in the request body, or configure PROJECT_ROOT environment variable",
		)
	}

	// ASSUMPTION: Project root is a valid, accessible directory
	// GUARD: Validate this before proceeding
	normalizedPath, err := ValidateScopePath(req.ScopePath, projectRoot)
	if err != nil {
		return nil, types.NewValidationErrorWithHint(
			"scopePath",
			fmt.Sprintf("invalid scope path: %v", err),
			"Ensure the path exists within the project root and contains no invalid characters",
		)
	}

	// Check for overlapping sandboxes
	conflicts, err := s.repo.CheckScopeOverlap(ctx, normalizedPath, projectRoot, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check scope overlap: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, &types.ScopeConflictError{Conflicts: conflicts}
	}

	// Create sandbox record
	sandbox := &types.Sandbox{
		ID:             uuid.New(),
		ScopePath:      normalizedPath,
		ProjectRoot:    projectRoot,
		Owner:          req.Owner,
		OwnerType:      req.OwnerType,
		Status:         types.StatusCreating,
		Driver:         string(s.driver.Type()),
		DriverVersion:  s.driver.Version(),
		Tags:           req.Tags,
		Metadata:       req.Metadata,
		IdempotencyKey: req.IdempotencyKey,
	}

	if sandbox.OwnerType == "" {
		sandbox.OwnerType = types.OwnerTypeUser
	}

	// Insert into database
	if err := s.repo.Create(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to create sandbox record: %w", err)
	}

	// Mount the overlay
	paths, err := s.driver.Mount(ctx, sandbox)
	if err != nil {
		// Update status to error
		sandbox.Status = types.StatusError
		sandbox.ErrorMsg = err.Error()
		s.repo.Update(ctx, sandbox)
		return sandbox, fmt.Errorf("failed to mount sandbox: %w", err)
	}

	// Update with mount paths
	sandbox.LowerDir = paths.LowerDir
	sandbox.UpperDir = paths.UpperDir
	sandbox.WorkDir = paths.WorkDir
	sandbox.MergedDir = paths.MergedDir
	sandbox.Status = types.StatusActive

	if err := s.repo.Update(ctx, sandbox); err != nil {
		// Cleanup on failure
		s.driver.Cleanup(ctx, sandbox)
		return nil, fmt.Errorf("failed to update sandbox: %w", err)
	}

	// Log audit event
	s.repo.LogAuditEvent(ctx, &types.AuditEvent{
		SandboxID: &sandbox.ID,
		EventType: "created",
		Actor:     req.Owner,
		ActorType: string(req.OwnerType),
		Details: map[string]interface{}{
			"scopePath":      sandbox.ScopePath,
			"projectRoot":    sandbox.ProjectRoot,
			"idempotencyKey": req.IdempotencyKey,
		},
	})

	return sandbox, nil
}

// Get retrieves a sandbox by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	sandbox, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox: %w", err)
	}
	if sandbox == nil {
		return nil, types.NewNotFoundError(id.String())
	}
	return sandbox, nil
}

// List retrieves sandboxes matching the filter.
func (s *Service) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	return s.repo.List(ctx, filter)
}

// Stop unmounts a sandbox but preserves its data.
//
// # Idempotency
//
// This operation is idempotent: calling Stop on an already-stopped sandbox
// returns success with the current sandbox state. This enables safe retries
// without requiring callers to check the status first.
func (s *Service) Stop(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// --- Idempotency Check ---
	// If already stopped, return success (no-op)
	// This makes Stop safe to retry without error
	if sandbox.Status == types.StatusStopped {
		return sandbox, nil
	}

	// Use explicit state transition check for other states
	if err := types.CanStop(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	// Unmount
	if err := s.driver.Unmount(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to unmount sandbox: %w", err)
	}

	// Update status
	now := time.Now()
	sandbox.Status = types.StatusStopped
	sandbox.StoppedAt = &now

	if err := s.repo.Update(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to update sandbox: %w", err)
	}

	s.repo.LogAuditEvent(ctx, &types.AuditEvent{
		SandboxID: &sandbox.ID,
		EventType: "stopped",
	})

	return sandbox, nil
}

// Delete removes a sandbox and all its data.
//
// # Idempotency
//
// This operation is idempotent: calling Delete on an already-deleted sandbox
// returns success without error. This enables safe retries and simplifies
// cleanup workflows.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		// If sandbox not found, treat as already deleted (idempotent success)
		if _, ok := err.(*types.NotFoundError); ok {
			return nil
		}
		return err
	}

	// --- Idempotency Check ---
	// If already deleted, return success (no-op)
	if sandbox.Status == types.StatusDeleted {
		return nil
	}

	// Cleanup driver resources
	if err := s.driver.Cleanup(ctx, sandbox); err != nil {
		// Log but continue - cleanup failures shouldn't block deletion
		fmt.Printf("warning: driver cleanup failed: %v\n", err)
	}

	// Mark as deleted in database
	if err := s.repo.Delete(ctx, id); err != nil {
		// If already deleted (race condition), treat as success
		if err.Error() == "sandbox not found or already deleted" {
			return nil
		}
		return fmt.Errorf("failed to delete sandbox: %w", err)
	}

	s.repo.LogAuditEvent(ctx, &types.AuditEvent{
		SandboxID: &sandbox.ID,
		EventType: "deleted",
	})

	return nil
}

// GetDiff generates a diff for the sandbox changes.
//
// # Preconditions
//
// The sandbox must be in a state where diff generation is valid (Active, Stopped,
// or terminal states for historical view). The overlay directories must exist.
//
// # Assumptions Made
//
//   - UpperDir contains the writable layer with modifications
//   - LowerDir contains the read-only original files
//   - The filesystem is in a consistent state (no writes in progress)
//   - The 'diff' command is available on the system for modified files
//
// # Returns
//
// A DiffResult containing the list of changed files and a unified diff string.
// Returns empty diff if no changes were made.
func (s *Service) GetDiff(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// ASSUMPTION: Diff can be generated for this status
	// GUARD: Check explicitly with helpful error
	if err := types.CanGenerateDiff(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	// ASSUMPTION: UpperDir is set when sandbox was created
	// GUARD: Fail with clear message if this invariant is violated
	if sandbox.UpperDir == "" {
		return nil, &types.ValidationError{
			Field:   "upperDir",
			Message: "sandbox upper directory not initialized (internal error)",
			Hint:    "This indicates the sandbox was not properly created. Delete and recreate it.",
		}
	}

	// ASSUMPTION: LowerDir is set (required for diff generation)
	// GUARD: Check this as well for completeness
	if sandbox.LowerDir == "" {
		return nil, &types.ValidationError{
			Field:   "lowerDir",
			Message: "sandbox lower directory not initialized (internal error)",
			Hint:    "This indicates the sandbox was not properly created. Delete and recreate it.",
		}
	}

	// Get changed files from driver
	changes, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, types.NewDriverError("getChangedFiles", err)
	}

	// Handle the case of no changes gracefully
	if len(changes) == 0 {
		return &types.DiffResult{
			SandboxID:   sandbox.ID,
			Files:       []*types.FileChange{},
			UnifiedDiff: "",
			Generated:   time.Now(),
		}, nil
	}

	// Generate unified diff
	gen := diff.NewGenerator()
	return gen.GenerateDiff(ctx, sandbox, changes)
}

// Approve applies sandbox changes to the canonical repo.
//
// # Idempotency
//
// This operation is idempotent: calling Approve on an already-approved sandbox
// returns a success result indicating the prior approval. This enables safe retries
// without re-applying changes or creating duplicate commits.
func (s *Service) Approve(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error) {
	sandbox, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}

	// --- Idempotency Check ---
	// If already approved, return success without re-applying changes
	// This makes Approve safe to retry after network timeouts, etc.
	if sandbox.Status == types.StatusApproved {
		return &types.ApprovalResult{
			Success:   true,
			Applied:   0, // No changes applied this time
			AppliedAt: *sandbox.ApprovedAt,
		}, nil
	}

	// Use explicit state transition check for other states
	if err := types.CanApprove(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	// Run approval policy validation if configured
	if s.approvalPolicy != nil {
		if err := s.approvalPolicy.ValidateApproval(ctx, sandbox, req); err != nil {
			return nil, fmt.Errorf("approval policy validation failed: %w", err)
		}
	}

	// Get all changes
	changes, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes: %w", err)
	}

	// Filter changes if specific files requested
	if req.Mode == "files" && len(req.FileIDs) > 0 {
		changes = diff.FilterChanges(changes, req.FileIDs)
	}

	if len(changes) == 0 {
		return &types.ApprovalResult{
			Success:   true,
			Applied:   0,
			AppliedAt: time.Now(),
		}, nil
	}

	// Run validation policy hooks before applying
	if s.validationPolicy != nil {
		if err := s.validationPolicy.ValidateBeforeApply(ctx, sandbox, changes); err != nil {
			return nil, fmt.Errorf("pre-apply validation failed: %w", err)
		}
	}

	// Generate diff for selected changes
	gen := diff.NewGenerator()
	diffResult, err := gen.GenerateDiff(ctx, sandbox, changes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff: %w", err)
	}

	// Determine commit message and author using attribution policy
	commitMsg := req.CommitMsg
	author := req.Actor
	if s.attributionPolicy != nil {
		if commitMsg == "" {
			commitMsg = s.attributionPolicy.GetCommitMessage(ctx, sandbox, changes, req.CommitMsg)
		}
		author = s.attributionPolicy.GetCommitAuthor(ctx, sandbox, req.Actor)

		// Append co-authors if any
		coAuthors := s.attributionPolicy.GetCoAuthors(ctx, sandbox, req.Actor)
		if len(coAuthors) > 0 {
			commitMsg = policy.FormatCommitMessage(commitMsg, coAuthors)
		}
	}

	// Apply the diff
	patcher := diff.NewPatcher()
	applyResult, err := patcher.ApplyDiff(ctx, sandbox.ProjectRoot, diffResult.UnifiedDiff, diff.ApplyOptions{
		CommitMsg: commitMsg,
		Author:    author,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to apply diff: %w", err)
	}

	if !applyResult.Success {
		return &types.ApprovalResult{
			Success:   false,
			Failed:    len(changes),
			ErrorMsg:  fmt.Sprintf("patch application failed: %v", applyResult.Errors),
			AppliedAt: time.Now(),
		}, nil
	}

	// Update sandbox status
	now := time.Now()
	sandbox.Status = types.StatusApproved
	sandbox.ApprovedAt = &now
	s.repo.Update(ctx, sandbox)

	s.repo.LogAuditEvent(ctx, &types.AuditEvent{
		SandboxID: &sandbox.ID,
		EventType: "approved",
		Actor:     req.Actor,
		Details: map[string]interface{}{
			"filesApplied": len(changes),
			"commitHash":   applyResult.CommitHash,
		},
	})

	return &types.ApprovalResult{
		Success:    true,
		Applied:    len(changes),
		CommitHash: applyResult.CommitHash,
		AppliedAt:  now,
	}, nil
}

// Reject marks sandbox changes as rejected.
//
// # Idempotency
//
// This operation is idempotent: calling Reject on an already-rejected sandbox
// returns success with the current sandbox state. This enables safe retries
// without requiring callers to check the status first.
func (s *Service) Reject(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// --- Idempotency Check ---
	// If already rejected, return success (no-op)
	// This makes Reject safe to retry without error
	if sandbox.Status == types.StatusRejected {
		return sandbox, nil
	}

	// Use explicit state transition check for other states
	if err := types.CanReject(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	sandbox.Status = types.StatusRejected
	if err := s.repo.Update(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to update sandbox: %w", err)
	}

	s.repo.LogAuditEvent(ctx, &types.AuditEvent{
		SandboxID: &sandbox.ID,
		EventType: "rejected",
		Actor:     actor,
	})

	return sandbox, nil
}

// GetWorkspacePath returns the path where sandbox operations should occur.
func (s *Service) GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}

	// Use explicit state check for workspace path availability
	if err := types.CanGetWorkspacePath(sandbox.Status); err != nil {
		return "", err
	}

	return sandbox.MergedDir, nil
}

// Legacy error type aliases for backwards compatibility.
// New code should use types.ScopeConflictError and types.NotFoundError directly.

// ScopeConflictError is an alias for types.ScopeConflictError.
// Deprecated: Use types.ScopeConflictError directly.
type ScopeConflictError = types.ScopeConflictError

// NotFoundError is an alias for types.NotFoundError.
// Deprecated: Use types.NotFoundError directly.
type NotFoundError = types.NotFoundError
