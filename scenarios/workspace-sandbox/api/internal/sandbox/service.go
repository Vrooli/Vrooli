package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// Reject marks sandbox changes as rejected.
	// Returns StateError if sandbox cannot be rejected.
	Reject(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error)

	// Discard removes specific files from a sandbox without applying them.
	// This allows rejecting individual files while keeping others pending.
	Discard(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error)

	// GetWorkspacePath returns the path where sandbox operations should occur.
	// Returns error if sandbox is not mounted.
	GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error)

	// --- Retry/Rebase Workflow (OT-P2-003) ---

	// CheckConflicts checks if the canonical repo has changed since sandbox creation
	// and identifies any conflicting files.
	CheckConflicts(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error)

	// Rebase updates the sandbox's BaseCommitHash to the current repo state.
	// This allows the sandbox to be aware of new changes in the canonical repo
	// and enables accurate conflict detection for subsequent approvals.
	Rebase(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error)

	// ValidatePath checks if a path is valid for use as a sandbox scope.
	// This enables the UI to validate paths before attempting to create a sandbox.
	ValidatePath(ctx context.Context, path, projectRoot string) (*types.PathValidationResult, error)

	// --- Provenance Tracking ---

	// GetPendingChanges returns pending (uncommitted) changes grouped by sandbox.
	GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error)

	// GetFileProvenance returns the history of changes for a specific file.
	GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error)

	// GetCommitPreview returns a preview of what would be committed.
	// This includes reconciliation with git status to detect externally-committed files.
	GetCommitPreview(ctx context.Context, req *types.CommitPreviewRequest) (*types.CommitPreviewResult, error)

	// CommitPending commits pending changes to git and updates provenance records.
	CommitPending(ctx context.Context, req *types.CommitPendingRequest) (*types.CommitPendingResult, error)
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
	// Check for idempotent request (safe retries)
	if existing, ok := s.checkIdempotency(ctx, req); ok {
		return existing, nil
	}

	// Validate and normalize the request
	projectRoot, normalizedPath, err := s.validateCreateRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Create and mount the sandbox
	return s.createAndMountSandbox(ctx, req, projectRoot, normalizedPath)
}

// checkIdempotency checks if a sandbox was already created with the given idempotency key.
// Returns (existing sandbox, true) if found, or (nil, false) if not.
func (s *Service) checkIdempotency(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, bool) {
	if req.IdempotencyKey == "" {
		return nil, false
	}

	existing, err := s.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil {
		// Log error but don't block - idempotency is a convenience, not required
		fmt.Printf("warning: failed to check idempotency key: %v\n", err)
		return nil, false
	}
	if existing != nil {
		// Found existing sandbox with this key - return it (idempotent success)
		return existing, true
	}
	return nil, false
}

// validateCreateRequest validates the create request and returns the resolved project root
// and normalized scope path. Returns an error if validation fails.
func (s *Service) validateCreateRequest(ctx context.Context, req *types.CreateRequest) (string, string, error) {
	// Resolve project root from request or config
	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		return "", "", types.NewValidationErrorWithHint(
			"projectRoot",
			"project root is required but not provided",
			"Set projectRoot in the request body, or configure PROJECT_ROOT environment variable",
		)
	}

	// Validate and normalize scope path
	normalizedPath, err := ValidateScopePath(req.ScopePath, projectRoot)
	if err != nil {
		return "", "", types.NewValidationErrorWithHint(
			"scopePath",
			fmt.Sprintf("invalid scope path: %v", err),
			"Ensure the path exists within the project root and contains no invalid characters",
		)
	}

	// Check for overlapping sandboxes
	conflicts, err := s.repo.CheckScopeOverlap(ctx, normalizedPath, projectRoot, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to check scope overlap: %w", err)
	}
	if len(conflicts) > 0 {
		return "", "", &types.ScopeConflictError{Conflicts: conflicts}
	}

	return projectRoot, normalizedPath, nil
}

