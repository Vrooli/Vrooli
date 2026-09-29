package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/types"
)

// service_review.go: per-sandbox review surface.
//
// GetDiff exposes the change list. Approve/ApplyAtRunEnd/Reject/Discard
// drive the per-sandbox state machine. CheckConflicts/Rebase keep the
// sandbox aligned with the canonical repo's HEAD.
//
// All the run-end-metadata / acceptance-filter machinery sits under
// Approve; ApplyAtRunEnd is a thin translator that forwards to it so
// per-file state cannot diverge between operator and auto-apply paths.

// GetDiff owns live-versus-archived evidence selection for every transport.
// Terminal sandboxes never fall back to a surviving or recreated overlay.
func (s *Service) GetDiff(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	switch sandbox.Status {
	case types.StatusApproved, types.StatusRejected, types.StatusDeleted:
		archived, err := s.GetArchive(ctx, id)
		if err != nil || archived != nil {
			return archived, err
		}
		return &types.DiffResult{
			SandboxID: id, Files: []*types.FileChange{}, Generated: sandbox.UpdatedAt,
			ArchiveState: types.ArchiveStateNotCaptured,
		}, nil
	case types.StatusCreating:
		return &types.DiffResult{SandboxID: id, Files: []*types.FileChange{}, Generated: sandbox.CreatedAt}, nil
	}

	if err := types.CanGenerateDiff(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	if err := s.ensureDiffPaths(ctx, sandbox); err != nil {
		return nil, err
	}

	if err := s.ensureDiffDirectories(sandbox); err != nil {
		if errors.Is(err, errMissingUpperDir) {
			if sandbox.Status == types.StatusError {
				return &types.DiffResult{
					SandboxID:   sandbox.ID,
					Files:       []*types.FileChange{},
					UnifiedDiff: "",
					Generated:   s.clock.Now(),
				}, nil
			}
			return nil, &types.ValidationError{
				Field:   "upperDir",
				Message: "sandbox upper directory missing on disk",
				Hint:    "Sandbox data may have been cleaned up. Delete and recreate the sandbox.",
			}
		}
		return nil, err
	}

	changes, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, types.NewDriverError("getChangedFiles", err)
	}
	changes = filterDiffChanges(changes)
	applyAcceptanceInfo(sandbox, changes)

	var totalSizeBytes int64
	for _, change := range changes {
		if change.ChangeType == types.ChangeTypeAdded || change.ChangeType == types.ChangeTypeModified {
			totalSizeBytes += change.FileSize
		}
	}

	if sandbox.SizeBytes != totalSizeBytes || sandbox.FileCount != len(changes) {
		sandbox.SizeBytes = totalSizeBytes
		sandbox.FileCount = len(changes)
		if err := s.repo.UpdateWithVersionCheck(ctx, sandbox, sandbox.Version); err != nil {
			fmt.Printf("warning: failed to update sandbox metrics: %v\n", err)
		}
	}

	if len(changes) == 0 {
		return &types.DiffResult{
			SandboxID:   sandbox.ID,
			Files:       []*types.FileChange{},
			UnifiedDiff: "",
			PatchSHA256: diff.HashPatch(""),
			Generated:   s.clock.Now(),
		}, nil
	}

	gen := diff.NewGenerator(s.starter)
	opts := &diff.GenerateOptions{
		PathPrefix: scopePathPrefix(sandbox),
	}
	return gen.GenerateDiff(ctx, sandbox, changes, opts)
}

// filterDiffChanges drops any .git-relative entries from the change set.
// .git/ files must never appear in diffs the operator approves.
func filterDiffChanges(changes []*types.FileChange) []*types.FileChange {
	if len(changes) == 0 {
		return changes
	}
	filtered := make([]*types.FileChange, 0, len(changes))
	for _, change := range changes {
		if change == nil {
			continue
		}
		cleanPath := filepath.Clean(change.FilePath)
		if cleanPath == ".git" || strings.HasPrefix(cleanPath, ".git"+string(filepath.Separator)) {
			continue
		}
		filtered = append(filtered, change)
	}
	return filtered
}

