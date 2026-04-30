package sandbox

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// Discard removes specific files from a sandbox without applying them.
// Allows rejecting individual files while keeping others pending for
// review. Lives here (not in service_review.go) because it does not
// drive a state transition — it's an in-place file mutation.
func (s *Service) Discard(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error) {
	sandbox, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}

	if sandbox.Status != types.StatusActive && sandbox.Status != types.StatusStopped {
		return nil, types.NewStateError(&types.InvalidTransitionError{
			Current: sandbox.Status,
			Reason:  fmt.Sprintf("cannot discard files from %s sandbox", sandbox.Status),
		})
	}

	allChanges, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	idToPath := make(map[uuid.UUID]string)
	pathToChange := make(map[string]*types.FileChange)
	for _, change := range allChanges {
		idToPath[change.ID] = change.FilePath
		pathToChange[change.FilePath] = change
	}

	var filesToDiscard []string

	for _, fileID := range req.FileIDs {
		if p, ok := idToPath[fileID]; ok {
			filesToDiscard = append(filesToDiscard, p)
		}
	}

	for _, p := range req.FilePaths {
		if _, ok := pathToChange[p]; ok {
			found := false
			for _, existing := range filesToDiscard {
				if existing == p {
					found = true
					break
				}
			}
			if !found {
				filesToDiscard = append(filesToDiscard, p)
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

	discardedCount := 0
	var discardedFiles []string
	for _, filePath := range filesToDiscard {
		if err := s.driver.RemoveFromUpper(ctx, sandbox, filePath); err != nil {
			s.logAuditEvent(ctx, sandbox, "discard_warning", req.Actor, "", map[string]interface{}{
				"file":  filePath,
				"error": err.Error(),
			})
			continue
		}
		discardedCount++
		discardedFiles = append(discardedFiles, filePath)
	}

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

// service_lifecycle.go: lifecycle CRUD on existing sandboxes (Get,
// List, Stop, Start, Delete, GetWorkspacePath) plus the helpers that
// drive automatic transitions from terminal states (lifecycle config).
//
// Behavior normalization + validation (normalizeBehavior, validateBehavior)
// also live here because they are inputs to lifecycle transitions.

// Get retrieves a sandbox by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	sandbox, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox: %w", err)
	}
	if sandbox == nil {
		return nil, types.NewNotFoundError(id.String())
	}
	// Re-derive transient mount paths (notably HomeMergedDir) on every
	// read so process-spawn handlers see them. The home overlay isn't
	// persisted in the DB; without this, every Service.Get would lose
	// HomeMergedDir and bwrap would skip the home bind for the second
	// process onward.
	s.inferMountPaths(sandbox)
	return sandbox, nil
}

// List retrieves sandboxes matching the filter.
func (s *Service) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	return s.repo.List(ctx, filter)
}