// createAndMountSandbox creates the sandbox record, mounts the overlay, and returns the sandbox.
func (s *Service) createAndMountSandbox(ctx context.Context, req *types.CreateRequest, projectRoot, normalizedPath string) (*types.Sandbox, error) {
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

	// Capture the base commit hash for conflict detection (OT-P2-002)
	baseCommitHash, err := diff.GetGitCommitHash(ctx, projectRoot)
	if err != nil {
		// Log but don't fail - repo might not be a git repo
		s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
			"message": "failed to get base commit hash: " + err.Error(),
		})
	} else if baseCommitHash != "" {
		sandbox.BaseCommitHash = baseCommitHash
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

	// Log audit event [OT-P1-004]
	s.logAuditEvent(ctx, sandbox, "created", req.Owner, string(req.OwnerType), map[string]interface{}{
		"scopePath":      sandbox.ScopePath,
		"projectRoot":    sandbox.ProjectRoot,
		"idempotencyKey": req.IdempotencyKey,
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

	// Log audit event [OT-P1-004]
	s.logAuditEvent(ctx, sandbox, "stopped", "", "", nil)

	return sandbox, nil
}

// Start remounts a stopped sandbox to resume work.
//
// # Idempotency
//
// This operation is idempotent: calling Start on an already-active sandbox
// returns success with the current sandbox state. This enables safe retries
// without requiring callers to check the status first.
func (s *Service) Start(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// --- Idempotency Check ---
	// If already active, return success (no-op)
	// This makes Start safe to retry without error
	if sandbox.Status == types.StatusActive {
		return sandbox, nil
	}

	// Use explicit state transition check for other states
	if err := types.CanStart(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	// Remount the overlay
	paths, err := s.driver.Mount(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to remount sandbox: %w", err)
	}

	// Update with mount paths (they may have changed)
	sandbox.LowerDir = paths.LowerDir
	sandbox.UpperDir = paths.UpperDir
	sandbox.WorkDir = paths.WorkDir
	sandbox.MergedDir = paths.MergedDir
	sandbox.Status = types.StatusActive
	sandbox.StoppedAt = nil // Clear stopped timestamp
	sandbox.LastUsedAt = time.Now()

	if err := s.repo.Update(ctx, sandbox); err != nil {
		// Attempt to cleanup on failure
		s.driver.Unmount(ctx, sandbox)
		return nil, fmt.Errorf("failed to update sandbox: %w", err)
	}

	// Log audit event [OT-P1-004]
	s.logAuditEvent(ctx, sandbox, "started", "", "", nil)

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

	// Log audit event [OT-P1-004] - Note: sandbox is being deleted, capture final state
	s.logAuditEvent(ctx, sandbox, "deleted", "", "", nil)

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

	// Calculate and update sandbox size metrics from the changes
	var totalSizeBytes int64
	for _, change := range changes {
		// Only count added and modified files for size (deleted files don't take space)
		if change.ChangeType == types.ChangeTypeAdded || change.ChangeType == types.ChangeTypeModified {
			totalSizeBytes += change.FileSize
		}
	}

	// Update sandbox metrics if they've changed
	if sandbox.SizeBytes != totalSizeBytes || sandbox.FileCount != len(changes) {
		sandbox.SizeBytes = totalSizeBytes
		sandbox.FileCount = len(changes)
		// Best-effort update - don't fail diff generation on metrics update failure
		s.repo.Update(ctx, sandbox)
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

	// Get all changes - this is the total count before filtering
	allChanges, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes: %w", err)
	}

	// [OT-P2-002] Conflict Detection
	// Check if the canonical repo has changed since sandbox creation
	conflictCheck, err := diff.CheckForConflicts(ctx, sandbox, allChanges)
	if err != nil {
		// Log but don't fail - conflict check is advisory
		s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
			"message": "failed to check for conflicts: " + err.Error(),
		})
	}

	// If conflicts detected and not forced, return error with conflict info
	if conflictCheck != nil && conflictCheck.HasChanged && !req.Force {
		// If there are actual file conflicts, return an error
		if len(conflictCheck.ConflictingFiles) > 0 {
			return nil, types.NewRepoChangedErrorWithFiles(
				sandbox.ID.String(),
				conflictCheck.BaseCommitHash,
				conflictCheck.CurrentHash,
				conflictCheck.ConflictingFiles,
			)
		}

		// No file conflicts but repo changed - warn but continue
		// Include conflict info in result for visibility
		baseHash := conflictCheck.BaseCommitHash
		currentHash := conflictCheck.CurrentHash
		if len(baseHash) > 8 {
			baseHash = baseHash[:8]
		}
		if len(currentHash) > 8 {
			currentHash = currentHash[:8]
		}
		s.logAuditEvent(ctx, sandbox, "sandbox.info", "system", "system", map[string]interface{}{
			"message":     "repo changed since sandbox creation but no conflicting files",
			"baseHash":    baseHash,
			"currentHash": currentHash,
		})
	}

	totalChanges := len(allChanges)

	// [OT-P1-002] Track which changes will be applied (for partial approval)
	changes := allChanges

	// Filter changes if specific files requested
	if req.Mode == "files" && len(req.FileIDs) > 0 {
		changes = diff.FilterChanges(allChanges, req.FileIDs)
	}

	if len(changes) == 0 {
		return &types.ApprovalResult{
			Success:   true,
			Applied:   0,
			Remaining: totalChanges,
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

	// [OT-P1-001] Hunk-Level Approval
	// Filter to only selected hunks if hunks mode is requested
	if req.Mode == "hunks" && len(req.HunkRanges) > 0 {
		diffResult.UnifiedDiff = diff.FilterHunks(diffResult.UnifiedDiff, req.HunkRanges, changes)
		if diffResult.UnifiedDiff == "" {
			return &types.ApprovalResult{
				Success:   true,
				Applied:   0,
				AppliedAt: time.Now(),
			}, nil
		}
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

	// Extract file paths for selective staging (prevents git add -A bug)
	filePaths := make([]string, len(changes))
	for i, change := range changes {
		filePaths[i] = change.FilePath
	}

	// Apply the diff
	patcher := diff.NewPatcher()
	applyResult, err := patcher.ApplyDiff(ctx, sandbox.ProjectRoot, diffResult.UnifiedDiff, diff.ApplyOptions{
		CommitMsg:    commitMsg,
		Author:       author,
		CreateCommit: req.CreateCommit, // Only commit if explicitly requested
		FilePaths:    filePaths,        // For selective staging
	})
	if err != nil {
		return nil, fmt.Errorf("failed to apply diff: %w", err)
	}

	if !applyResult.Success {
		return &types.ApprovalResult{
			Success:   false,
			Failed:    len(changes),
			Remaining: totalChanges,
			ErrorMsg:  fmt.Sprintf("patch application failed: %v", applyResult.Errors),
			AppliedAt: time.Now(),
		}, nil
	}

	// Record provenance for all applied changes
	appliedChanges := make([]*types.AppliedChange, len(changes))
	for i, c := range changes {
		appliedChanges[i] = &types.AppliedChange{
			ID:               uuid.New(),
			SandboxID:        sandbox.ID,
			SandboxOwner:     sandbox.Owner,
			SandboxOwnerType: string(sandbox.OwnerType),
			FilePath:         filepath.Join(sandbox.ProjectRoot, c.FilePath),
			ProjectRoot:      sandbox.ProjectRoot,
			ChangeType:       string(c.ChangeType),
			FileSize:         c.FileSize,
		}
	}

	// Record in database (best-effort, don't fail on provenance error)
	if err := s.repo.RecordAppliedChanges(ctx, appliedChanges); err != nil {
		s.logAuditEvent(ctx, sandbox, "provenance.warning", "system", "system", map[string]interface{}{
			"message": "failed to record provenance: " + err.Error(),
		})
	}

	// If commit was created, mark provenance records as committed
	if applyResult.CommitHash != "" {
		ids := make([]uuid.UUID, len(appliedChanges))
		for i, c := range appliedChanges {
			ids[i] = c.ID
		}
		if err := s.repo.MarkChangesCommitted(ctx, ids, applyResult.CommitHash, commitMsg); err != nil {
			s.logAuditEvent(ctx, sandbox, "provenance.warning", "system", "system", map[string]interface{}{
				"message": "failed to mark changes committed: " + err.Error(),
			})
		}
	}

	// [OT-P1-002] Partial Approval Workflow
	// Determine if this is a partial or full approval
	remainingChanges := totalChanges - len(changes)
	isPartial := remainingChanges > 0

	now := time.Now()

	if isPartial {
		// Partial approval: clean up only the applied files from upper layer,
		// keep sandbox in current state for follow-up approvals
		for _, change := range changes {
			if err := s.driver.RemoveFromUpper(ctx, sandbox, change.FilePath); err != nil {
				// Log but don't fail - the changes were successfully applied
				// to the canonical repo, cleanup is best-effort
				s.logAuditEvent(ctx, sandbox, "partial_cleanup_warning", req.Actor, "", map[string]interface{}{
					"file":  change.FilePath,
					"error": err.Error(),
				})
			}
		}
		// Update last_used_at to track activity
		sandbox.LastUsedAt = now
		s.repo.Update(ctx, sandbox)

		// Log partial approval event
		s.logAuditEvent(ctx, sandbox, "partial_approved", req.Actor, "", map[string]interface{}{
			"filesApplied":   len(changes),
			"filesRemaining": remainingChanges,
			"commitHash":     applyResult.CommitHash,
			"mode":           req.Mode,
		})
	} else {
		// Full approval: transition to StatusApproved
		sandbox.Status = types.StatusApproved
		sandbox.ApprovedAt = &now
		s.repo.Update(ctx, sandbox)

		// Log full approval event [OT-P1-004]
		s.logAuditEvent(ctx, sandbox, "approved", req.Actor, "", map[string]interface{}{
			"filesApplied": len(changes),
			"commitHash":   applyResult.CommitHash,
			"mode":         req.Mode,
		})
	}

	return &types.ApprovalResult{
		Success:    true,
		Applied:    len(changes),
		Remaining:  remainingChanges,
		IsPartial:  isPartial,
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

	// Log audit event [OT-P1-004]
	s.logAuditEvent(ctx, sandbox, "rejected", actor, "", nil)

	return sandbox, nil
}

// Discard removes specific files from a sandbox without applying them.
// This allows rejecting individual files while keeping others pending for review.
func (s *Service) Discard(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error) {
	sandbox, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}

	// Can only discard from active or stopped sandboxes
	if sandbox.Status != types.StatusActive && sandbox.Status != types.StatusStopped {
		return nil, types.NewStateError(&types.InvalidTransitionError{
			Current: sandbox.Status,
			Reason:  fmt.Sprintf("cannot discard files from %s sandbox", sandbox.Status),
		})
	}

	// Get current changes to map file IDs to paths
	allChanges, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	// Build lookup maps
	idToPath := make(map[uuid.UUID]string)
	pathToChange := make(map[string]*types.FileChange)
	for _, change := range allChanges {
		idToPath[change.ID] = change.FilePath
		pathToChange[change.FilePath] = change
	}

	// Determine which files to discard
	var filesToDiscard []string

	// Process file IDs
	for _, fileID := range req.FileIDs {
		if path, ok := idToPath[fileID]; ok {
			filesToDiscard = append(filesToDiscard, path)
		}
	}

	// Process file paths (alternative input method)
	for _, path := range req.FilePaths {
		if _, ok := pathToChange[path]; ok {
			// Avoid duplicates
			found := false
			for _, existing := range filesToDiscard {
				if existing == path {
					found = true
					break
				}
			}
			if !found {
				filesToDiscard = append(filesToDiscard, path)
			}
		}
	}

	if len(filesToDiscard) == 0 {
		return &types.DiscardResult{
			Success:   true,
			Discarded: 0,
			Remaining: len(allChanges),
		}, nil
	}

	// Remove each file from the upper layer
	discardedCount := 0
	var discardedFiles []string
	for _, filePath := range filesToDiscard {
		if err := s.driver.RemoveFromUpper(ctx, sandbox, filePath); err != nil {
			// Log but continue - partial discard is acceptable
			s.logAuditEvent(ctx, sandbox, "discard_warning", req.Actor, "", map[string]interface{}{
				"file":  filePath,
				"error": err.Error(),
			})
			continue
		}
		discardedCount++
		discardedFiles = append(discardedFiles, filePath)
	}

	// Log audit event
	s.logAuditEvent(ctx, sandbox, "discarded", req.Actor, "", map[string]interface{}{
		"filesDiscarded": discardedCount,
		"files":          discardedFiles,
	})

	return &types.DiscardResult{
		Success:   true,
		Discarded: discardedCount,
		Remaining: len(allChanges) - discardedCount,
		Files:     discardedFiles,
	}, nil
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

// --- Retry/Rebase Workflow (OT-P2-003) ---

// CheckConflicts checks if the canonical repo has changed since sandbox creation
// and identifies any conflicting files.
//
// This operation is safe to call multiple times and doesn't modify the sandbox state.
// Use this to determine if a rebase is needed before approving changes.
func (s *Service) CheckConflicts(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	response := &types.ConflictCheckResponse{
		BaseCommitHash: sandbox.BaseCommitHash,
		CheckedAt:      time.Now(),
	}

	// Get sandbox changes
	sandboxChanges, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox changes: %w", err)
	}

	// Extract file paths for the response
	for _, change := range sandboxChanges {
		response.SandboxChangedFiles = append(response.SandboxChangedFiles, change.FilePath)
	}

	// If no base commit hash, we can't detect conflicts (non-git repo)
	if sandbox.BaseCommitHash == "" {
		return response, nil
	}

	// Use existing conflict detection
	conflictCheck, err := diff.CheckForConflicts(ctx, sandbox, sandboxChanges)
	if err != nil {
		return nil, fmt.Errorf("failed to check for conflicts: %w", err)
	}

	response.HasConflict = conflictCheck.HasChanged
	response.CurrentHash = conflictCheck.CurrentHash
	response.RepoChangedFiles = conflictCheck.RepoChangedFiles
	response.ConflictingFiles = conflictCheck.ConflictingFiles

	return response, nil
}

// Rebase updates the sandbox's BaseCommitHash to the current repo state.
//
// This operation updates the sandbox's reference point for conflict detection.
// After rebasing:
//   - Future conflict checks will compare against the new base commit
//   - The diff will be regenerated against the current canonical repo state
//   - If there were conflicts before, they may be resolved (or new ones detected)
//
// Note: This does NOT merge changes from the canonical repo into the sandbox.
// The sandbox's changes remain unchanged; only the baseline reference is updated.
func (s *Service) Rebase(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error) {
	sandbox, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}

	// Only allow rebase for active or stopped sandboxes
	if sandbox.Status != types.StatusActive && sandbox.Status != types.StatusStopped {
		return nil, types.NewStateError(&types.InvalidTransitionError{
			Current: sandbox.Status,
			Reason:  fmt.Sprintf("cannot rebase sandbox in %s status", sandbox.Status),
		})
	}

	result := &types.RebaseResult{
		PreviousBaseHash: sandbox.BaseCommitHash,
		Strategy:         req.Strategy,
		RebasedAt:        time.Now(),
	}

	// If strategy is empty, default to regenerate
	if req.Strategy == "" {
		req.Strategy = types.RebaseStrategyRegenerate
		result.Strategy = types.RebaseStrategyRegenerate
	}

	// Get the current commit hash
	newHash, err := diff.GetGitCommitHash(ctx, sandbox.ProjectRoot)
	if err != nil {
		result.Success = false
		result.ErrorMsg = fmt.Sprintf("failed to get current repo commit hash: %v", err)
		return result, nil
	}

	if newHash == "" {
		result.Success = false
		result.ErrorMsg = "canonical repo is not a git repository"
		return result, nil
	}

	result.NewBaseHash = newHash

	// Check what files have changed in the repo since sandbox creation
	if sandbox.BaseCommitHash != "" && sandbox.BaseCommitHash != newHash {
		repoChangedFiles, err := diff.GetChangedFilesSinceCommit(ctx, sandbox.ProjectRoot, sandbox.BaseCommitHash)
		if err != nil {
			// Log but don't fail - we can still update the hash
			s.logAuditEvent(ctx, sandbox, "rebase.warning", req.Actor, "", map[string]interface{}{
				"message": "failed to get repo changed files: " + err.Error(),
			})
		} else {
			result.RepoChangedFiles = repoChangedFiles

			// Find conflicting files (changed in both sandbox and repo)
			sandboxChanges, _ := s.driver.GetChangedFiles(ctx, sandbox)
			result.ConflictingFiles = diff.FindConflictingFiles(sandboxChanges, repoChangedFiles)
		}
	}

	// Update the sandbox's BaseCommitHash
	sandbox.BaseCommitHash = newHash
	sandbox.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, sandbox); err != nil {
		result.Success = false
		result.ErrorMsg = fmt.Sprintf("failed to update sandbox: %v", err)
		return result, nil
	}

	result.Success = true

	// Log audit event
	s.logAuditEvent(ctx, sandbox, "rebased", req.Actor, "", map[string]interface{}{
		"previousBaseHash": result.PreviousBaseHash,
		"newBaseHash":      result.NewBaseHash,
		"strategy":         string(result.Strategy),
		"repoChangedFiles": len(result.RepoChangedFiles),
		"conflictingFiles": len(result.ConflictingFiles),
	})

	return result, nil
}

// logAuditEvent creates and logs an audit event with full sandbox state.
// [OT-P1-004] Audit Trail Metadata
//
// This captures an immutable snapshot of the sandbox state at the time of the event,
// enabling full auditability and forensic analysis of sandbox lifecycle changes.
func (s *Service) logAuditEvent(ctx context.Context, sandbox *types.Sandbox, eventType, actor, actorType string, details map[string]interface{}) {
	// Build sandbox state snapshot
	sandboxState := map[string]interface{}{
		"id":          sandbox.ID.String(),
		"scopePath":   sandbox.ScopePath,
		"projectRoot": sandbox.ProjectRoot,
		"status":      string(sandbox.Status),
		"owner":       sandbox.Owner,
		"ownerType":   string(sandbox.OwnerType),
		"sizeBytes":   sandbox.SizeBytes,
		"fileCount":   sandbox.FileCount,
		"driver":      sandbox.Driver,
		"createdAt":   sandbox.CreatedAt.Format(time.RFC3339),
	}

	// Add optional timestamps
	if sandbox.StoppedAt != nil {
		sandboxState["stoppedAt"] = sandbox.StoppedAt.Format(time.RFC3339)
	}
	if sandbox.ApprovedAt != nil {
		sandboxState["approvedAt"] = sandbox.ApprovedAt.Format(time.RFC3339)
	}
	if sandbox.DeletedAt != nil {
		sandboxState["deletedAt"] = sandbox.DeletedAt.Format(time.RFC3339)
	}

	// Add error message if present
	if sandbox.ErrorMsg != "" {
		sandboxState["errorMessage"] = sandbox.ErrorMsg
	}

	// Determine actor type if not provided
	if actorType == "" && actor != "" {
		actorType = "user" // Default
	}

	event := &types.AuditEvent{
		SandboxID:    &sandbox.ID,
		EventType:    eventType,
		Actor:        actor,
		ActorType:    actorType,
		Details:      details,
		SandboxState: sandboxState,
	}

	// Log to database (fire-and-forget, don't block on audit log failures)
	if err := s.repo.LogAuditEvent(ctx, event); err != nil {
		// Log the error but don't fail the operation
		fmt.Printf("warning: failed to log audit event: %v\n", err)
	}
}

// ValidatePath checks if a path is valid for use as a sandbox scope.
//
// This method centralizes all path validation logic that was previously scattered
// in the handler. It checks:
//   - Path is absolute
//   - Path is not a dangerous system directory
//   - Path exists on the filesystem
//   - Path is a directory (not a file)
//   - Path is within the project root
//
// The result always includes the path and projectRoot for echo-back, and sets
// Valid=true only if all checks pass.
func (s *Service) ValidatePath(ctx context.Context, path, projectRoot string) (*types.PathValidationResult, error) {
	result := &types.PathValidationResult{
		Path:        path,
		ProjectRoot: projectRoot,
	}

	// Use default project root if not specified
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
		result.ProjectRoot = projectRoot
	}

	// Check if path is absolute
	if !filepath.IsAbs(path) {
		result.Valid = false
		result.Error = "Path must be absolute"
		return result, nil
	}

	// Check for dangerous system paths
	cleanPath := filepath.Clean(path)
	dangerousPaths := []string{"/", "/bin", "/sbin", "/usr", "/etc", "/var", "/tmp", "/root", "/home"}
	for _, dangerous := range dangerousPaths {
		if cleanPath == dangerous {
			result.Valid = false
			result.Error = "Cannot use system directories"
			return result, nil
		}
	}

	// Check if path exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		result.Valid = false
		result.Exists = false
		result.Error = "Path does not exist"
		return result, nil
	}
	if err != nil {
		result.Valid = false
		result.Error = fmt.Sprintf("Cannot access path: %v", err)
		return result, nil
	}

	result.Exists = true
	result.IsDirectory = info.IsDir()

	// Check if it's a directory
	if !info.IsDir() {
		result.Valid = false
		result.Error = "Path must be a directory"
		return result, nil
	}

	// Check if path is within project root (if project root is set)
	if projectRoot != "" {
		cleanProjectRoot := filepath.Clean(projectRoot)
		if cleanPath != cleanProjectRoot && !strings.HasPrefix(cleanPath, cleanProjectRoot+string(filepath.Separator)) {
			result.Valid = false
			result.Error = fmt.Sprintf("Path must be within %s", projectRoot)
			result.WithinProjectRoot = false
			return result, nil
		}
		result.WithinProjectRoot = true
	}

	result.Valid = true
	return result, nil
}