// Approve applies sandbox changes to the canonical repo.
//
// Idempotent: calling Approve on an already-approved sandbox returns
// a success result indicating the prior approval (no re-apply, no
// duplicate commit).
func (s *Service) Approve(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error) {
	if req == nil {
		return nil, types.NewValidationError("request", "request body is required")
	}
	// Do not rewrite caller-owned request fields when binding a retained review.
	request := *req
	req = &request
	if (req.ReviewRequestID == uuid.Nil) != (req.ExpectedReviewSHA256 == "") {
		return nil, types.NewValidationError("review", "review request ID and expected review digest are required together")
	}
	if req.ReviewRequestID != uuid.Nil {
		if req.Mode != "all" || req.CreateCommit || len(req.FileIDs) != 0 || len(req.HunkRanges) != 0 {
			return nil, types.NewValidationError("review", "retained review approval requires all changes without a commit or partial selection")
		}
		snapshot, err := s.GetReviewSnapshot(ctx, req.SandboxID, req.ReviewRequestID)
		if err != nil {
			return nil, err
		}
		if snapshot.SHA256 != req.ExpectedReviewSHA256 {
			return nil, types.NewValidationError("review", "review content identity does not match retained input")
		}
		if err := checkReviewedPatch(req.ExpectedPatchSHA256, snapshot.PatchSHA256); err != nil {
			return nil, err
		}
		req.ExpectedPatchSHA256 = snapshot.PatchSHA256
	}
	if req.ExpectedPatchSHA256 != "" {
		digest, err := hex.DecodeString(req.ExpectedPatchSHA256)
		if err != nil || len(digest) != sha256.Size || strings.ToLower(req.ExpectedPatchSHA256) != req.ExpectedPatchSHA256 {
			return nil, types.NewValidationError("expectedPatchSha256", "must be a lowercase SHA-256 hex digest")
		}
		if req.Mode != "all" || len(req.FileIDs) != 0 || len(req.HunkRanges) != 0 {
			return nil, types.NewValidationError("mode", "reviewed patch approval requires all changes without file or hunk selection")
		}
	}
	release, err := s.lockReview(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	sandbox, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}

	if sandbox.Status == types.StatusApproved || (sandbox.Status == types.StatusDeleted && sandbox.ApprovedAt != nil) {
		if req.ReviewRequestID != uuid.Nil && (metadataString(sandbox.Metadata, metadataApprovedReviewSHA256) != req.ExpectedReviewSHA256 || metadataString(sandbox.Metadata, metadataApprovedReviewRequestID) != req.ReviewRequestID.String()) {
			return nil, types.NewValidationError("review", "terminal approval does not belong to this retained review")
		}
		appliedDigest := metadataString(sandbox.Metadata, metadataApprovedPatchSHA256)
		if err := checkReviewedPatch(req.ExpectedPatchSHA256, appliedDigest); err != nil {
			return nil, err
		}
		if sandbox.ApprovedAt == nil {
			return nil, fmt.Errorf("approved sandbox lacks approval timestamp")
		}
		return &types.ApprovalResult{
			AppliedPatchSHA256: appliedDigest,
			Success:            true,
			Applied:            0,
			AppliedAt:          *sandbox.ApprovedAt,
		}, nil
	}

	if err := types.CanApprove(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}
	if req.ReviewRequestID != uuid.Nil && sandbox.Status != types.StatusStopped {
		return nil, types.NewValidationError("review", "retained review approval requires a stopped sandbox")
	}
	if err := s.requireUnpublishedArchive(ctx, sandbox.ID); err != nil {
		return nil, err
	}

	applyResult, err := s.recoverPreparedApproval(ctx, sandbox, req)
	if err == nil && applyResult == nil {
		if req.ReviewRequestID != uuid.Nil {
			if err := s.checkReviewSource(ctx, sandbox, req, false); err != nil {
				return nil, err
			}
		}
		applyResult, err = s.applyAcceptedChanges(ctx, sandbox, req, true)
	}
	if err != nil {
		return nil, err
	}
	if applyResult.Empty {
		return &types.ApprovalResult{
			Success:   true,
			Applied:   0,
			Remaining: applyResult.TotalChanges,
			AppliedAt: s.clock.Now(),
		}, nil
	}
	if !applyResult.Success {
		return applyResult.ApprovalResult(), nil
	}

	if applyResult.PreparedApproval != nil {
		if err := s.recordPreparedApprovalProvenance(ctx, sandbox, applyResult.PreparedApproval); err != nil {
			return nil, fmt.Errorf("source applied but approval provenance is pending; retry original approval: %w", err)
		}
	} else {
		s.recordApprovalProvenance(ctx, sandbox, applyResult.Changes, req, applyResult.CommitHash, applyResult.CommitMsg)
	}
	if len(applyResult.Rejected) > 0 {
		pendingChanges, err := s.newPendingReviewChanges(ctx, sandbox, applyResult.Rejected)
		if err != nil {
			s.logAuditEvent(ctx, sandbox, "provenance.warning", "system", "system", map[string]interface{}{
				"message": "failed to inspect pending-review provenance: " + err.Error(),
			})
		} else if _, err := s.recordFileProvenance(ctx, sandbox, pendingChanges, req, types.ProvenanceFileStatePendingReview, "", ""); err != nil {
			s.logAuditEvent(ctx, sandbox, "provenance.warning", "system", "system", map[string]interface{}{
				"message": "failed to record pending-review provenance: " + err.Error(),
			})
		}
	}
	result := s.finalizeApproval(ctx, sandbox, req, applyResult)
	result.AppliedPatchSHA256 = applyResult.PatchSHA256
	if applyResult.Recovered {
		result.Applied, result.AppliedSizeBytes = 0, 0
	}
	// Lifecycle deletion acquires the same owner lock. Release only after the
	// archive/status transaction has published and consumed the pending intent.
	release()
	if result.Success {
		if !result.IsPartial {
			s.applyLifecycleOnTerminal(ctx, sandbox, types.StatusApproved)
		}
		s.notifyAgentManager(ctx, sandbox, "approved", req.Actor, result)
	}
	return result, nil
}

