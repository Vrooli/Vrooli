package sandbox

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

func (s *Service) TurnCheckpoint(ctx context.Context, req *types.TurnCheckpointRequest) (*types.TurnCheckpointResult, error) {
	if req == nil {
		return nil, types.NewValidationError("request", "request body is required")
	}
	if err := validateTurnCheckpointRequest(req); err != nil {
		return nil, err
	}

	sandbox, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}
	if err := types.CanCheckpointTurn(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	now := s.clock.Now()
	sandbox.Status = types.StatusCheckpointing
	sandbox.LastUsedAt = now
	sandbox.UpdatedAt = now
	sandbox.ErrorMsg = ""
	if err := s.repo.Update(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to mark sandbox checkpointing: %w", err)
	}

	restoreActive := func(stage string, cause error) error {
		sandbox.Status = types.StatusActive
		sandbox.ErrorMsg = ""
		sandbox.LastUsedAt = s.clock.Now()
		sandbox.UpdatedAt = sandbox.LastUsedAt
		if err := s.repo.Update(ctx, sandbox); err != nil {
			return fmt.Errorf("%s: %w; additionally failed to restore active checkpoint state: %v", stage, cause, err)
		}
		return cause
	}

	approvalReq := &types.ApprovalRequest{
		SandboxID:         req.SandboxID,
		Mode:              "all",
		Actor:             req.Actor,
		CommitMsg:         req.CommitMsg,
		Source:            req.Source,
		Force:             req.Force,
		CreateCommit:      req.CreateCommit,
		AgentManagerRunID: req.AgentManagerRunID,
		ConversationID:    req.ConversationID,
		Cost:              req.Cost,
		RunOutcome:        req.RunOutcome,
	}

	applyResult, err := s.applyAcceptedChanges(ctx, sandbox, approvalReq)
	if err != nil {
		return nil, restoreActive("failed to apply accepted turn changes", err)
	}
	if !applyResult.Empty && !applyResult.Success {
		message := applyResult.ErrorMsg
		if message == "" {
			message = "turn apply failed"
		}
		if err := restoreActive("failed turn apply", errors.New(message)); err != nil {
			return nil, err
		}
		return &types.TurnCheckpointResult{
			SandboxID:  sandbox.ID,
			Status:     sandbox.Status,
			Success:    false,
			Applied:    len(applyResult.Changes),
			Failed:     applyResult.Failed,
			Remaining:  applyResult.Remaining,
			ErrorMsg:   applyResult.ErrorMsg,
			AppliedAt:  applyResult.AppliedAt,
			CommitHash: applyResult.CommitHash,
		}, nil
	}

	if !applyResult.Empty {
		if _, err := s.recordFileProvenance(ctx, sandbox, applyResult.Changes, approvalReq, types.ProvenanceFileStateApplied, applyResult.CommitHash, applyResult.CommitMsg); err != nil {
			return nil, restoreActive("failed to record applied turn provenance", err)
		}
		for _, change := range applyResult.Changes {
			if err := s.driver.RemoveFromUpper(ctx, sandbox, change.FilePath); err != nil {
				s.logAuditEvent(ctx, sandbox, "turn_checkpoint_cleanup_warning", req.Actor, "", map[string]interface{}{
					"file":  change.FilePath,
					"error": err.Error(),
				})
				return nil, restoreActive("failed to checkpoint applied file", fmt.Errorf("failed to checkpoint applied file %s: %w", change.FilePath, err))
			}
		}
	}
	if len(applyResult.Rejected) > 0 {
		pendingChanges, err := s.newPendingReviewChanges(ctx, sandbox, applyResult.Rejected)
		if err != nil {
			return nil, restoreActive("failed to prepare pending-review turn provenance", err)
		}
		if _, err := s.recordFileProvenance(ctx, sandbox, pendingChanges, approvalReq, types.ProvenanceFileStatePendingReview, "", ""); err != nil {
			return nil, restoreActive("failed to record pending-review turn provenance", err)
		}
	}

	baseHash, hashErr := s.gitOps.GetCommitHash(ctx, sandbox.ProjectRoot)
	if hashErr != nil {
		s.logAuditEvent(ctx, sandbox, "turn_checkpoint.warning", req.Actor, "", map[string]interface{}{
			"message": "failed to refresh base commit hash: " + hashErr.Error(),
		})
	} else {
		sandbox.BaseCommitHash = baseHash
	}

	s.runPreTeardownHooks(ctx, sandbox, "turn_checkpoint")
	if err := s.driver.Unmount(ctx, sandbox); err != nil {
		_ = restoreActive("failed to unmount checkpointed sandbox", err)
		return nil, fmt.Errorf("failed to unmount checkpointed sandbox: %w", err)
	}

	now = s.clock.Now()
	sandbox.Status = types.StatusCheckpointed
	sandbox.LastUsedAt = now
	sandbox.UpdatedAt = now
	if err := s.repo.Update(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to update checkpointed sandbox: %w", err)
	}

	checkpointID := uuid.New().String()
	var appliedSizeBytes int64
	for _, change := range applyResult.Changes {
		appliedSizeBytes += change.FileSize
	}
	s.logAuditEvent(ctx, sandbox, "turn_checkpointed", req.Actor, "agent", map[string]interface{}{
		"agentManagerRunId": req.AgentManagerRunID,
		"conversationId":    req.ConversationID,
		"turnId":            req.TurnID,
		"turnSequence":      req.TurnSequence,
		"cost":              req.Cost,
		"runOutcome":        req.RunOutcome,
		"source":            string(req.Source),
		"applied":           len(applyResult.Changes),
		"remaining":         applyResult.Remaining,
		"isPartial":         applyResult.Remaining > 0,
		"checkpointId":      checkpointID,
		"baseCommitHash":    sandbox.BaseCommitHash,
	})

	return &types.TurnCheckpointResult{
		SandboxID:        sandbox.ID,
		Status:           sandbox.Status,
		Success:          true,
		Applied:          len(applyResult.Changes),
		Remaining:        applyResult.Remaining,
		IsPartial:        applyResult.Remaining > 0,
		CommitHash:       applyResult.CommitHash,
		BaseCommitHash:   sandbox.BaseCommitHash,
		CheckpointID:     checkpointID,
		AppliedAt:        now,
		AppliedSizeBytes: appliedSizeBytes,
		DiffPath:         fmt.Sprintf("/api/v1/sandboxes/%s/diff", sandbox.ID),
	}, nil
}