// Legacy error type aliases for backwards compatibility.
// New code should use types.ScopeConflictError and types.NotFoundError directly.

// ScopeConflictError is an alias for types.ScopeConflictError.
// Deprecated: Use types.ScopeConflictError directly.
type ScopeConflictError = types.ScopeConflictError

// NotFoundError is an alias for types.NotFoundError.
// Deprecated: Use types.NotFoundError directly.
type NotFoundError = types.NotFoundError

// --- Provenance Tracking ---

// GetPendingChanges returns pending (uncommitted) changes grouped by sandbox.
func (s *Service) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	return s.repo.GetPendingChanges(ctx, projectRoot, limit, offset)
}

// GetFileProvenance returns the history of changes for a specific file.
func (s *Service) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	return s.repo.GetFileProvenance(ctx, filePath, projectRoot, limit)
}

// CommitPending commits pending changes to git and updates provenance records.
// This allows batching multiple sandbox changes into a single commit.
//
// Reconciliation behavior:
// - Files that are still uncommitted in git will be staged and committed
// - Files that were already committed externally will be marked as reconciled
//   with a special commit hash indicating external commit
func (s *Service) CommitPending(ctx context.Context, req *types.CommitPendingRequest) (*types.CommitPendingResult, error) {
	result := &types.CommitPendingResult{}

	// Determine project root
	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		result.ErrorMsg = "project root is required"
		return result, nil
	}

	// Get all pending change files
	pendingChanges, err := s.repo.GetPendingChangeFiles(ctx, projectRoot, req.SandboxIDs)
	if err != nil {
		result.ErrorMsg = fmt.Sprintf("failed to get pending changes: %v", err)
		return result, nil
	}

	if len(pendingChanges) == 0 {
		result.Success = true
		result.FilesCommitted = 0
		return result, nil
	}

	// Build mapping of relative path -> change record
	pathToChange := make(map[string]*types.AppliedChange)
	relPaths := make([]string, 0, len(pendingChanges))
	for _, change := range pendingChanges {
		relPath := change.FilePath
		if strings.HasPrefix(relPath, projectRoot) {
			relPath = strings.TrimPrefix(relPath, projectRoot)
			relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		}
		if relPath != "" {
			relPaths = append(relPaths, relPath)
			pathToChange[relPath] = change
		}
	}

	if len(relPaths) == 0 {
		result.Success = true
		result.FilesCommitted = 0
		return result, nil
	}

	// Reconcile with git status to find which files are actually uncommitted
	reconciled, err := diff.ReconcilePendingWithGit(ctx, projectRoot, relPaths)
	if err != nil {
		// If reconciliation fails, fall back to committing all files
		fmt.Printf("warning: reconciliation failed, proceeding without: %v\n", err)
		reconciled = &diff.ReconcileResult{StillPending: relPaths}
	}

	// Mark externally-committed files in DB (with special marker)
	if len(reconciled.AlreadyCommitted) > 0 {
		var externallyCommittedIDs []uuid.UUID
		for _, path := range reconciled.AlreadyCommitted {
			if change, ok := pathToChange[path]; ok {
				externallyCommittedIDs = append(externallyCommittedIDs, change.ID)
			}
		}
		if len(externallyCommittedIDs) > 0 {
			// Mark as "externally committed" - use a special marker
			if err := s.repo.MarkChangesCommitted(ctx, externallyCommittedIDs, "EXTERNAL", "Committed externally (reconciled)"); err != nil {
				fmt.Printf("warning: failed to mark externally committed: %v\n", err)
			}
		}
	}

	// If no files are actually uncommitted, we're done
	if len(reconciled.StillPending) == 0 {
		result.Success = true
		result.FilesCommitted = 0
		return result, nil
	}

	// Generate commit message if not provided
	commitMsg := req.CommitMessage
	if commitMsg == "" {
		// Use a more descriptive default message
		commitMsg = s.generateDefaultCommitMessage(reconciled.StillPending, pathToChange)
	}

	// Create the commit with only the files that are actually uncommitted
	patcher := diff.NewPatcher()
	commitHash, err := patcher.CreateCommitFromFiles(ctx, projectRoot, diff.ApplyOptions{
		CommitMsg:    commitMsg,
		Author:       req.Actor,
		CreateCommit: true,
		FilePaths:    reconciled.StillPending,
	})
	if err != nil {
		result.ErrorMsg = fmt.Sprintf("failed to create commit: %v", err)
		return result, nil
	}

	// Mark committed files in DB
	var committedIDs []uuid.UUID
	for _, path := range reconciled.StillPending {
		if change, ok := pathToChange[path]; ok {
			committedIDs = append(committedIDs, change.ID)
		}
	}

	if len(committedIDs) > 0 {
		if err := s.repo.MarkChangesCommitted(ctx, committedIDs, commitHash, commitMsg); err != nil {
			fmt.Printf("warning: failed to mark changes committed: %v\n", err)
		}
	}

	result.Success = true
	result.FilesCommitted = len(reconciled.StillPending)
	result.CommitHash = commitHash

	return result, nil
}

