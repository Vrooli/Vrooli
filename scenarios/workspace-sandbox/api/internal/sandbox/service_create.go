package sandbox

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// service_create.go: sandbox creation path.
//
// Create is the entry point; it composes idempotency checks, request
// validation, and the actual mount + repo insert. The split is by
// concern: each helper does one thing and returns early.

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
// # Errors
//
// Returns ValidationError for invalid input, ScopeConflictError if the scope
// overlaps with an existing sandbox, or DriverError if mounting fails.
func (s *Service) Create(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
	if existing, ok := s.checkIdempotency(ctx, req); ok {
		return existing, nil
	}

	s.resolveNoLock(req)

	projectRoot, normalizedScopePath, normalizedReservedPaths, err := s.validateCreateRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	behavior := normalizeBehavior(req.Behavior)
	if err := validateBehavior(behavior); err != nil {
		return nil, err
	}
	req.Behavior = behavior

	return s.createAndMountSandbox(ctx, req, projectRoot, normalizedScopePath, normalizedReservedPaths)
}

// checkIdempotency checks if a sandbox was already created with the
// given idempotency key. Returns (existing sandbox, true) if found,
// or (nil, false) if not.
func (s *Service) checkIdempotency(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, bool) {
	if req.IdempotencyKey == "" {
		return nil, false
	}

	existing, err := s.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil {
		fmt.Printf("warning: failed to check idempotency key: %v\n", err)
		return nil, false
	}
	if existing != nil {
		return existing, true
	}
	return nil, false
}

// validateCreateRequest validates the create request and returns the
// resolved project root, normalized scope path (mount scope), and
// normalized reserved paths (mutual exclusion). Returns an error if
// validation fails.
func (s *Service) validateCreateRequest(ctx context.Context, req *types.CreateRequest) (string, string, []string, error) {
	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		return "", "", nil, types.NewValidationErrorWithHint(
			"projectRoot",
			"project root is required but not provided",
			"Set projectRoot in the request body, or configure PROJECT_ROOT environment variable",
		)
	}

	normalizedScopePath, err := ValidateScopePath(req.ScopePath, projectRoot)
	if err != nil {
		return "", "", nil, types.NewValidationErrorWithHint(
			"scopePath",
			fmt.Sprintf("invalid scope path: %v", err),
			"Ensure the path exists within the project root and contains no invalid characters",
		)
	}

	if *req.NoLock {
		return projectRoot, normalizedScopePath, []string{}, nil
	}

	rawReservedPaths := req.ReservedPaths
	if len(rawReservedPaths) == 0 {
		if strings.TrimSpace(req.ReservedPath) != "" {
			rawReservedPaths = []string{req.ReservedPath}
		} else {
			rawReservedPaths = []string{normalizedScopePath}
		}
	}

	normalizedReservedPaths := make([]string, 0, len(rawReservedPaths))
	fieldName := "reservedPaths"
	if len(req.ReservedPaths) == 0 {
		fieldName = "reservedPath"
	}

	cleanScope := filepath.Clean(normalizedScopePath)

	// Normalize, validate, dedupe, and prune redundant reserved paths.
	// Rules:
	// - Each reserved path must be within project root and within the mount scope.
	// - Descendants of an already-reserved prefix are redundant and are skipped.
	// - If a new prefix is an ancestor of existing reserved prefixes, it replaces them.
	for _, raw := range rawReservedPaths {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		normalized, err := ValidateScopePath(raw, projectRoot)
		if err != nil {
			return "", "", nil, types.NewValidationErrorWithHint(
				fieldName,
				fmt.Sprintf("invalid reserved path: %v", err),
				"Ensure reserved paths exist within the project root and contain no invalid characters",
			)
		}

		cleanReserved := filepath.Clean(normalized)
		if cleanReserved != cleanScope && !strings.HasPrefix(cleanReserved, cleanScope+string(filepath.Separator)) {
			return "", "", nil, types.NewValidationErrorWithHint(
				fieldName,
				"reserved path must be within scope path",
				"Set scopePath to the project root (full-repo mount), or choose reservedPaths inside the scopePath",
			)
		}

		redundant := false
		for _, kept := range normalizedReservedPaths {
			cleanKept := filepath.Clean(kept)
			if cleanReserved == cleanKept || strings.HasPrefix(cleanReserved, cleanKept+string(filepath.Separator)) {
				redundant = true
				break
			}
		}
		if redundant {
			continue
		}

		pruned := normalizedReservedPaths[:0]
		for _, kept := range normalizedReservedPaths {
			cleanKept := filepath.Clean(kept)
			if cleanKept == cleanReserved || strings.HasPrefix(cleanKept, cleanReserved+string(filepath.Separator)) {
				continue
			}
			pruned = append(pruned, kept)
		}
		normalizedReservedPaths = pruned

		normalizedReservedPaths = append(normalizedReservedPaths, normalized)
	}

	if len(normalizedReservedPaths) == 0 {
		normalizedReservedPaths = []string{normalizedScopePath}
	}

	var allConflicts []types.PathConflict
	for _, reservedAbs := range normalizedReservedPaths {
		conflicts, err := s.repo.CheckScopeOverlap(ctx, reservedAbs, projectRoot, nil)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to check scope overlap: %w", err)
		}
		if len(conflicts) > 0 {
			allConflicts = append(allConflicts, conflicts...)
		}
	}
	if len(allConflicts) > 0 {
		return "", "", nil, &types.ScopeConflictError{Conflicts: allConflicts}
	}

	return projectRoot, normalizedScopePath, normalizedReservedPaths, nil
}

