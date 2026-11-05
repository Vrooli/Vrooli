// Package orchestration provides the core orchestration service for agent-manager.
//
// This file contains APPROVAL WORKFLOW operations.
// All approval-related logic is grouped here for easier understanding.

package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// approvalLog returns the approval workflow's component-tagged logger.
// Centralised so every approval/rejection site logs under the same
// component name; aggregator queries can then filter on
// component=approval cleanly.
func approvalLog() *slog.Logger { return obs.Component("approval") }

// =============================================================================
// FULL APPROVAL
// =============================================================================

// ApproveRun applies all sandbox changes to the canonical repository.
func (o *Orchestrator) ApproveRun(ctx context.Context, req ApproveRequest) (*ApproveResult, error) {
	// Fetch and validate run
	run, err := o.getRunForApproval(ctx, req.RunID)
	if err != nil {
		return nil, err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("approve")
	}

	if allowed, reason := domain.CanApproveRun(run); !allowed {
		return nil, domain.NewStateError("Run", string(run.Status), "approve", reason)
	}

	// Apply changes via sandbox
	sandboxID := *run.SandboxID
	result, err := o.sandbox.Approve(ctx, sandbox.ApproveRequest{
		SandboxID:           sandboxID,
		Actor:               req.Actor,
		CommitMsg:           req.CommitMsg,
		Force:               req.Force,
		ExpectedPatchSHA256: req.ExpectedPatchSHA256,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, &domain.SandboxError{
			SandboxID: &sandboxID,
			Operation: "approve",
			Cause:     errors.New("workspace-sandbox returned no approval result"),
		}
	}
	if !result.Success {
		failure := strings.TrimSpace(result.ErrorMsg)
		if failure == "" {
			failure = "workspace-sandbox approval did not succeed"
		}
		return nil, &domain.SandboxError{
			SandboxID: &sandboxID,
			Operation: "approve",
			Cause:     errors.New(failure),
		}
	}

	// Workspace Sandbox may accept only part of a manual Mode=all request.
	// Preserve that outcome on the run instead of marking all changes approved.
	if result.IsPartial || result.Remaining > 0 {
		if err := o.markRunPartiallyApproved(ctx, run); err != nil {
			approvalLog().Warn("mark run partially approved failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
			return nil, err
		}
		return mapApproveResult(result), nil
	}

	// Update run to approved state only after the provider reports full success.
	if err := o.markRunApproved(ctx, run, req.Actor); err != nil {
		approvalLog().Warn("mark run approved failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return nil, err
	}

	return mapApproveResult(result), nil
}

// =============================================================================
// REJECTION
// =============================================================================

// RejectRun marks sandbox changes as rejected without applying.
func (o *Orchestrator) RejectRun(ctx context.Context, id uuid.UUID, actor, reason string) error {
	return o.rejectRun(ctx, id, actor, reason, "")
}

// RejectRunWithExpectedPatch is the Hub-driven decision path. It refuses to
// claim rejection unless the sandbox owner confirms the exact reviewed patch.
func (o *Orchestrator) RejectRunWithExpectedPatch(ctx context.Context, id uuid.UUID, actor, reason, expectedPatchSHA256 string) error {
	digest, err := hex.DecodeString(expectedPatchSHA256)
	if err != nil || len(digest) != sha256.Size || strings.ToLower(expectedPatchSHA256) != expectedPatchSHA256 {
		return domain.NewValidationError("expectedPatchSha256", "must be a lowercase SHA-256 hex digest")
	}
	return o.rejectRun(ctx, id, actor, reason, expectedPatchSHA256)
}

func (o *Orchestrator) rejectRun(ctx context.Context, id uuid.UUID, actor, reason, expectedPatchSHA256 string) error {
	run, err := o.getRunForApproval(ctx, id)
	if err != nil {
		return err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return importedRunLifecycleError("reject")
	}
	if allowed, rejectReason := domain.CanRejectRun(run); !allowed {
		return domain.NewStateError("Run", string(run.Status), "reject", rejectReason)
	}
	if expectedPatchSHA256 != "" {
		if run.SandboxID == nil {
			return domain.NewValidationError("sandboxId", "reviewed rejection requires the run sandbox")
		}
		if o.sandbox == nil {
			return domain.NewConfigMissingError("sandbox", "reviewed rejection requires a sandbox provider", nil)
		}
	}

	if run.SandboxID != nil && o.sandbox != nil {
		var rejectErr error
		if expectedPatchSHA256 != "" {
			rejectErr = o.sandbox.RejectReviewed(ctx, sandbox.RejectRequest{
				SandboxID: *run.SandboxID, Actor: actor, ExpectedPatchSHA256: expectedPatchSHA256,
			})
		} else {
			rejectErr = o.sandbox.Reject(ctx, *run.SandboxID, actor)
		}
		if rejectErr != nil {
			approvalLog().Warn("sandbox reject failed",
				obs.KeySandboxID, run.SandboxID.String(), obs.KeyError, rejectErr.Error())
			// Rejection is a prerequisite to disposal and run-state mutation. On
			// provider failure or patch mismatch, preserve the pending run/sandbox.
			return rejectErr
		}
		if err := o.sandbox.Delete(ctx, *run.SandboxID); err != nil {
			approvalLog().Warn("sandbox delete failed",
				obs.KeySandboxID, run.SandboxID.String(), obs.KeyError, err.Error())
		}
	}
	return o.markRunRejected(ctx, run, actor, reason)
}

// =============================================================================
// PARTIAL APPROVAL
// =============================================================================

// PartialApprove approves only selected files from the sandbox.
func (o *Orchestrator) PartialApprove(ctx context.Context, req PartialApproveRequest) (*ApproveResult, error) {
	// Fetch and validate run
	run, err := o.getRunForApproval(ctx, req.RunID)
	if err != nil {
		return nil, err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("partial-approve")
	}

	if allowed, reason := domain.CanApproveRun(run); !allowed {
		return nil, domain.NewStateError("Run", string(run.Status), "partial_approve", reason)
	}

	// Apply partial changes via sandbox
	result, err := o.sandbox.PartialApprove(ctx, sandbox.PartialApproveRequest{
		SandboxID: *run.SandboxID,
		FileIDs:   req.FileIDs,
		Actor:     req.Actor,
		CommitMsg: req.CommitMsg,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, &domain.SandboxError{
			SandboxID: run.SandboxID,
			Operation: "partial_approve",
			Cause:     errors.New("workspace-sandbox returned no approval result"),
		}
	}
	if !result.Success {
		failure := strings.TrimSpace(result.ErrorMsg)
		if failure == "" {
			failure = "workspace-sandbox partial approval did not succeed"
		}
		return nil, &domain.SandboxError{
			SandboxID: run.SandboxID,
			Operation: "partial_approve",
			Cause:     errors.New(failure),
		}
	}
	// Update run state based on remaining files
	if result.Remaining == 0 && !result.IsPartial {
		if err := o.markRunApproved(ctx, run, req.Actor); err != nil {
			approvalLog().Warn("mark run approved failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
			return nil, err
		}
	} else {
		if err := o.markRunPartiallyApproved(ctx, run); err != nil {
			approvalLog().Warn("mark run partially approved failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
			return nil, err
		}
	}

	return mapApproveResult(result), nil
}

// SyncRunFromSandbox updates run state based on workspace-sandbox approval events.
func (o *Orchestrator) SyncRunFromSandbox(ctx context.Context, req SandboxSyncRequest) (*domain.Run, error) {
	run, err := o.GetRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("sandbox-sync")
	}

	if req.SandboxID != nil && run.SandboxID != nil && *req.SandboxID != *run.SandboxID {
		return nil, domain.NewValidationError("sandboxId", "sandbox ID does not match run")
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	actor := strings.TrimSpace(req.Actor)
	if actor == "" {
		actor = "workspace-sandbox"
	}

	now := o.now()
	switch status {
	case "approved":
		if req.IsPartial {
			run.ApprovalState = domain.ApprovalStatePartiallyApproved
		} else {
			run.ApprovalState = domain.ApprovalStateApproved
			run.ApprovedBy = actor
			run.ApprovedAt = &now
			run.Status = domain.RunStatusComplete
			run.Phase = domain.RunPhaseCompleted
			run.EndedAt = &now
		}
	case "rejected":
		run.ApprovalState = domain.ApprovalStateRejected
		run.ApprovedBy = actor
		run.ApprovedAt = &now
		run.Status = domain.RunStatusFailed
		run.Phase = domain.RunPhaseCompleted
		run.EndedAt = &now
		if req.Reason != "" {
			run.ErrorMsg = req.Reason
		}
	default:
		return nil, domain.NewValidationError("status", "unsupported sandbox sync status")
	}

	run.UpdatedAt = now
	if err := o.runs.Update(ctx, run); err != nil {
		return nil, err
	}

	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(run)
	}

	return run, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// getRunForApproval fetches a run and validates it has basic approval prerequisites.
func (o *Orchestrator) getRunForApproval(ctx context.Context, runID uuid.UUID) (*domain.Run, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if o.sandbox == nil {
		return nil, domain.NewConfigMissingError("sandbox", "sandbox provider not configured", nil)
	}

	return run, nil
}

// markRunApproved updates run to fully approved state.
func (o *Orchestrator) markRunApproved(ctx context.Context, run *domain.Run, actor string) error {
	now := o.now()
	run.ApprovalState = domain.ApprovalStateApproved
	run.ApprovedBy = actor
	run.ApprovedAt = &now
	run.Status = domain.RunStatusComplete
	run.Phase = domain.RunPhaseCompleted
	run.EndedAt = &now
	run.UpdatedAt = now
	return o.runs.Update(ctx, run)
}

// markRunRejected updates run to rejected state.
func (o *Orchestrator) markRunRejected(ctx context.Context, run *domain.Run, actor, reason string) error {
	now := o.now()
	run.ApprovalState = domain.ApprovalStateRejected
	run.ApprovedBy = actor
	run.ApprovedAt = &now
	run.ErrorMsg = reason
	run.Status = domain.RunStatusFailed
	run.Phase = domain.RunPhaseCompleted
	run.EndedAt = &now
	run.UpdatedAt = now
	return o.runs.Update(ctx, run)
}

// markRunPartiallyApproved updates run to partial approval state.
func (o *Orchestrator) markRunPartiallyApproved(ctx context.Context, run *domain.Run) error {
	now := o.now()
	run.ApprovalState = domain.ApprovalStatePartiallyApproved
	run.UpdatedAt = now
	return o.runs.Update(ctx, run)
}

// mapApproveResult converts sandbox result to orchestration result.
func mapApproveResult(r *sandbox.ApproveResult) *ApproveResult {
	return &ApproveResult{
		Success:    r.Success,
		Applied:    r.Applied,
		Remaining:  r.Remaining,
		IsPartial:  r.IsPartial,
		CommitHash: r.CommitHash,
		AppliedAt:  r.AppliedAt,
		ErrorMsg:   r.ErrorMsg,
	}
}
