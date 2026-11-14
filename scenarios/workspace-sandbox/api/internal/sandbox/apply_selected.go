package sandbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/policy"
	"workspace-sandbox/internal/types"
)

type selectedApplyResult struct {
	Success          bool
	Empty            bool
	Changes          []*types.FileChange
	Rejected         []*types.FileChange
	TotalChanges     int
	Failed           int
	Remaining        int
	CommitHash       string
	PatchSHA256      string
	CommitMsg        string
	ErrorMsg         string
	AppliedAt        time.Time
	PreparedArchive  *types.DiffArchive
	PreparedApproval *types.PreparedApproval
	Recovered        bool
}

func (r selectedApplyResult) ApprovalResult() *types.ApprovalResult {
	return &types.ApprovalResult{
		Success:    r.Success,
		Applied:    len(r.Changes),
		Failed:     r.Failed,
		Remaining:  r.Remaining,
		CommitHash: r.CommitHash,
		ErrorMsg:   r.ErrorMsg,
		AppliedAt:  r.AppliedAt,
	}
}

func (s *Service) applyAcceptedChanges(ctx context.Context, sandbox *types.Sandbox, req *types.ApprovalRequest, terminal bool) (*selectedApplyResult, error) {
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
	if !req.OverrideAcceptance && (req.Mode != "all" || req.ExpectedPatchSHA256 != "") && len(rejected) > 0 {
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
		if err := checkReviewedPatch(req.ExpectedPatchSHA256, diff.HashPatch("")); err != nil {
			return nil, err
		}
		return &selectedApplyResult{
			Success:      true,
			Empty:        true,
			Rejected:     rejected,
			TotalChanges: totalChanges,
			Remaining:    totalChanges,
			AppliedAt:    s.clock.Now(),
		}, nil
	}

	if s.validationPolicy != nil {
		if err := s.validationPolicy.ValidateBeforeApply(ctx, sandbox, changes); err != nil {
			return nil, fmt.Errorf("pre-apply validation failed: %w", err)
		}
	}

	prepareArchive := terminal && req.Mode != "hunks" && len(changes) == totalChanges && s.archiveRepo != nil && s.blobs != nil
	var diffResult *types.DiffResult
	var captured *capturedApprovalDiff
	if prepareArchive {
		captured, err = s.captureApprovalDiff(ctx, sandbox, changes, 0, 0)
		if err == nil {
			defer captured.close()
			diffResult = captured.result
		}
	} else {
		diffResult, err = diff.NewGenerator(s.starter).GenerateDiff(ctx, sandbox, changes, &diff.GenerateOptions{PathPrefix: scopePathPrefix(sandbox)})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff: %w", err)
	}
	changes = diffResult.Files

	if req.Mode == "hunks" && len(req.HunkRanges) > 0 {
		diffResult.UnifiedDiff = diff.FilterHunks(diffResult.UnifiedDiff, req.HunkRanges, changes)
		if diffResult.UnifiedDiff == "" {
			return &selectedApplyResult{
				Success:      true,
				Empty:        true,
				Rejected:     rejected,
				TotalChanges: totalChanges,
				AppliedAt:    s.clock.Now(),
			}, nil
		}
	}

	// Check the same in-memory bytes passed to ApplyDiff, not a separate read
	// of the mutable overlay. Conflict force does not weaken this precondition.
	patchSHA256 := diff.HashPatch(diffResult.UnifiedDiff)
	if err := checkReviewedPatch(req.ExpectedPatchSHA256, patchSHA256); err != nil {
		return nil, err
	}
	var prepared *types.DiffArchive
	var intent *types.PreparedApproval
	if prepareArchive {
		prepared = newDiffArchive(sandbox, types.StatusApproved)
		// Blob failures must occur before source effects. These content-addressed
		// bytes are retained on later apply/SQL failure, never re-read from an
		// overlay that the apply or a continuing process could have changed.
		if _, err := s.captureBlobs(ctx, &captured.sandbox, prepared, diffResult); err != nil {
			return nil, fmt.Errorf("prepare approval archive: %w", err)
		}
		if !req.CreateCommit {
			intent, err = s.prepareApproval(ctx, sandbox, req, captured, prepared)
			if err != nil {
				return nil, err
			}
		}
	}
	result, err := s.applyGeneratedChanges(ctx, sandbox, req, diffResult.UnifiedDiff, changes, rejected, totalChanges, prepared)
	if result != nil {
		result.PreparedApproval = intent
	}
	return result, err
}

func (s *Service) applyGeneratedChanges(ctx context.Context, sandbox *types.Sandbox, req *types.ApprovalRequest, patch string, changes, rejected []*types.FileChange, totalChanges int, prepared *types.DiffArchive) (*selectedApplyResult, error) {
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
	patchResult, err := patcher.ApplyDiff(ctx, sandbox.ProjectRoot, patch, diff.ApplyOptions{
		CommitMsg:    commitMsg,
		Author:       author,
		CreateCommit: req.CreateCommit,
		FilePaths:    filePaths,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to apply diff: %w", err)
	}

	if !patchResult.Success {
		return &selectedApplyResult{
			Success:      false,
			Changes:      changes,
			Rejected:     rejected,
			TotalChanges: totalChanges,
			Failed:       len(changes),
			Remaining:    totalChanges,
			ErrorMsg:     fmt.Sprintf("patch application failed: %v", patchResult.Errors),
			AppliedAt:    s.clock.Now(),
		}, nil
	}

	return &selectedApplyResult{
		Success:         true,
		Changes:         changes,
		Rejected:        rejected,
		TotalChanges:    totalChanges,
		Remaining:       totalChanges - len(changes),
		CommitHash:      patchResult.CommitHash,
		PatchSHA256:     diff.HashPatch(patch),
		CommitMsg:       commitMsg,
		AppliedAt:       s.clock.Now(),
		PreparedArchive: prepared,
	}, nil
}
