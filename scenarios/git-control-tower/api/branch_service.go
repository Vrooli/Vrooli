package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// BranchDeps contains dependencies for branch operations.
type BranchDeps struct {
	Git     GitRunner
	RepoDir string
}

// ListBranches returns local and remote branches for the repository.
// [REQ:GCT-OT-P1-001] Branch operations
func ListBranches(ctx context.Context, deps BranchDeps) (*RepoBranchesResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	out, err := deps.Git.Branches(ctx, repoDir)
	if err != nil {
		return nil, err
	}

	refs, err := ParseBranchRefs(out)
	if err != nil {
		return nil, err
	}

	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     deps.Git,
		RepoDir: repoDir,
	})
	if err != nil {
		return nil, err
	}

	locals := make([]BranchInfo, 0)
	remotes := make([]BranchInfo, 0)

	for _, ref := range refs {
		info := BranchInfo{
			Name:         ref.ShortName,
			Upstream:     ref.Upstream,
			OID:          ref.OID,
			LastCommitAt: ref.LastCommitAt,
		}
		if !ref.IsRemote && ref.ShortName == status.Branch.Head {
			info.IsCurrent = true
			info.Ahead = status.Branch.Ahead
			info.Behind = status.Branch.Behind
			if status.Branch.Upstream != "" {
				info.Upstream = status.Branch.Upstream
			}
			if status.Branch.OID != "" {
				info.OID = status.Branch.OID
			}
		}

		if ref.IsRemote {
			remotes = append(remotes, info)
		} else {
			locals = append(locals, info)
		}
	}

	return &RepoBranchesResponse{
		Current:   status.Branch.Head,
		Locals:    locals,
		Remotes:   remotes,
		Timestamp: time.Now().UTC(),
	}, nil
}

// CreateBranch creates a new branch and optionally checks it out.
func CreateBranch(ctx context.Context, deps BranchDeps, req CreateBranchRequest) (*BranchCreateResponse, error) {
	resp := &BranchCreateResponse{Timestamp: time.Now().UTC()}
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		resp.Success = false
		resp.ValidationErrors = []string{"branch name is required"}
		return resp, nil
	}

	if err := deps.Git.CheckRefFormat(ctx, repoDir, name); err != nil {
		resp.Success = false
		resp.ValidationErrors = []string{err.Error()}
		return resp, nil
	}

	refs, err := deps.Git.Branches(ctx, repoDir)
	if err != nil {
		return nil, err
	}
	parsed, err := ParseBranchRefs(refs)
	if err != nil {
		return nil, err
	}
	if branchExists(parsed, name) {
		resp.Success = false
		resp.Error = "branch already exists"
		return resp, nil
	}

	var status *RepoStatus
	if req.Checkout || strings.TrimSpace(req.From) == "" {
		st, err := GetRepoStatus(ctx, RepoStatusDeps{Git: deps.Git, RepoDir: repoDir})
		if err != nil {
			return nil, err
		}
		status = st
	}

	if req.Checkout && status != nil && isDirtySummary(status.Summary) && !req.AllowDirty {
		resp.Success = false
		resp.Warning = &BranchWarning{
			Message:              "Working tree has uncommitted changes",
			RequiresConfirmation: true,
			DirtySummary:         &status.Summary,
		}
		return resp, nil
	}

	from := strings.TrimSpace(req.From)
	if from == "" && status != nil {
		from = status.Branch.Head
	}

	if err := deps.Git.CreateBranch(ctx, repoDir, name, from); err != nil {
		resp.Success = false
		resp.Error = err.Error()
		return resp, nil
	}

	if req.Checkout {
		if err := deps.Git.CheckoutBranch(ctx, repoDir, name); err != nil {
			resp.Success = false
			resp.Error = err.Error()
			return resp, nil
		}
		if st, err := GetRepoStatus(ctx, RepoStatusDeps{Git: deps.Git, RepoDir: repoDir}); err == nil {
			resp.Branch = &BranchInfo{
				Name:      st.Branch.Head,
				Upstream:  st.Branch.Upstream,
				OID:       st.Branch.OID,
				Ahead:     st.Branch.Ahead,
				Behind:    st.Branch.Behind,
				IsCurrent: true,
			}
		}
	} else {
		resp.Branch = &BranchInfo{Name: name}
	}

	resp.Success = true
	return resp, nil
}