func (s *Service) Resume(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if sandbox.Status == types.StatusActive {
		return sandbox, nil
	}
	if err := types.CanResumeWork(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	paths, err := s.driver.Mount(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to resume sandbox: %w", err)
	}

	sandbox.LowerDir = paths.LowerDir
	sandbox.UpperDir = paths.UpperDir
	sandbox.WorkDir = paths.WorkDir
	sandbox.MergedDir = paths.MergedDir
	sandbox.HomeLowerDir = paths.HomeLowerDir
	sandbox.HomeUpperDir = paths.HomeUpperDir
	sandbox.HomeWorkDir = paths.HomeWorkDir
	sandbox.HomeMergedDir = paths.HomeMergedDir
	sandbox.Status = types.StatusActive
	sandbox.LastUsedAt = s.clock.Now()

	if err := s.repo.Update(ctx, sandbox); err != nil {
		if unmountErr := s.driver.Unmount(ctx, sandbox); unmountErr != nil {
			fmt.Printf("warning: driver unmount failed: %v\n", unmountErr)
		}
		return nil, fmt.Errorf("failed to update resumed sandbox: %w", err)
	}

	s.logAuditEvent(ctx, sandbox, "resumed", "", "", nil)
	return sandbox, nil
}

func (s *Service) newPendingReviewChanges(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange) ([]*types.FileChange, error) {
	existing, err := s.repo.GetPendingChangeFiles(ctx, sandbox.ProjectRoot, []uuid.UUID{sandbox.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect existing pending-review provenance: %w", err)
	}
	seen := make(map[string]bool, len(existing))
	for _, change := range existing {
		if change == nil || change.SandboxID != sandbox.ID || change.ProvenanceState != string(types.ProvenanceFileStatePendingReview) {
			continue
		}
		seen[filepath.Clean(change.FilePath)] = true
		if rel, err := filepath.Rel(sandbox.ProjectRoot, change.FilePath); err == nil {
			seen[filepath.ToSlash(filepath.Clean(rel))] = true
		}
	}

	out := make([]*types.FileChange, 0, len(changes))
	for _, change := range changes {
		if change == nil {
			continue
		}
		absPath := filepath.Clean(filepath.Join(sandbox.ProjectRoot, change.FilePath))
		relPath := filepath.ToSlash(filepath.Clean(change.FilePath))
		if seen[absPath] || seen[relPath] {
			continue
		}
		out = append(out, change)
	}
	return out, nil
}

func validateTurnCheckpointRequest(req *types.TurnCheckpointRequest) error {
	if req.SandboxID == uuid.Nil {
		return types.NewValidationError("sandboxId", "field is required")
	}
	if req.AgentManagerRunID == "" {
		return types.NewValidationError("agentManagerRunId", "field is required for turn checkpoint provenance")
	}
	if !req.Source.IsValidInbound() {
		return types.NewValidationErrorWithHint(
			"source",
			"invalid source for turn checkpoint",
			"only the SourceAgentManagerAutoApply ('agent-manager-auto-apply') value is accepted on this endpoint",
		)
	}
	if req.Source != types.SourceAgentManagerAutoApply {
		return types.NewValidationErrorWithHint(
			"source",
			"turn checkpoint requires source=agent-manager-auto-apply",
			"operator approvals must use POST /sandboxes/{id}/approve instead",
		)
	}
	if req.RunOutcome != "" {
		switch req.RunOutcome {
		case "success", "failure", "cancelled", "timeout":
		default:
			return types.NewValidationErrorWithHint(
				"runOutcome",
				"invalid runOutcome for turn checkpoint",
				"valid values: success, failure, cancelled, timeout",
			)
		}
	}
	return nil
}
