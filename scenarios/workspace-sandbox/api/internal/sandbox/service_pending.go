package sandbox

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/types"
)

// CommitReconciliationReport records one best-effort manual-commit sweep.
type CommitReconciliationReport struct {
	Scanned  int
	Repaired int
	Retired  int
	Failed   int
}

// ReconcileCommittedChanges resolves real hashes for changes committed outside
// Vrooli. It never mutates git state; rows without an eligible commit remain
// pending for a later pass.
func (s *Service) ReconcileCommittedChanges(ctx context.Context) CommitReconciliationReport {
	report := CommitReconciliationReport{}
	limit := s.config.CommitResolutionBatchLimit
	if limit < 1 {
		limit = 200
	}
	changes, err := s.repo.GetUnresolvedCommitChanges(ctx, limit)
	if err != nil {
		report.Failed = 1
		return report
	}
	repoState := make(map[string]bool)
	for _, change := range changes {
		report.Scanned++
		relPath := relativeToProjectRoot(change.ProjectRoot, change.FilePath)
		isRepo, known := repoState[change.ProjectRoot]
		if !known {
			isRepo = s.gitOps.IsGitRepo(ctx, change.ProjectRoot)
			repoState[change.ProjectRoot] = isRepo
		}
		if !isRepo {
			continue
		}
		hash, err := s.gitOps.ResolveCommitForPath(ctx, change.ProjectRoot, relPath, change.AppliedAt)
		if err != nil {
			report.Failed++
			continue
		}
		if hash == "" {
			if err := s.repo.IncrementCommitResolutionAttempts(ctx, change.ID); err != nil {
				report.Failed++
				continue
			}
			horizon := s.config.CommitResolutionHorizon
			if horizon < time.Hour {
				horizon = 720 * time.Hour
			}
			if s.clock.Now().Sub(change.AppliedAt) >= horizon {
				tracked, trackedErr := s.gitOps.IsTracked(ctx, change.ProjectRoot, relPath)
				if trackedErr != nil {
					report.Failed++
					continue
				}
				if !tracked {
					if err := s.repo.MarkCommitUnresolvable(ctx, change.ID, s.clock.Now()); err != nil {
						report.Failed++
						continue
					}
					report.Retired++
				}
			}
			continue
		}
		if err := s.repo.MarkChangesCommitted(ctx, []uuid.UUID{change.ID}, hash, "Committed externally (reconciled)"); err != nil {
			report.Failed++
			continue
		}
		report.Repaired++
	}
	return report
}

func (s *Service) PurgeUnresolvableCommitChanges(ctx context.Context) (int, error) {
	retention := s.config.UnresolvedProvenanceRetention
	if retention < time.Hour {
		retention = 168 * time.Hour
	}
	return s.repo.PurgeUnresolvableCommitChanges(ctx, s.clock.Now().Add(-retention))
}

// CommitReconcileInterval exists as a named policy boundary for callers that
// schedule the sweep. Zero disables periodic reconciliation.
// service_pending.go: provenance + pending-changes surface.
//
// Pending changes are AppliedChange rows that have been written to the
// sandbox but not yet committed to the canonical git repo. The API
// here lets agent-manager (and operators) inspect, group, preview,
// and finalize those pending records — including reconciling against
// git status to detect external commits.

// GetPendingChanges returns pending (uncommitted) changes grouped by sandbox.
func (s *Service) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	return s.repo.GetPendingChanges(ctx, projectRoot, limit, offset)
}

// GetFileProvenance returns the history of changes for a specific file.
func (s *Service) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	return s.repo.GetFileProvenance(ctx, filePath, projectRoot, limit)
}

