package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SyncStatusDeps contains dependencies for sync status operations.
type SyncStatusDeps struct {
	Git     GitRunner
	RepoDir string
}

// GetSyncStatus retrieves the push/pull status for the repository.
// [REQ:GCT-OT-P0-006] Push/pull status
func GetSyncStatus(ctx context.Context, deps SyncStatusDeps, req SyncStatusRequest) (*SyncStatusResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	remote := req.Remote
	if remote == "" {
		remote = "origin"
	}

	resp := &SyncStatusResponse{
		Timestamp: time.Now().UTC(),
	}

	// DECISION BOUNDARY: Optionally fetch from remote for accurate counts
	if req.Fetch {
		if err := deps.Git.FetchRemote(ctx, repoDir, remote); err != nil {
			// Fetch failure is non-fatal - continue with stale data
			resp.FetchError = err.Error()
		} else {
			resp.Fetched = true
		}
	}

	// Get remote URL for display
	remoteURL, err := deps.Git.GetRemoteURL(ctx, repoDir, remote)
	if err == nil {
		resp.RemoteURL = remoteURL
	}

	// Get repository status (includes branch info and file changes)
	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     deps.Git,
		RepoDir: repoDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get repo status: %w", err)
	}

	// Populate branch info
	resp.Branch = status.Branch.Head
	resp.Upstream = status.Branch.Upstream
	resp.Ahead = status.Branch.Ahead
	resp.Behind = status.Branch.Behind
	resp.HasUpstream = status.Branch.Upstream != ""

	// Calculate uncommitted changes
	hasUncommitted := status.Summary.Staged > 0 || status.Summary.Unstaged > 0
	resp.HasUncommittedChanges = hasUncommitted

	// Calculate needs
	resp.NeedsPush = resp.Ahead > 0
	resp.NeedsPull = resp.Behind > 0

	// DECISION BOUNDARY: Determine safety for push/pull operations
	resp.CanPush = calculateCanPush(resp)
	resp.CanPull = calculateCanPull(resp)

	// Generate safety warnings
	resp.SafetyWarnings = generateSafetyWarnings(resp, status)

	// Generate recommendations
	resp.Recommendations = generateRecommendations(resp, status)

	return resp, nil
}

// calculateCanPush determines if push is safe.
// DECISION BOUNDARY: Push safety rules
func calculateCanPush(resp *SyncStatusResponse) bool {
	// Must have upstream to push
	if !resp.HasUpstream {
		return false
	}

	// Must have commits to push
	if resp.Ahead == 0 {
		return false
	}

	// If behind, regular push won't work (need pull first or force)
	if resp.Behind > 0 {
		return false
	}

	// Having uncommitted changes doesn't block push, but it's risky
	// Let the warnings inform the user
	return true
}

// calculateCanPull determines if pull is safe.
// DECISION BOUNDARY: Pull safety rules
func calculateCanPull(resp *SyncStatusResponse) bool {
	// Must have upstream to pull
	if !resp.HasUpstream {
		return false
	}

	// Must be behind to need pull
	if resp.Behind == 0 {
		return false
	}

	// Uncommitted changes may conflict with pull
	// Still allow it but warn the user
	return true
}

// generateSafetyWarnings creates warnings about the current state.
func generateSafetyWarnings(resp *SyncStatusResponse, status *RepoStatus) []string {
	var warnings []string

	if !resp.HasUpstream {
		warnings = append(warnings, "No upstream branch configured - push requires explicit remote/branch")
	}

	if resp.NeedsPush && resp.NeedsPull {
		warnings = append(warnings, "Branch has diverged from remote - pull before push to avoid force-push")
	}

	if resp.HasUncommittedChanges && resp.NeedsPull {
		warnings = append(warnings, "Uncommitted changes may conflict with incoming changes during pull")
	}

	if status.Summary.Conflicts > 0 {
		warnings = append(warnings, fmt.Sprintf("%d file(s) have unresolved merge conflicts", status.Summary.Conflicts))
	}

	if resp.Behind > 10 {
		warnings = append(warnings, fmt.Sprintf("Branch is %d commits behind - consider rebasing", resp.Behind))
	}

	return warnings
}

// generateRecommendations creates suggested actions.
func generateRecommendations(resp *SyncStatusResponse, status *RepoStatus) []string {
	var recs []string

	if !resp.HasUpstream {
		recs = append(recs, "Set upstream with: git push -u origin <branch>")
	}

	if status.Summary.Conflicts > 0 {
		recs = append(recs, "Resolve merge conflicts before continuing")
		return recs // Conflicts take priority
	}

	if resp.HasUncommittedChanges {
		if status.Summary.Staged > 0 {
			recs = append(recs, "Commit staged changes or stash them")
		}
		if status.Summary.Unstaged > 0 {
			recs = append(recs, "Stage and commit changes, or stash them")
		}
	}

	if resp.NeedsPull && resp.NeedsPush {
		recs = append(recs, "Pull remote changes first, then push")
	} else if resp.NeedsPull {
		recs = append(recs, "Pull to update local branch")
	} else if resp.NeedsPush {
		recs = append(recs, "Push to update remote branch")
	}

	if len(recs) == 0 && !resp.HasUncommittedChanges && !resp.NeedsPush && !resp.NeedsPull {
		recs = append(recs, "Branch is in sync with remote")
	}

	return recs
}