// Stop unmounts a sandbox but preserves its data.
//
// Idempotent: calling Stop on an already-stopped sandbox returns
// success with the current sandbox state.
func (s *Service) Stop(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if sandbox.Status == types.StatusStopped {
		return sandbox, nil
	}

	if err := types.CanStop(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	// Run pre-teardown hooks before unmounting, so external systems can
	// evacuate processes from the merged directory while it's still accessible.
	s.runPreTeardownHooks(ctx, sandbox, "stop")

	if err := s.driver.Unmount(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to unmount sandbox: %w", err)
	}

	now := s.clock.Now()
	sandbox.Status = types.StatusStopped
	sandbox.StoppedAt = &now

	if err := s.repo.Update(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to update sandbox: %w", err)
	}

	s.logAuditEvent(ctx, sandbox, "stopped", "", "", nil)

	return sandbox, nil
}

// Start remounts a stopped sandbox to resume work.
//
// Idempotent: calling Start on an already-active sandbox returns
// success with the current sandbox state.
func (s *Service) Start(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if sandbox.Status == types.StatusActive {
		return sandbox, nil
	}

	if err := types.CanStart(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	paths, err := s.driver.Mount(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to remount sandbox: %w", err)
	}

	sandbox.LowerDir = paths.LowerDir
	sandbox.UpperDir = paths.UpperDir
	sandbox.WorkDir = paths.WorkDir
	sandbox.MergedDir = paths.MergedDir
	// Home overlay (see docstring on Sandbox.HomeMergedDir).
	sandbox.HomeLowerDir = paths.HomeLowerDir
	sandbox.HomeUpperDir = paths.HomeUpperDir
	sandbox.HomeWorkDir = paths.HomeWorkDir
	sandbox.HomeMergedDir = paths.HomeMergedDir
	sandbox.Status = types.StatusActive
	sandbox.StoppedAt = nil
	sandbox.LastUsedAt = s.clock.Now()

	if err := s.repo.Update(ctx, sandbox); err != nil {
		if unmountErr := s.driver.Unmount(ctx, sandbox); unmountErr != nil {
			fmt.Printf("warning: driver unmount failed: %v\n", unmountErr)
		}
		return nil, fmt.Errorf("failed to update sandbox: %w", err)
	}

	s.logAuditEvent(ctx, sandbox, "started", "", "", nil)

	return sandbox, nil
}

// Delete removes a sandbox and all its data.
//
// Idempotent: calling Delete on an already-deleted sandbox returns
// success without error.
//
// Snapshot policy: Delete captures a diff archive only when no archive
// already exists for this sandbox. The archive's content is captured
// when the sandbox can still produce a diff (Active/Stopped); otherwise
// it is recorded with archive_state="not_captured" so the History UI
// can render an explicit "no diff captured" state. When called via the
// auto-Delete-on-Approve/Reject lifecycle path, the archive was already
// inserted by Approve/Reject — Delete leaves it untouched and only
// flips the sandbox status to Deleted (the archive remains keyed at the
// terminal status it captured).
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		if _, ok := err.(*types.NotFoundError); ok {
			return nil
		}
		return err
	}

	if sandbox.Status == types.StatusDeleted {
		return nil
	}

	// Snapshot before teardown if we're deleting a sandbox that doesn't
	// already have an archive. CanGenerateDiff captures the Active/
	// Stopped/etc. policy; Error sandboxes and missing-overlay paths get
	// a not_captured row.
	needsSnapshot := true
	if s.archiveRepo != nil {
		existing, getErr := s.archiveRepo.Get(ctx, id)
		if getErr != nil {
			return fmt.Errorf("failed to check existing archive: %w", getErr)
		}
		if existing != nil {
			needsSnapshot = false
		}
	}

	deletedAt := s.clock.Now()
	if needsSnapshot {
		// Error sandboxes get a not_captured archive row even though
		// CanGenerateDiff would technically allow a snapshot — the
		// overlay is typically broken in ways that produce misleading
		// diffs. Creating sandboxes have no upper dir at all. Both
		// surface as "no diff captured" in the UI.
		captured := types.CanGenerateDiff(sandbox.Status) == nil &&
			sandbox.Status != types.StatusError
		if err := s.snapshotAndTransition(ctx, sandbox, types.StatusDeleted, captured, func(sb *types.Sandbox) {
			sb.DeletedAt = &deletedAt
		}); err != nil {
			s.logAuditEvent(ctx, sandbox, "snapshot_failed", "", "system", map[string]interface{}{
				"phase": "delete",
				"error": err.Error(),
			})
			return fmt.Errorf("failed to snapshot+delete sandbox: %w", err)
		}
	} else {
		// Archive already exists from a prior Approve/Reject; this
		// Delete is the lifecycle cleanup that follows. Just flip the
		// sandbox status to Deleted (no re-snapshot, no archive
		// mutation — the archive is immutable post-creation).
		sandbox.Status = types.StatusDeleted
		sandbox.DeletedAt = &deletedAt
		if err := s.repo.Update(ctx, sandbox); err != nil {
			return fmt.Errorf("failed to update sandbox to deleted: %w", err)
		}
	}

	// Run pre-teardown hooks before cleanup, so external systems can
	// evacuate processes from the merged directory before it's removed.
	s.runPreTeardownHooks(ctx, sandbox, "delete")

	if err := s.driver.Cleanup(ctx, sandbox); err != nil {
		fmt.Printf("warning: driver cleanup failed: %v\n", err)
	}

	// I-MOUNT-1: own the daemon teardown deterministically. The driver
	// Cleanup unmounts; this kills any fuse-overlayfs daemon left behind
	// (e.g. the userspace daemon that survives Unmount on some kernels).
	// Background reaper stays as a safety net for API-crash paths.
	s.killDaemonsForSandbox(ctx, id)

	s.logAuditEvent(ctx, sandbox, "deleted", "", "", nil)

	return nil
}