const metadataApprovedPatchSHA256 = "approved_patch_sha256"
const metadataApprovedReviewSHA256 = "approved_review_sha256"
const metadataApprovedReviewRequestID = "approved_review_request_id"

func checkReviewedPatch(expected, actual string) error {
	if expected != "" && expected != actual {
		return types.NewValidationErrorWithHint("expectedPatchSha256", "reviewed patch does not match the patch being approved", "Read and review the current diff before requesting approval; force cannot bypass content identity")
	}
	return nil
}

// ApplyAtRunEnd is the final agent-manager run-end apply path. Continuable
// turns use TurnCheckpoint; this endpoint applies accepted changes and then
// follows approval/final-disposal semantics.
//
// Source MUST be SourceAgentManagerAutoApply; other values are rejected.
func (s *Service) ApplyAtRunEnd(ctx context.Context, req *types.ApplyAtRunEndRequest) (*types.ApprovalResult, error) {
	if req == nil {
		return nil, types.NewValidationError("request", "request body is required")
	}
	if err := validateApplyAtRunEndRequest(req); err != nil {
		return nil, err
	}

	result, err := s.Approve(ctx, &types.ApprovalRequest{
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
	})
	if err != nil {
		return nil, err
	}

	// Best-effort audit annotation for the run-context metadata. Approve
	// already logged the "approved"/"partial_approved" event; this adds
	// a parallel record so audit consumers can group by run/conversation.
	if sandbox, getErr := s.Get(ctx, req.SandboxID); getErr == nil {
		s.logAuditEvent(ctx, sandbox, "applied_at_run_end", req.Actor, "agent", map[string]interface{}{
			"agentManagerRunId": req.AgentManagerRunID,
			"conversationId":    req.ConversationID,
			"cost":              req.Cost,
			"runOutcome":        req.RunOutcome,
			"source":            string(req.Source),
			"applied":           result.Applied,
			"remaining":         result.Remaining,
			"isPartial":         result.IsPartial,
		})
	}

	return result, nil
}

// validateApplyAtRunEndRequest rejects malformed apply-at-run-end
// requests before they reach the shared Approve path.
func validateApplyAtRunEndRequest(req *types.ApplyAtRunEndRequest) error {
	if strings.TrimSpace(req.AgentManagerRunID) == "" {
		return types.NewValidationError(
			"agentManagerRunId",
			"field is required for apply-at-run-end (provenance attribution)",
		)
	}
	if !req.Source.IsValidInbound() {
		return types.NewValidationErrorWithHint(
			"source",
			"invalid source for apply-at-run-end",
			"only the SourceAgentManagerAutoApply ('agent-manager-auto-apply') value is accepted on this endpoint",
		)
	}
	if req.Source != types.SourceAgentManagerAutoApply {
		return types.NewValidationErrorWithHint(
			"source",
			"apply-at-run-end requires source=agent-manager-auto-apply",
			"operator approvals must use POST /sandboxes/{id}/approve instead",
		)
	}
	if req.RunOutcome != "" {
		switch req.RunOutcome {
		case "success", "failure", "cancelled", "timeout":
			// ok
		default:
			return types.NewValidationErrorWithHint(
				"runOutcome",
				"invalid runOutcome for apply-at-run-end",
				"valid values: success, failure, cancelled, timeout",
			)
		}
	}
	return nil
}

