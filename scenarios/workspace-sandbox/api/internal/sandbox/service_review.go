package sandbox

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/policy"
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

// GetDiff generates a diff for the sandbox changes.
//
// # Preconditions
//
// The sandbox must be in a state where diff generation is valid (Active, Stopped,
// or terminal states for historical view). The overlay directories must exist.
func (s *Service) GetDiff(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
	sandbox, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
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
		if err := s.repo.Update(ctx, sandbox); err != nil {
			fmt.Printf("warning: failed to update sandbox metrics: %v\n", err)
		}
	}

	if len(changes) == 0 {
		return &types.DiffResult{
			SandboxID:   sandbox.ID,
			Files:       []*types.FileChange{},
			UnifiedDiff: "",
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
	sandbox, err := s.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, err
	}

	if sandbox.Status == types.StatusApproved {
		return &types.ApprovalResult{
			Success:   true,
			Applied:   0,
			AppliedAt: *sandbox.ApprovedAt,
		}, nil
	}

	if err := types.CanApprove(sandbox.Status); err != nil {
		return nil, types.NewStateError(err.(*types.InvalidTransitionError))
	}

	allChanges, err := s.driver.GetChangedFiles(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes: %w", err)
	}

	allChanges = filterDiffChanges(allChanges)

	if err := s.preflightConflicts(ctx, sandbox, allChanges, req.Force); err != nil {
		return nil, err
	}

	totalChanges := len(allChanges)
	changes := allChanges

	if req.Mode == "hunks" && len(req.HunkRanges) > 0 {
		fileIDs := make([]uuid.UUID, 0, len(req.HunkRanges))
		seen := make(map[uuid.UUID]bool)
		for _, hr := range req.HunkRanges {
			if !seen[hr.FileID] {
				seen[hr.FileID] = true
				fileIDs = append(fileIDs, hr.FileID)
			}
		}
		changes = diff.FilterChanges(allChanges, fileIDs)
	}

	if req.Mode == "files" && len(req.FileIDs) > 0 {
		changes = diff.FilterChanges(allChanges, req.FileIDs)
	}

	accepted, rejected := filterChangesByAcceptance(sandbox, changes, req.OverrideAcceptance)
	if !req.OverrideAcceptance && req.Mode != "all" && len(rejected) > 0 {
		// Build diagnostic message showing which files were rejected and why.
		// Without file-level detail, callers can't tell a bad glob from an
		// empty deny rule from an intentional restriction.
		rejectedDetails := make([]string, 0, len(rejected))
		for _, r := range rejected {
			reason := "unknown"
			if r.Acceptance != nil {
				reason = r.Acceptance.Reason
			}
			rejectedDetails = append(rejectedDetails, fmt.Sprintf("%s (%s)", r.FilePath, reason))
		}
		msg := fmt.Sprintf(
			"%d file(s) rejected by acceptance rules: %s",
			len(rejected),
			strings.Join(rejectedDetails, ", "),
		)
		return nil, types.NewValidationErrorWithHint(
			"acceptance",
			msg,
			"Use overrideAcceptance=true to apply files outside acceptance rules",
		)
	}
	changes = accepted

	if len(changes) == 0 {
		return &types.ApprovalResult{
			Success:   true,
			Applied:   0,
			Remaining: totalChanges,
			AppliedAt: s.clock.Now(),
		}, nil
	}

	if s.validationPolicy != nil {
		if err := s.validationPolicy.ValidateBeforeApply(ctx, sandbox, changes); err != nil {
			return nil, fmt.Errorf("pre-apply validation failed: %w", err)
		}
	}

	gen := diff.NewGenerator(s.starter)
	diffOpts := &diff.GenerateOptions{
		PathPrefix: scopePathPrefix(sandbox),
	}
	diffResult, err := gen.GenerateDiff(ctx, sandbox, changes, diffOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff: %w", err)
	}

	// [OT-P1-001] Hunk-Level Approval
	if req.Mode == "hunks" && len(req.HunkRanges) > 0 {
		diffResult.UnifiedDiff = diff.FilterHunks(diffResult.UnifiedDiff, req.HunkRanges, changes)
		if diffResult.UnifiedDiff == "" {
			return &types.ApprovalResult{
				Success:   true,
				Applied:   0,
				AppliedAt: s.clock.Now(),
			}, nil
		}
	}

	commitMsg := req.CommitMsg
	author := req.Actor
	if s.attributionPolicy != nil {
		if commitMsg == "" {
			commitMsg = s.attributionPolicy.GetCommitMessage(ctx, sandbox, changes, req.CommitMsg)
		}
		author = s.attributionPolicy.GetCommitAuthor(ctx, sandbox, req.Actor)

		coAuthors := s.attributionPolicy.GetCoAuthors(ctx, sandbox, req.Actor)
		if len(coAuthors) > 0 {
			commitMsg = policy.FormatCommitMessage(commitMsg, coAuthors)
		}
	}

	filePaths := make([]string, len(changes))
	for i, change := range changes {
		filePaths[i] = change.FilePath
	}

	patcher := diff.NewPatcher(s.starter)
	applyResult, err := patcher.ApplyDiff(ctx, sandbox.ProjectRoot, diffResult.UnifiedDiff, diff.ApplyOptions{
		CommitMsg:    commitMsg,
		Author:       author,
		CreateCommit: req.CreateCommit,
		FilePaths:    filePaths,
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
			AppliedAt: s.clock.Now(),
		}, nil
	}

	s.recordApprovalProvenance(ctx, sandbox, changes, req, applyResult.CommitHash, commitMsg)
	return s.finalizeApproval(ctx, sandbox, req, applyResult.CommitHash, changes, totalChanges), nil
}