// createAndMountSandbox builds the sandbox record, persists it, mounts
// the overlay, and updates the record with the resulting paths.
func (s *Service) createAndMountSandbox(ctx context.Context, req *types.CreateRequest, projectRoot, normalizedScopePath string, normalizedReservedPaths []string) (*types.Sandbox, error) {
	primaryReserved := ""
	if !*req.NoLock {
		primaryReserved = normalizedScopePath
		if len(normalizedReservedPaths) > 0 {
			primaryReserved = normalizedReservedPaths[0]
		}
	} else {
		normalizedReservedPaths = []string{}
	}
	sandbox := &types.Sandbox{
		ID:             uuid.New(),
		Name:           req.Name,
		ScopePath:      normalizedScopePath,
		ReservedPath:   primaryReserved,
		ReservedPaths:  normalizedReservedPaths,
		NoLock:         *req.NoLock,
		ProjectRoot:    projectRoot,
		AuxiliaryRoots: append([]string(nil), req.AuxiliaryRoots...),
		Owner:          req.Owner,
		OwnerType:      req.OwnerType,
		Status:         types.StatusCreating,
		DriverID:       string(s.driver.ID()),
		DriverVersion:  s.driver.Version(),
		Tags:           req.Tags,
		Metadata:       req.Metadata,
		Behavior:       req.Behavior,
		IdempotencyKey: req.IdempotencyKey,
	}

	if sandbox.OwnerType == "" {
		sandbox.OwnerType = types.OwnerTypeUser
	}

	if err := s.repo.Create(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to create sandbox record: %w", err)
	}

	// Capture the base commit hash for conflict detection (OT-P2-002)
	baseCommitHash, err := s.gitOps.GetCommitHash(ctx, projectRoot)
	if err != nil {
		s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
			"message": "failed to get base commit hash: " + err.Error(),
		})
	} else if baseCommitHash != "" {
		sandbox.BaseCommitHash = baseCommitHash
	}

	paths, err := s.driver.Mount(ctx, sandbox)
	if err != nil {
		sandbox.Status = types.StatusError
		sandbox.ErrorMsg = err.Error()
		if updateErr := s.repo.Update(ctx, sandbox); updateErr != nil {
			fmt.Printf("warning: failed to update sandbox status after mount failure: %v\n", updateErr)
		}
		return sandbox, fmt.Errorf("failed to mount sandbox: %w", err)
	}

	sandbox.LowerDir = paths.LowerDir
	sandbox.UpperDir = paths.UpperDir
	sandbox.WorkDir = paths.WorkDir
	sandbox.MergedDir = paths.MergedDir
	// Home-overlay paths are transient (not persisted). Drivers populate
	// them when they bring up a per-sandbox $HOME overlay; bwrap reads
	// HomeMergedDir to bind it at /home/<user> inside the namespace.
	sandbox.HomeLowerDir = paths.HomeLowerDir
	sandbox.HomeUpperDir = paths.HomeUpperDir
	sandbox.HomeWorkDir = paths.HomeWorkDir
	sandbox.HomeMergedDir = paths.HomeMergedDir
	sandbox.Status = types.StatusActive

	if err := s.repo.Update(ctx, sandbox); err != nil {
		if cleanupErr := s.driver.Cleanup(ctx, sandbox); cleanupErr != nil {
			fmt.Printf("warning: driver cleanup failed: %v\n", cleanupErr)
		}
		return nil, fmt.Errorf("failed to update sandbox: %w", err)
	}

	s.logAuditEvent(ctx, sandbox, "created", req.Owner, string(req.OwnerType), map[string]interface{}{
		"scopePath":      sandbox.ScopePath,
		"reservedPath":   sandbox.ReservedPath,
		"reservedPaths":  sandbox.ReservedPaths,
		"projectRoot":    sandbox.ProjectRoot,
		"idempotencyKey": req.IdempotencyKey,
	})

	return sandbox, nil
}
