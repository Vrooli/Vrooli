package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PublishBranch pushes the current branch to a remote.
func PublishBranch(ctx context.Context, deps BranchDeps, req PublishBranchRequest) (*BranchPublishResponse, error) {
	resp := &BranchPublishResponse{Timestamp: time.Now().UTC()}
	repoDir, err := validateBranchDeps(deps)
	if err != nil {
		return nil, err
	}

	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: deps.Git, RepoDir: repoDir})
	if err != nil {
		return nil, err
	}

	branch, branchErr := resolvePublishBranch(req.Branch, status.Branch.Head)
	if branchErr != "" {
		resp.Success = false
		resp.Error = branchErr
		return resp, nil
	}

	remote := strings.TrimSpace(req.Remote)
	if remote == "" {
		remote = resolveBranchRemote(ctx, deps, branch, status.Branch.Upstream)
	}

	if warning, err := checkBehindRemote(ctx, deps, repoDir, remote, req.Fetch); err != nil {
		return nil, err
	} else if warning != nil {
		resp.Success = false
		resp.Warning = warning
		return resp, nil
	}

	setUpstream := req.SetUpstream
	sync, _ := GetSyncStatus(ctx, SyncStatusDeps{Git: deps.Git, RepoDir: repoDir}, SyncStatusRequest{Remote: remote, Fetch: false})
	if sync != nil && !sync.HasUpstream {
		setUpstream = true
	}

	if err := deps.Git.Push(ctx, repoDir, remote, branch, setUpstream, nil); err != nil {
		resp.Success = false
		resp.Remote = remote
		resp.Branch = branch
		resp.Error = err.Error()
		return resp, nil
	}

	resp.Success = true
	resp.Remote = remote
	resp.Branch = branch
	return resp, nil
}

// resolvePublishBranch resolves and validates the branch name for publishing.
func resolvePublishBranch(reqBranch, headBranch string) (string, string) {
	branch := strings.TrimSpace(reqBranch)
	if branch == "" {
		branch = headBranch
	}
	if branch == "" {
		return "", "branch name is required"
	}
	if reqBranch != "" && branch != headBranch {
		return "", "publish is limited to the current branch"
	}
	return branch, ""
}

// checkBehindRemote checks if the branch is behind the remote and returns a warning if so.
func checkBehindRemote(ctx context.Context, deps BranchDeps, repoDir, remote string, fetch bool) (*BranchWarning, error) {
	sync, err := GetSyncStatus(ctx, SyncStatusDeps{Git: deps.Git, RepoDir: repoDir}, SyncStatusRequest{
		Remote: remote,
		Fetch:  fetch,
	})
	if err != nil {
		return nil, err
	}
	if sync.Behind <= 0 {
		return nil, nil
	}
	warning := &BranchWarning{Message: "Branch is behind remote. Pull before publishing."}
	if !fetch {
		warning.Message = "Fetch remote updates before publishing."
		warning.RequiresFetch = true
	}
	return warning, nil
}

func isDirtySummary(summary RepoStatusSummary) bool {
	return summary.Staged > 0 || summary.Unstaged > 0 || summary.Untracked > 0 || summary.Conflicts > 0
}

func branchExists(refs []ParsedBranchRef, name string) bool {
	for _, ref := range refs {
		if !ref.IsRemote && ref.ShortName == name {
			return true
		}
	}
	return false
}

func findRemoteBranch(refs []ParsedBranchRef, name string) (ParsedBranchRef, bool) {
	for _, ref := range refs {
		if ref.IsRemote && ref.ShortName == name {
			return ref, true
		}
	}
	return ParsedBranchRef{}, false
}

func splitRemoteBranch(remoteName string) (string, string, error) {
	parts := strings.SplitN(remoteName, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid remote branch name")
	}
	return parts[0], parts[1], nil
}

func populateSwitchBranch(ctx context.Context, deps BranchDeps, resp *BranchSwitchResponse) (*BranchSwitchResponse, error) {
	status, err := GetRepoStatus(ctx, RepoStatusDeps(deps))
	if err != nil {
		return nil, err
	}
	resp.Success = true
	resp.Branch = branchInfoFromStatus(status)
	return resp, nil
}

// validateBranchDeps validates common branch operation dependencies.
func validateBranchDeps(deps BranchDeps) (string, error) {
	if deps.Git == nil {
		return "", fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return "", fmt.Errorf("repo dir is required")
	}
	return repoDir, nil
}

// branchInfoFromStatus creates a BranchInfo from repo status.
func branchInfoFromStatus(status *RepoStatus) *BranchInfo {
	return &BranchInfo{
		Name:      status.Branch.Head,
		Upstream:  status.Branch.Upstream,
		OID:       status.Branch.OID,
		Ahead:     status.Branch.Ahead,
		Behind:    status.Branch.Behind,
		IsCurrent: true,
	}
}

// checkBranchExists checks whether a local branch with the given name exists.
func checkBranchExists(ctx context.Context, deps BranchDeps, repoDir, name string) (bool, error) {
	refs, err := deps.Git.Branches(ctx, repoDir)
	if err != nil {
		return false, err
	}
	parsed, err := ParseBranchRefs(refs)
	if err != nil {
		return false, err
	}
	return branchExists(parsed, name), nil
}

// fetchStatusIfNeeded fetches repo status only when checkout or from-resolution requires it.
func fetchStatusIfNeeded(ctx context.Context, deps BranchDeps, repoDir string, checkout, fromEmpty bool) (*RepoStatus, error) {
	if !checkout && !fromEmpty {
		return nil, nil
	}
	return GetRepoStatus(ctx, RepoStatusDeps{Git: deps.Git, RepoDir: repoDir})
}

// checkDirtyWorktree returns a warning if the worktree is dirty and allowDirty is false.
func checkDirtyWorktree(ctx context.Context, deps BranchDeps, repoDir string, allowDirty bool) (*BranchWarning, error) {
	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: deps.Git, RepoDir: repoDir})
	if err != nil {
		return nil, err
	}
	if isDirtySummary(status.Summary) && !allowDirty {
		return &BranchWarning{
			Message:              "Working tree has uncommitted changes",
			RequiresConfirmation: true,
			DirtySummary:         &status.Summary,
		}, nil
	}
	return nil, nil
}

// listParsedBranches retrieves and parses branch refs.
func listParsedBranches(ctx context.Context, deps BranchDeps, repoDir string) ([]ParsedBranchRef, error) {
	refsRaw, err := deps.Git.Branches(ctx, repoDir)
	if err != nil {
		return nil, err
	}
	return ParseBranchRefs(refsRaw)
}

func resolveBranchRemote(ctx context.Context, deps BranchDeps, branch string, upstream string) string {
	key := fmt.Sprintf("branch.%s.remote", branch)
	if remote, err := deps.Git.ConfigGet(ctx, deps.RepoDir, key); err == nil {
		if trimmed := strings.TrimSpace(remote); trimmed != "" {
			return trimmed
		}
	}
	if upstream != "" {
		if remote, _, err := splitRemoteBranch(upstream); err == nil {
			return remote
		}
	}
	return "origin"
}
