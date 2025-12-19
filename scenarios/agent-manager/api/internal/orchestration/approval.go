// Package orchestration provides the core orchestration service for agent-manager.
//
// This file contains APPROVAL WORKFLOW operations.
// All approval-related logic is grouped here for easier understanding.

package orchestration

import (
	"context"
	"fmt"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

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

	// Check if approvable using domain decision helper
	if allowed, reason := run.IsApprovable(); !allowed {
		return nil, domain.NewStateError("Run", string(run.Status), "approve", reason)
	}

	// Apply changes via sandbox
	result, err := o.sandbox.Approve(ctx, sandbox.ApproveRequest{
		SandboxID: *run.SandboxID,
		Actor:     req.Actor,
		CommitMsg: req.CommitMsg,
		Force:     req.Force,
	})
	if err != nil {
		return nil, fmt.Errorf("sandbox approval failed: %w", err)
	}

	// Update run to approved state
	if err := o.markRunApproved(ctx, run, req.Actor); err != nil {
		// Log but don't fail - the approval itself succeeded
		// This is a known edge case to handle
	}

	return mapApproveResult(result), nil
}

// =============================================================================
// REJECTION
// =============================================================================

// RejectRun marks sandbox changes as rejected without applying.
func (o *Orchestrator) RejectRun(ctx context.Context, id uuid.UUID, actor, reason string) error {
	// Fetch and validate run
	run, err := o.getRunForApproval(ctx, id)
	if err != nil {
		return err
	}

	// Check if rejectable using domain decision helper
	if allowed, rejectReason := run.IsRejectable(); !allowed {
		return domain.NewStateError("Run", string(run.Status), "reject", rejectReason)
	}

	// Reject in sandbox (cleanup resources)
	if run.SandboxID != nil && o.sandbox != nil {
		if err := o.sandbox.Reject(ctx, *run.SandboxID, actor); err != nil {
			return fmt.Errorf("sandbox rejection failed: %w", err)
		}
	}

	// Update run state
	return o.markRunRejected(ctx, run, reason)
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

	// Check if approvable (same rules as full approval)
	if allowed, reason := run.IsApprovable(); !allowed {
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
		return nil, fmt.Errorf("partial approval failed: %w", err)
	}

	// Update run state based on remaining files
	if result.Remaining == 0 {
		o.markRunApproved(ctx, run, req.Actor)
	} else {
		o.markRunPartiallyApproved(ctx, run)
	}

	return mapApproveResult(result), nil
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
		return nil, fmt.Errorf("no sandbox provider configured")
	}

	return run, nil
}

// markRunApproved updates run to fully approved state.
func (o *Orchestrator) markRunApproved(ctx context.Context, run *domain.Run, actor string) error {
	now := time.Now()
	run.ApprovalState = domain.ApprovalStateApproved
	run.ApprovedBy = actor
	run.ApprovedAt = &now
	run.UpdatedAt = now
	return o.runs.Update(ctx, run)
}

// markRunRejected updates run to rejected state.
func (o *Orchestrator) markRunRejected(ctx context.Context, run *domain.Run, reason string) error {
	now := time.Now()
	run.ApprovalState = domain.ApprovalStateRejected
	run.ErrorMsg = reason
	run.UpdatedAt = now
	return o.runs.Update(ctx, run)
}

// markRunPartiallyApproved updates run to partial approval state.
func (o *Orchestrator) markRunPartiallyApproved(ctx context.Context, run *domain.Run) error {
	now := time.Now()
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