// Reject marks sandbox changes as rejected.
//
// Idempotent: calling Reject on an already-rejected sandbox returns
// success with the current sandbox state.
func (s *Service) Reject(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error) {
	release, err := s.lockUnprepared(ctx, id)
	if err != nil {
		return nil, err
	}
	defer release()
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if sandbox.Status == types.StatusRejected {
		return sandbox, nil
	}

	if err := types.CanReject(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	captured := types.CanGenerateDiff(sandbox.Status) == nil
	if err := s.snapshotAndTransition(ctx, sandbox, types.StatusRejected, captured, nil, nil); err != nil {
		s.logAuditEvent(ctx, sandbox, "snapshot_failed", actor, "", map[string]interface{}{
			"phase": "reject",
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to snapshot+reject sandbox: %w", err)
	}

	s.logAuditEvent(ctx, sandbox, "rejected", actor, "", nil)

	release()
	s.applyLifecycleOnTerminal(ctx, sandbox, types.StatusRejected)

	s.notifyAgentManager(ctx, sandbox, "rejected", actor, nil)

	return sandbox, nil
}

// Discard lives in service_lifecycle.go (file-level removal, not a
// state transition). CheckConflicts and Rebase live in service_rebase.go.

// preflightConflicts runs the [OT-P2-002] conflict check and refuses
// the apply when the canonical repo has diverged with files that
// overlap the sandbox changes (unless force=true). Idempotent calls
// from the conflict-detection seam stay best-effort: a probe error
// is audit-logged but does not block apply.
func (s *Service) preflightConflicts(ctx context.Context, sandbox *types.Sandbox, allChanges []*types.FileChange, force bool) error {
	conflictCheck, err := s.gitOps.CheckForConflicts(ctx, sandbox, allChanges)
	if err != nil {
		s.logAuditEvent(ctx, sandbox, "sandbox.warning", "system", "system", map[string]interface{}{
			"message": "failed to check for conflicts: " + err.Error(),
		})
		return nil
	}
	if conflictCheck == nil || !conflictCheck.HasChanged || force {
		return nil
	}
	if len(conflictCheck.ConflictingFiles) > 0 {
		return types.NewRepoChangedErrorWithFiles(
			sandbox.ID.String(),
			conflictCheck.BaseCommitHash,
			conflictCheck.CurrentHash,
			conflictCheck.ConflictingFiles,
		)
	}
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
	return nil
}

// finalizeApproval applies the partial-vs-full state transition,
// emits the appropriate audit + agent-manager notifications, and
// returns the ApprovalResult. [OT-P1-002] Partial Approval Workflow:
// partial = applied files cleaned up from upper, sandbox stays open;
// full = sandbox transitions to StatusApproved + lifecycle terminal.
//
// On the full-approval branch, the diff archive is captured
// transactionally with the status flip (see service_archive.go). If
// the snapshot fails, the sandbox stays in its pre-terminal state and
// the failure is returned to the caller via *ApprovalResult.ErrorMsg —
// the patch was already applied to the canonical repo. Non-committing whole-file
// approval retains its durable intent so the original request can reconcile
// source and publish without re-applying. Other paths still need intervention.
func (s *Service) finalizeApproval(ctx context.Context, sandbox *types.Sandbox, req *types.ApprovalRequest, applied *selectedApplyResult) *types.ApprovalResult {
	changes, totalChanges, commitHash := applied.Changes, applied.TotalChanges, applied.CommitHash
	remainingChanges := totalChanges - len(changes)
	isPartial := remainingChanges > 0
	now := s.clock.Now()
	var appliedSizeBytes int64
	for _, change := range changes {
		appliedSizeBytes += change.FileSize
	}

	if isPartial {
		for _, change := range changes {
			if err := s.driver.RemoveFromUpper(ctx, sandbox, change.FilePath); err != nil {
				s.logAuditEvent(ctx, sandbox, "partial_cleanup_warning", req.Actor, "", map[string]interface{}{
					"file":  change.FilePath,
					"error": err.Error(),
				})
			}
		}
		sandbox.LastUsedAt = now
		if err := s.repo.Update(ctx, sandbox); err != nil {
			fmt.Printf("warning: failed to update sandbox after partial approval: %v\n", err)
		}

		s.logAuditEvent(ctx, sandbox, "partial_approved", req.Actor, "", map[string]interface{}{
			"filesApplied":   len(changes),
			"filesRemaining": remainingChanges,
			"commitHash":     commitHash,
			"mode":           req.Mode,
		})
	} else {
		captured := types.CanGenerateDiff(sandbox.Status) == nil
		if err := s.snapshotAndTransition(ctx, sandbox, types.StatusApproved, captured, func(sb *types.Sandbox) {
			sb.ApprovedAt = &now
			if sb.Metadata == nil {
				sb.Metadata = make(map[string]interface{})
			}
			sb.Metadata[metadataApprovedPatchSHA256] = applied.PatchSHA256
			if req.ReviewRequestID != uuid.Nil {
				sb.Metadata[metadataApprovedReviewSHA256] = req.ExpectedReviewSHA256
				sb.Metadata[metadataApprovedReviewRequestID] = req.ReviewRequestID.String()
			}
		}, applied.PreparedArchive); err != nil {
			s.logAuditEvent(ctx, sandbox, "snapshot_failed", req.Actor, "", map[string]interface{}{
				"phase":      "approve",
				"error":      err.Error(),
				"commitHash": commitHash,
			})
			return &types.ApprovalResult{
				Success:    false,
				Applied:    len(changes),
				Failed:     0,
				Remaining:  remainingChanges,
				CommitHash: commitHash,
				ErrorMsg:   fmt.Sprintf("apply succeeded but archive snapshot failed: %v", err),
				AppliedAt:  now,
			}
		}

		s.logAuditEvent(ctx, sandbox, "approved", req.Actor, "", map[string]interface{}{
			"filesApplied": len(changes),
			"commitHash":   commitHash,
			"mode":         req.Mode,
		})

	}

	result := &types.ApprovalResult{
		Success:          true,
		Applied:          len(changes),
		Remaining:        remainingChanges,
		IsPartial:        isPartial,
		CommitHash:       commitHash,
		AppliedAt:        now,
		AppliedSizeBytes: appliedSizeBytes,
		DiffPath:         fmt.Sprintf("/api/v1/sandboxes/%s/diff", sandbox.ID),
	}
	return result
}

// recordApprovalProvenance writes one AppliedChange per applied file,
// stamps the run-end metadata, and (when a commit was produced) marks
// them committed. Best-effort: provenance failures are audit-logged
// but never surface to the caller, since the apply itself already
// succeeded.
func (s *Service) recordApprovalProvenance(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange, req *types.ApprovalRequest, commitHash, commitMsg string) {
	if _, err := s.recordFileProvenance(ctx, sandbox, changes, req, types.ProvenanceFileStateApplied, commitHash, commitMsg); err != nil {
		s.logAuditEvent(ctx, sandbox, "provenance.warning", "system", "system", map[string]interface{}{
			"message": "failed to record provenance: " + err.Error(),
		})
	}
}

func (s *Service) recordFileProvenance(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange, req *types.ApprovalRequest, state types.ProvenanceFileState, commitHash, commitMsg string) ([]uuid.UUID, error) {
	if len(changes) == 0 {
		return nil, nil
	}

	runID := req.AgentManagerRunID
	if runID == "" {
		runID = metadataString(sandbox.Metadata, metadataAgentManagerRunID)
	}
	provenanceRoot := sandbox.ScopePath
	if provenanceRoot == "" {
		provenanceRoot = sandbox.ProjectRoot
	}
	appliedChanges := make([]*types.AppliedChange, len(changes))
	ids := make([]uuid.UUID, len(changes))
	for i, c := range changes {
		id := uuid.New()
		ids[i] = id
		appliedChanges[i] = &types.AppliedChange{
			ID:                id,
			SandboxID:         sandbox.ID,
			SandboxOwner:      sandbox.Owner,
			SandboxOwnerType:  string(sandbox.OwnerType),
			FilePath:          filepath.Join(provenanceRoot, c.FilePath),
			ProjectRoot:       sandbox.ProjectRoot,
			ChangeType:        string(c.ChangeType),
			FileSize:          c.FileSize,
			AgentManagerRunID: runID,
			ConversationID:    req.ConversationID,
			CostUSD:           req.Cost,
			RunOutcome:        req.RunOutcome,
			ProvenanceState:   string(state),
		}
		if c.ChangeType != types.ChangeTypeDeleted {
			if content, readErr := os.ReadFile(appliedChanges[i].FilePath); readErr == nil {
				digest := sha256.Sum256(content)
				appliedChanges[i].ContentDigest = fmt.Sprintf("sha256:%x", digest[:])
			}
		}
		appliedChanges[i].EvidenceRevision = commitHash
	}

	if err := s.repo.RecordAppliedChanges(ctx, appliedChanges); err != nil {
		return nil, err
	}
	if commitHash == "" {
		return ids, nil
	}
	if err := s.repo.MarkChangesCommitted(ctx, ids, commitHash, commitMsg); err != nil {
		return ids, err
	}
	return ids, nil
}