// generateDefaultCommitMessage creates a descriptive commit message.
func (s *Service) generateDefaultCommitMessage(paths []string, pathToChange map[string]*types.AppliedChange) string {
	if len(paths) == 0 {
		return "No changes"
	}

	// Count by type and collect unique owners
	var added, modified, deleted int
	owners := make(map[string]bool)

	for _, path := range paths {
		if change, ok := pathToChange[path]; ok {
			switch change.ChangeType {
			case "added":
				added++
			case "modified":
				modified++
			case "deleted":
				deleted++
			}
			if change.SandboxOwner != "" {
				owners[change.SandboxOwner] = true
			}
		}
	}

	// Build message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Apply %d sandbox changes", len(paths)))

	// Add owner attribution
	ownerList := make([]string, 0, len(owners))
	for owner := range owners {
		ownerList = append(ownerList, owner)
	}
	if len(ownerList) == 1 {
		msg.WriteString(fmt.Sprintf(" from %s", ownerList[0]))
	} else if len(ownerList) > 1 && len(ownerList) <= 3 {
		msg.WriteString(fmt.Sprintf(" from %s", strings.Join(ownerList, ", ")))
	}

	// Add breakdown
	msg.WriteString("\n\n")
	if added > 0 {
		msg.WriteString(fmt.Sprintf("- %d added\n", added))
	}
	if modified > 0 {
		msg.WriteString(fmt.Sprintf("- %d modified\n", modified))
	}
	if deleted > 0 {
		msg.WriteString(fmt.Sprintf("- %d deleted\n", deleted))
	}

	return msg.String()
}