// ApplyAtRunEnd is the agent-manager run-end apply path. It validates
// the run-context metadata, translates the request into an internal
// ApprovalRequest, and routes through Approve so the per-file state
// machine cannot drift between operator approval and auto-apply.
//
// Source MUST be SourceAgentManagerAutoApply; other values are rejected.
func (s *Service) ApplyAtRunEnd(ctx context.Context, req *types.ApplyAtRunEndRequest) (*types.ApprovalResult, error) {
	if req == nil {
		return nil, types.NewValidationError("request", "request body is required")
	}
	if err := validateApplyAtRunEndRequest(req); err != nil {
		return nil, err
	}

	// apply-at-run-end is always a full-sandbox apply ("all"); per-file
	// partitioning is owned by the acceptance filter inside Approve.
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

	result, err := s.Approve(ctx, approvalReq)
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

	sandbox.Status = types.StatusRejected
	if err := s.repo.Update(ctx, sandbox); err != nil {
		return nil, fmt.Errorf("failed to update sandbox: %w", err)
	}

	s.logAuditEvent(ctx, sandbox, "rejected", actor, "", nil)

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
func (s *Service) finalizeApproval(ctx context.Context, sandbox *types.Sandbox, req *types.ApprovalRequest, commitHash string, changes []*types.FileChange, totalChanges int) *types.ApprovalResult {
	remainingChanges := totalChanges - len(changes)
	isPartial := remainingChanges > 0
	now := s.clock.Now()

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
		sandbox.Status = types.StatusApproved
		sandbox.ApprovedAt = &now
		if err := s.repo.Update(ctx, sandbox); err != nil {
			fmt.Printf("warning: failed to update sandbox after approval: %v\n", err)
		}

		s.logAuditEvent(ctx, sandbox, "approved", req.Actor, "", map[string]interface{}{
			"filesApplied": len(changes),
			"commitHash":   commitHash,
			"mode":         req.Mode,
		})

		s.applyLifecycleOnTerminal(ctx, sandbox, types.StatusApproved)
	}

	result := &types.ApprovalResult{
		Success:    true,
		Applied:    len(changes),
		Remaining:  remainingChanges,
		IsPartial:  isPartial,
		CommitHash: commitHash,
		AppliedAt:  now,
	}
	s.notifyAgentManager(ctx, sandbox, "approved", req.Actor, result)
	return result
}

// recordApprovalProvenance writes one AppliedChange per applied file,
// stamps the run-end metadata, and (when a commit was produced) marks
// them committed. Best-effort: provenance failures are audit-logged
// but never surface to the caller, since the apply itself already
// succeeded.
func (s *Service) recordApprovalProvenance(ctx context.Context, sandbox *types.Sandbox, changes []*types.FileChange, req *types.ApprovalRequest, commitHash, commitMsg string) {
	runID := req.AgentManagerRunID
	if runID == "" {
		runID = metadataString(sandbox.Metadata, metadataAgentManagerRunID)
	}
	appliedChanges := make([]*types.AppliedChange, len(changes))
	for i, c := range changes {
		appliedChanges[i] = &types.AppliedChange{
			ID:                uuid.New(),
			SandboxID:         sandbox.ID,
			SandboxOwner:      sandbox.Owner,
			SandboxOwnerType:  string(sandbox.OwnerType),
			FilePath:          filepath.Join(sandbox.ProjectRoot, c.FilePath),
			ProjectRoot:       sandbox.ProjectRoot,
			ChangeType:        string(c.ChangeType),
			FileSize:          c.FileSize,
			AgentManagerRunID: runID,
			ConversationID:    req.ConversationID,
			CostUSD:           req.Cost,
			RunOutcome:        req.RunOutcome,
			ProvenanceState:   string(types.ProvenanceFileStateApplied),
		}
	}

	if err := s.repo.RecordAppliedChanges(ctx, appliedChanges); err != nil {
		s.logAuditEvent(ctx, sandbox, "provenance.warning", "system", "system", map[string]interface{}{
			"message": "failed to record provenance: " + err.Error(),
		})
	}

	if commitHash == "" {
		return
	}
	ids := make([]uuid.UUID, len(appliedChanges))
	for i, c := range appliedChanges {
		ids[i] = c.ID
	}
	if err := s.repo.MarkChangesCommitted(ctx, ids, commitHash, commitMsg); err != nil {
		s.logAuditEvent(ctx, sandbox, "provenance.warning", "system", "system", map[string]interface{}{
			"message": "failed to mark changes committed: " + err.Error(),
		})
	}
}