// CommitPending commits pending changes to git and updates provenance records.
// Allows batching multiple sandbox changes into a single commit.
//
// Reconciliation behavior:
//   - Files that are still uncommitted in git are staged and committed
//   - Files that were already committed externally are marked as
//     reconciled with a special commit hash indicating external commit
func (s *Service) CommitPending(ctx context.Context, req *types.CommitPendingRequest) (*types.CommitPendingResult, error) {
	result := &types.CommitPendingResult{}

	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		result.ErrorMsg = "project root is required"
		return result, nil
	}

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

	reconciled, err := s.gitOps.ReconcilePendingWithGit(ctx, projectRoot, relPaths)
	if err != nil {
		fmt.Printf("warning: reconciliation failed, proceeding without: %v\n", err)
		reconciled = &diff.ReconcileResult{StillPending: relPaths}
	}

	if len(reconciled.AlreadyCommitted) > 0 {
		var externallyCommittedIDs []uuid.UUID
		for _, p := range reconciled.AlreadyCommitted {
			if change, ok := pathToChange[p]; ok {
				externallyCommittedIDs = append(externallyCommittedIDs, change.ID)
			}
		}
		if len(externallyCommittedIDs) > 0 {
			if err := s.repo.MarkChangesCommitted(ctx, externallyCommittedIDs, "EXTERNAL", "Committed externally (reconciled)"); err != nil {
				fmt.Printf("warning: failed to mark externally committed: %v\n", err)
			}
		}
	}

	if len(reconciled.StillPending) == 0 {
		result.Success = true
		result.FilesCommitted = 0
		return result, nil
	}

	commitMsg := req.CommitMessage
	if commitMsg == "" {
		commitMsg = s.generateDefaultCommitMessage(reconciled.StillPending, pathToChange)
	}

	patcher := diff.NewPatcher(s.starter)
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

	var committedIDs []uuid.UUID
	for _, p := range reconciled.StillPending {
		if change, ok := pathToChange[p]; ok {
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

// generateDefaultCommitMessage creates a descriptive commit message
// for a CommitPending call when the caller didn't provide one.
func (s *Service) generateDefaultCommitMessage(paths []string, pathToChange map[string]*types.AppliedChange) string {
	if len(paths) == 0 {
		return "No changes"
	}

	var added, modified, deleted int
	owners := make(map[string]bool)

	for _, p := range paths {
		if change, ok := pathToChange[p]; ok {
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

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Apply %d sandbox changes", len(paths)))

	ownerList := make([]string, 0, len(owners))
	for owner := range owners {
		ownerList = append(ownerList, owner)
	}
	if len(ownerList) == 1 {
		msg.WriteString(fmt.Sprintf(" from %s", ownerList[0]))
	} else if len(ownerList) > 1 && len(ownerList) <= 3 {
		msg.WriteString(fmt.Sprintf(" from %s", strings.Join(ownerList, ", ")))
	}

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

// GetCommitPreview returns a preview of what would be committed,
// reconciled against git status to detect externally-committed files.
func (s *Service) GetCommitPreview(ctx context.Context, req *types.CommitPreviewRequest) (*types.CommitPreviewResult, error) {
	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		return nil, fmt.Errorf("project root is required")
	}

	pendingChanges, err := s.repo.GetPendingChangeFiles(ctx, projectRoot, req.SandboxIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending changes: %w", err)
	}

	filePathSet := normalizeCommitPreviewFilePaths(projectRoot, req.FilePaths)
	if len(filePathSet) > 0 {
		filtered := make([]*types.AppliedChange, 0, len(pendingChanges))
		for _, change := range pendingChanges {
			relPath := relativeToProjectRoot(projectRoot, change.FilePath)
			if _, ok := filePathSet[relPath]; ok {
				filtered = append(filtered, change)
			}
		}
		pendingChanges = filtered
	}

	result := &types.CommitPreviewResult{
		Files:            make([]types.CommitPreviewFile, 0, len(pendingChanges)),
		GroupedBySandbox: []types.CommitPreviewSandboxGroup{},
	}

	if len(pendingChanges) == 0 {
		result.SuggestedMessage = "No pending changes"
		return result, nil
	}

	relPaths := make([]string, 0, len(pendingChanges))
	for _, change := range pendingChanges {
		relPath := relativeToProjectRoot(projectRoot, change.FilePath)
		if relPath != "" {
			relPaths = append(relPaths, relPath)
		}
	}

	reconciled, err := s.gitOps.ReconcilePendingWithGit(ctx, projectRoot, relPaths)
	if err != nil {
		fmt.Printf("warning: failed to reconcile with git: %v\n", err)
		reconciled = &diff.ReconcileResult{StillPending: relPaths}
	}

	stillPendingSet := make(map[string]bool)
	for _, p := range reconciled.StillPending {
		stillPendingSet[p] = true
	}

	sandboxGroups := make(map[uuid.UUID]*types.CommitPreviewSandboxGroup)

	for _, change := range pendingChanges {
		relPath := relativeToProjectRoot(projectRoot, change.FilePath)

		status := "already_committed"
		if stillPendingSet[relPath] {
			status = "pending"
			result.CommittableFiles++
		} else {
			result.AlreadyCommittedFiles++
		}

		file := types.CommitPreviewFile{
			FilePath:          change.FilePath,
			RelativePath:      relPath,
			ChangeType:        change.ChangeType,
			SandboxID:         change.SandboxID,
			SandboxOwner:      change.SandboxOwner,
			AgentManagerRunID: change.AgentManagerRunID,
			AppliedAt:         change.AppliedAt,
			Status:            status,
		}
		result.Files = append(result.Files, file)

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

	for _, group := range sandboxGroups {
		if group.FileCount > 0 {
			result.GroupedBySandbox = append(result.GroupedBySandbox, *group)
		}
	}

	result.SuggestedMessage = s.generateCommitMessage(result)

	return result, nil
}

func normalizeCommitPreviewFilePaths(projectRoot string, filePaths []string) map[string]struct{} {
	if len(filePaths) == 0 {
		return nil
	}
	normalized := make(map[string]struct{}, len(filePaths))
	for _, p := range filePaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		cleaned := filepath.Clean(p)
		if filepath.IsAbs(cleaned) {
			if projectRoot == "" || !strings.HasPrefix(cleaned, projectRoot) {
				continue
			}
			cleaned = strings.TrimPrefix(cleaned, projectRoot)
			cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
		} else {
			cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
		}
		if cleaned == "" || cleaned == "." {
			continue
		}
		normalized[cleaned] = struct{}{}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func relativeToProjectRoot(projectRoot, p string) string {
	relPath := p
	if projectRoot != "" && strings.HasPrefix(relPath, projectRoot) {
		relPath = strings.TrimPrefix(relPath, projectRoot)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	}
	return relPath
}

// generateCommitMessage builds a descriptive commit message from a
// preview, used by CommitPreview when the caller wants a suggestion.
func (s *Service) generateCommitMessage(preview *types.CommitPreviewResult) string {
	if preview.CommittableFiles == 0 {
		return "No uncommitted changes to apply"
	}

	var msg strings.Builder

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

	msg.WriteString(fmt.Sprintf("Apply %d sandbox changes", preview.CommittableFiles))

	if len(owners) == 1 {
		msg.WriteString(fmt.Sprintf(" from %s", owners[0]))
	} else if len(owners) <= 3 {
		msg.WriteString(fmt.Sprintf(" from %s", strings.Join(owners, ", ")))
	} else {
		msg.WriteString(fmt.Sprintf(" from %d sandboxes", len(preview.GroupedBySandbox)))
	}
	msg.WriteString("\n")

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

// MarkCommitted marks pending changes as committed for files that were
// committed by an external tool (e.g., git-control-tower).
func (s *Service) MarkCommitted(ctx context.Context, req *types.MarkCommittedRequest) (*types.MarkCommittedResult, error) {
	projectRoot := req.ProjectRoot
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		return nil, fmt.Errorf("project root is required")
	}
	if len(req.FilePaths) == 0 {
		return &types.MarkCommittedResult{}, nil
	}

	marked, notFound, err := s.repo.MarkChangesCommittedByPath(ctx, projectRoot, req.FilePaths, req.CommitHash, req.CommitMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to mark changes committed: %w", err)
	}

	return &types.MarkCommittedResult{
		MarkedCount:   marked,
		NotFoundCount: notFound,
	}, nil
}

// GetProvenanceByRun returns pending applied changes grouped by
// agent-manager run ID.
func (s *Service) GetProvenanceByRun(ctx context.Context, projectRoot string) ([]types.ProvenanceRunGroup, error) {
	if projectRoot == "" {
		projectRoot = s.config.DefaultProjectRoot
	}
	if projectRoot == "" {
		return nil, fmt.Errorf("project root is required")
	}

	return s.repo.GetPendingChangesByRun(ctx, projectRoot)
}