// GetWorkspacePath returns the path where sandbox operations should occur.
func (s *Service) GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}

	if err := types.CanGetWorkspacePath(sandbox.Status); err != nil {
		return "", err
	}

	return sandbox.MergedDir, nil
}

// runPreTeardownHooks calls the configured teardown policy before
// sandbox unmount/delete. Failures are best-effort; teardown must
// never be blocked.
func (s *Service) runPreTeardownHooks(ctx context.Context, sandbox *types.Sandbox, reason string) {
	if s.teardownPolicy == nil {
		return
	}
	results := s.teardownPolicy.RunPreTeardownHooks(ctx, sandbox, reason)
	for _, r := range results {
		if !r.Success {
			fmt.Printf("warning: pre-teardown hook '%s' failed for sandbox %s: %v (output: %s)\n",
				r.HookName, sandbox.ID, r.Error, r.Output)
		}
	}
}

// applyLifecycleOnTerminal triggers automatic deletion when the
// sandbox transitions to a terminal status (Approved/Rejected) and
// the lifecycle config opts into deleteOn.
func (s *Service) applyLifecycleOnTerminal(ctx context.Context, sandbox *types.Sandbox, status types.Status) {
	if sandbox == nil {
		return
	}
	behavior := normalizeBehavior(sandbox.Behavior)
	if !shouldDeleteOnStatus(behavior.Lifecycle, status) {
		return
	}
	if err := s.Delete(ctx, sandbox.ID); err != nil {
		s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
			"message": "failed to delete sandbox on terminal status: " + err.Error(),
		})
	}
}

func shouldDeleteOnStatus(cfg types.LifecycleConfig, status types.Status) bool {
	switch status {
	case types.StatusApproved:
		return hasLifecycleEvent(cfg.DeleteOn, types.LifecycleEventApproved) ||
			hasLifecycleEvent(cfg.DeleteOn, types.LifecycleEventTerminal)
	case types.StatusRejected:
		return hasLifecycleEvent(cfg.DeleteOn, types.LifecycleEventRejected) ||
			hasLifecycleEvent(cfg.DeleteOn, types.LifecycleEventTerminal)
	default:
		return false
	}
}

func hasLifecycleEvent(events []types.LifecycleEvent, event types.LifecycleEvent) bool {
	for _, e := range events {
		if e == event {
			return true
		}
	}
	return false
}

// normalizeBehavior fills in default acceptance mode and normalizes
// allow/deny criteria. Idempotent.
func normalizeBehavior(b types.SandboxBehavior) types.SandboxBehavior {
	if b.Acceptance.Mode == "" {
		b.Acceptance.Mode = "allowlist"
	}
	b.Acceptance.Allow = normalizeCriteria(b.Acceptance.Allow)
	b.Acceptance.Deny = normalizeCriteria(b.Acceptance.Deny)
	return b
}

// validateBehavior rejects malformed sandbox behaviors at create time.
func validateBehavior(b types.SandboxBehavior) error {
	if b.Acceptance.Mode != "" && b.Acceptance.Mode != "allowlist" {
		return types.NewValidationError("acceptance.mode", "unsupported acceptance mode")
	}
	if b.Lifecycle.TTL < 0 {
		return types.NewValidationError("lifecycle.ttl", "ttl cannot be negative")
	}
	if b.Lifecycle.IdleTimeout < 0 {
		return types.NewValidationError("lifecycle.idleTimeout", "idleTimeout cannot be negative")
	}
	for _, p := range append(b.Acceptance.Allow.PathGlobs, b.Acceptance.Deny.PathGlobs...) {
		if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
			return types.NewValidationErrorWithHint(
				"acceptance.pathGlobs",
				"path globs must be project-root relative",
				"Remove the leading '/' and use project-root relative patterns",
			)
		}
	}
	return nil
}