// SwitchBranch changes the current branch.
func SwitchBranch(ctx context.Context, deps BranchDeps, req SwitchBranchRequest) (*BranchSwitchResponse, error) {
	resp := &BranchSwitchResponse{Timestamp: time.Now().UTC()}
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		resp.Success = false
		resp.Error = "branch name is required"
		return resp, nil
	}

	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: deps.Git, RepoDir: repoDir})
	if err != nil {
		return nil, err
	}
	if isDirtySummary(status.Summary) && !req.AllowDirty {
		resp.Success = false
		resp.Warning = &BranchWarning{
			Message:              "Working tree has uncommitted changes",
			RequiresConfirmation: true,
			DirtySummary:         &status.Summary,
		}
		return resp, nil
	}

	refsRaw, err := deps.Git.Branches(ctx, repoDir)
	if err != nil {
		return nil, err
	}
	refs, err := ParseBranchRefs(refsRaw)
	if err != nil {
		return nil, err
	}

	if branchExists(refs, name) {
		if err := deps.Git.CheckoutBranch(ctx, repoDir, name); err != nil {
			resp.Success = false
			resp.Error = err.Error()
			return resp, nil
		}
		return populateSwitchBranch(ctx, deps, resp)
	}

	remoteRef, ok := findRemoteBranch(refs, name)
	if ok {
		if !req.TrackRemote {
			resp.Success = false
			resp.Warning = &BranchWarning{
				Message:          "Branch exists on remote. Track it before switching.",
				RequiresTracking: true,
			}
			return resp, nil
		}
		remote, branchName, err := splitRemoteBranch(remoteRef.ShortName)
		if err != nil {
			resp.Success = false
			resp.Error = err.Error()
			return resp, nil
		}
		if err := deps.Git.TrackRemoteBranch(ctx, repoDir, remote, branchName); err != nil {
			resp.Success = false
			resp.Error = err.Error()
			return resp, nil
		}
		return populateSwitchBranch(ctx, deps, resp)
	}

	resp.Success = false
	resp.Error = "branch not found"
	return resp, nil
}

// PublishBranch pushes the current branch to a remote.
func PublishBranch(ctx context.Context, deps BranchDeps, req PublishBranchRequest) (*BranchPublishResponse, error) {
	resp := &BranchPublishResponse{Timestamp: time.Now().UTC()}
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: deps.Git, RepoDir: repoDir})
	if err != nil {
		return nil, err
	}

	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = status.Branch.Head
	}
	if branch == "" {
		resp.Success = false
		resp.Error = "branch name is required"
		return resp, nil
	}
	if req.Branch != "" && branch != status.Branch.Head {
		resp.Success = false
		resp.Error = "publish is limited to the current branch"
		return resp, nil
	}

	remote := strings.TrimSpace(req.Remote)
	if remote == "" {
		remote = resolveBranchRemote(ctx, deps, branch, status.Branch.Upstream)
	}

	sync, err := GetSyncStatus(ctx, SyncStatusDeps{Git: deps.Git, RepoDir: repoDir}, SyncStatusRequest{
		Remote: remote,
		Fetch:  req.Fetch,
	})
	if err != nil {
		return nil, err
	}

	if sync.Behind > 0 {
		resp.Success = false
		warning := &BranchWarning{Message: "Branch is behind remote. Pull before publishing."}
		if !req.Fetch {
			warning.Message = "Fetch remote updates before publishing."
			warning.RequiresFetch = true
		}
		resp.Warning = warning
		return resp, nil
	}

	setUpstream := req.SetUpstream
	if !sync.HasUpstream {
		setUpstream = true
	}

	if err := deps.Git.Push(ctx, repoDir, remote, branch, setUpstream); err != nil {
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
	resp.Branch = &BranchInfo{
		Name:      status.Branch.Head,
		Upstream:  status.Branch.Upstream,
		OID:       status.Branch.OID,
		Ahead:     status.Branch.Ahead,
		Behind:    status.Branch.Behind,
		IsCurrent: true,
	}
	return resp, nil
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
