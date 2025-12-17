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

// Create creates a new sandbox.
func (s *Service) Create(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
	// Determine project root
	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		return nil, fmt.Errorf("project root is required")
	}

	// Validate and normalize scope path
	normalizedPath, err := ValidateScopePath(req.ScopePath, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid scope path: %w", err)
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
		ID:            uuid.New(),
		ScopePath:     normalizedPath,
		ProjectRoot:   projectRoot,
		Owner:         req.Owner,
		OwnerType:     req.OwnerType,
		Status:        types.StatusCreating,
		Driver:        string(s.driver.Type()),
		DriverVersion: s.driver.Version(),
		Tags:          req.Tags,
		Metadata:      req.Metadata,
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
			"scopePath":   sandbox.ScopePath,
			"projectRoot": sandbox.ProjectRoot,
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
func (s *Service) Stop(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Use explicit state transition check
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
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Cleanup driver resources
	if err := s.driver.Cleanup(ctx, sandbox); err != nil {
		// Log but continue
		fmt.Printf("warning: driver cleanup failed: %v\n", err)
	}

	// Mark as deleted in database
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete sandbox: %w", err)
	}

	s.repo.LogAuditEvent(ctx, &types.AuditEvent{
		SandboxID: &sandbox.ID,
		EventType: "deleted",
	})

	return nil
}

// GetDiff generates a diff for the sandbox changes.
func (s *Service) GetDiff(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if sandbox.UpperDir == "" {
		return nil, fmt.Errorf("sandbox has no upper directory")
	}

	// Get changed files from driver
	changes, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	// Generate unified diff
	gen := diff.NewGenerator()
	return gen.GenerateDiff(ctx, sandbox, changes)
}

// Approve applies sandbox changes to the canonical repo.
func (s *Service) Approve(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error) {
	sandbox, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}

	// Use explicit state transition check
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
func (s *Service) Reject(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Use explicit state transition check
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
