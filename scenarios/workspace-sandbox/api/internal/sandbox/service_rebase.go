package sandbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/types"
)

// service_rebase.go: retry/rebase workflow [OT-P2-003].
//
// CheckConflicts is read-only and surfaces drift since sandbox
// creation. Rebase advances the sandbox's BaseCommitHash to the
// current repo state without merging — the sandbox's per-file
// changes stay intact; only the baseline reference is updated.

// CheckConflicts checks if the canonical repo has changed since
// sandbox creation and identifies any conflicting files. Read-only;
// safe to call repeatedly.
func (s *Service) CheckConflicts(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	response := &types.ConflictCheckResponse{
		BaseCommitHash: sandbox.BaseCommitHash,
		CheckedAt:      time.Now(),
	}

	sandboxChanges, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox changes: %w", err)
	}

	for _, change := range sandboxChanges {
		response.SandboxChangedFiles = append(response.SandboxChangedFiles, change.FilePath)
	}

	if sandbox.BaseCommitHash == "" {
		return response, nil
	}

	conflictCheck, err := s.gitOps.CheckForConflicts(ctx, sandbox, sandboxChanges)
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
// Does NOT merge canonical-repo changes into the sandbox; only updates
// the baseline reference for conflict detection.
func (s *Service) Rebase(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error) {
	sandbox, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}

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

	if req.Strategy == "" {
		req.Strategy = types.RebaseStrategyRegenerate
		result.Strategy = types.RebaseStrategyRegenerate
	}

	newHash, err := s.gitOps.GetCommitHash(ctx, sandbox.ProjectRoot)
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

	if sandbox.BaseCommitHash != "" && sandbox.BaseCommitHash != newHash {
		repoChangedFiles, err := s.gitOps.GetChangedFilesSince(ctx, sandbox.ProjectRoot, sandbox.BaseCommitHash)
		if err != nil {
			s.logAuditEvent(ctx, sandbox, "rebase.warning", req.Actor, "", map[string]interface{}{
				"message": "failed to get repo changed files: " + err.Error(),
			})
		} else {
			result.RepoChangedFiles = repoChangedFiles

			sandboxChanges, _ := s.driver.GetChangedFiles(ctx, sandbox)
			result.ConflictingFiles = diff.FindConflictingFiles(sandboxChanges, repoChangedFiles)
		}
	}

	sandbox.BaseCommitHash = newHash
	sandbox.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, sandbox); err != nil {
		result.Success = false
		result.ErrorMsg = fmt.Sprintf("failed to update sandbox: %v", err)
		return result, nil
	}

	result.Success = true

	s.logAuditEvent(ctx, sandbox, "rebased", req.Actor, "", map[string]interface{}{
		"previousBaseHash": result.PreviousBaseHash,
		"newBaseHash":      result.NewBaseHash,
		"strategy":         string(result.Strategy),
		"repoChangedFiles": len(result.RepoChangedFiles),
		"conflictingFiles": len(result.ConflictingFiles),
	})

	return result, nil
}