// GetCommitPreview returns a preview of what would be committed.
// This includes reconciliation with git status to detect externally-committed files.
func (s *Service) GetCommitPreview(ctx context.Context, req *types.CommitPreviewRequest) (*types.CommitPreviewResult, error) {
	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		return nil, fmt.Errorf("project root is required")
	}

	// Get all pending change files from database
	pendingChanges, err := s.repo.GetPendingChangeFiles(ctx, projectRoot, req.SandboxIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending changes: %w", err)
	}

	result := &types.CommitPreviewResult{
		Files:            make([]types.CommitPreviewFile, 0, len(pendingChanges)),
		GroupedBySandbox: []types.CommitPreviewSandboxGroup{},
	}

	if len(pendingChanges) == 0 {
		result.SuggestedMessage = "No pending changes"
		return result, nil
	}

	// Extract relative paths for reconciliation
	relPaths := make([]string, 0, len(pendingChanges))
	for _, change := range pendingChanges {
		relPath := change.FilePath
		if strings.HasPrefix(relPath, projectRoot) {
			relPath = strings.TrimPrefix(relPath, projectRoot)
			relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		}
		if relPath != "" {
			relPaths = append(relPaths, relPath)
		}
	}

	// Reconcile with git status
	reconciled, err := diff.ReconcilePendingWithGit(ctx, projectRoot, relPaths)
	if err != nil {
		// Log but don't fail - we can still show the pending files
		fmt.Printf("warning: failed to reconcile with git: %v\n", err)
		reconciled = &diff.ReconcileResult{StillPending: relPaths}
	}

	// Build sets for quick lookup
	stillPendingSet := make(map[string]bool)
	for _, p := range reconciled.StillPending {
		stillPendingSet[p] = true
	}

	// Group by sandbox for summary
	sandboxGroups := make(map[uuid.UUID]*types.CommitPreviewSandboxGroup)

	for _, change := range pendingChanges {
		relPath := change.FilePath
		if strings.HasPrefix(relPath, projectRoot) {
			relPath = strings.TrimPrefix(relPath, projectRoot)
			relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		}

		status := "already_committed"
		if stillPendingSet[relPath] {
			status = "pending"
			result.CommittableFiles++
		} else {
			result.AlreadyCommittedFiles++
		}

		file := types.CommitPreviewFile{
			FilePath:     change.FilePath,
			RelativePath: relPath,
			ChangeType:   change.ChangeType,
			SandboxID:    change.SandboxID,
			SandboxOwner: change.SandboxOwner,
			AppliedAt:    change.AppliedAt,
			Status:       status,
		}
		result.Files = append(result.Files, file)

		// Update sandbox group
		group, exists := sandboxGroups[change.SandboxID]
		if !exists {
			group = &types.CommitPreviewSandboxGroup{
				SandboxID:    change.SandboxID,
				SandboxOwner: change.SandboxOwner,
			}
			sandboxGroups[change.SandboxID] = group
		}
		if status == "pending" {
			group.FileCount++
			switch change.ChangeType {
			case "added":
				group.Added++
			case "modified":
				group.Modified++
			case "deleted":
				group.Deleted++
			}
		}
	}

	// Convert sandbox groups map to slice
	for _, group := range sandboxGroups {
		if group.FileCount > 0 {
			result.GroupedBySandbox = append(result.GroupedBySandbox, *group)
		}
	}

	// Generate suggested commit message
	result.SuggestedMessage = s.generateCommitMessage(result)

	return result, nil
}

// generateCommitMessage creates a descriptive commit message from the preview.
func (s *Service) generateCommitMessage(preview *types.CommitPreviewResult) string {
	if preview.CommittableFiles == 0 {
		return "No uncommitted changes to apply"
	}

	var msg strings.Builder

	// Count totals across all sandboxes
	var totalAdded, totalModified, totalDeleted int
	owners := make([]string, 0)
	seenOwners := make(map[string]bool)

	for _, group := range preview.GroupedBySandbox {
		totalAdded += group.Added
		totalModified += group.Modified
		totalDeleted += group.Deleted
		if !seenOwners[group.SandboxOwner] {
			seenOwners[group.SandboxOwner] = true
			owners = append(owners, group.SandboxOwner)
		}
	}

	// Write summary line
	msg.WriteString(fmt.Sprintf("Apply %d sandbox changes", preview.CommittableFiles))

	// Add attribution if from specific owners
	if len(owners) == 1 {
		msg.WriteString(fmt.Sprintf(" from %s", owners[0]))
	} else if len(owners) <= 3 {
		msg.WriteString(fmt.Sprintf(" from %s", strings.Join(owners, ", ")))
	} else {
		msg.WriteString(fmt.Sprintf(" from %d sandboxes", len(preview.GroupedBySandbox)))
	}
	msg.WriteString("\n")

	// Add breakdown
	msg.WriteString("\n")
	if totalAdded > 0 {
		msg.WriteString(fmt.Sprintf("- %d files added\n", totalAdded))
	}
	if totalModified > 0 {
		msg.WriteString(fmt.Sprintf("- %d files modified\n", totalModified))
	}
	if totalDeleted > 0 {
		msg.WriteString(fmt.Sprintf("- %d files deleted\n", totalDeleted))
	}

	// Add file list if not too many
	if preview.CommittableFiles <= 10 {
		msg.WriteString("\nFiles:\n")
		for _, file := range preview.Files {
			if file.Status == "pending" {
				prefix := " "
				switch file.ChangeType {
				case "added":
					prefix = "+"
				case "modified":
					prefix = "M"
				case "deleted":
					prefix = "-"
				}
				msg.WriteString(fmt.Sprintf("  %s %s\n", prefix, file.RelativePath))
			}
		}
	}

	return msg.String()
}
