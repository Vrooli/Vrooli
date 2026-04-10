package main

import (
	"context"
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
	repoDir, err := validateBranchDeps(deps)
	if err != nil {
		return nil, err
	}

	refs, err := listParsedBranches(ctx, deps, repoDir)
	if err != nil {
		return nil, err
	}

	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: deps.Git, RepoDir: repoDir})
	if err != nil {
		return nil, err
	}

	locals, remotes := classifyBranchRefs(refs, status)
	return &RepoBranchesResponse{
		Current:   status.Branch.Head,
		Locals:    locals,
		Remotes:   remotes,
		Timestamp: time.Now().UTC(),
	}, nil
}

// classifyBranchRefs converts parsed refs into local and remote BranchInfo slices.
func classifyBranchRefs(refs []ParsedBranchRef, status *RepoStatus) ([]BranchInfo, []BranchInfo) {
	locals := make([]BranchInfo, 0)
	remotes := make([]BranchInfo, 0)
	for _, ref := range refs {
		info := refToBranchInfo(ref, status)
		if ref.IsRemote {
			remotes = append(remotes, info)
		} else {
			locals = append(locals, info)
		}
	}
	return locals, remotes
}

// refToBranchInfo converts a ParsedBranchRef to a BranchInfo, enriching with status if current.
func refToBranchInfo(ref ParsedBranchRef, status *RepoStatus) BranchInfo {
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
	return info
}

// CreateBranch creates a new branch and optionally checks it out.
func CreateBranch(ctx context.Context, deps BranchDeps, req CreateBranchRequest) (*BranchCreateResponse, error) {
	resp := &BranchCreateResponse{Timestamp: time.Now().UTC()}
	repoDir, err := validateBranchDeps(deps)
	if err != nil {
		return nil, err
	}

	name, validationErr := validateBranchName(ctx, deps, repoDir, req.Name)
	if validationErr != nil {
		resp.Success = false
		resp.ValidationErrors = validationErr
		return resp, nil
	}

	if exists, err := checkBranchExists(ctx, deps, repoDir, name); err != nil {
		return nil, err
	} else if exists {
		resp.Success = false
		resp.Error = "branch already exists"
		return resp, nil
	}

	status, err := fetchStatusIfNeeded(ctx, deps, repoDir, req.Checkout, strings.TrimSpace(req.From) == "")
	if err != nil {
		return nil, err
	}

	if warning := checkDirtyForCheckout(req.Checkout, req.AllowDirty, status); warning != nil {
		resp.Success = false
		resp.Warning = warning
		return resp, nil
	}

	from := resolveFromBranch(req.From, status)
	if err := deps.Git.CreateBranch(ctx, repoDir, name, from); err != nil {
		resp.Success = false
		resp.Error = err.Error()
		return resp, nil
	}

	return finalizeCreateBranch(ctx, deps, resp, repoDir, name, req.Checkout)
}

// validateBranchName validates and returns the trimmed branch name, or validation errors.
func validateBranchName(ctx context.Context, deps BranchDeps, repoDir, rawName string) (string, []string) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return "", []string{"branch name is required"}
	}
	if err := deps.Git.CheckRefFormat(ctx, repoDir, name); err != nil {
		return "", []string{err.Error()}
	}
	return name, nil
}

// checkDirtyForCheckout returns a warning if checkout is requested on a dirty worktree.
func checkDirtyForCheckout(checkout, allowDirty bool, status *RepoStatus) *BranchWarning {
	if !checkout || status == nil || !isDirtySummary(status.Summary) || allowDirty {
		return nil
	}
	return &BranchWarning{
		Message:              "Working tree has uncommitted changes",
		RequiresConfirmation: true,
		DirtySummary:         &status.Summary,
	}
}

// resolveFromBranch resolves the base branch for creation.
func resolveFromBranch(reqFrom string, status *RepoStatus) string {
	from := strings.TrimSpace(reqFrom)
	if from == "" && status != nil {
		from = status.Branch.Head
	}
	return from
}

// finalizeCreateBranch handles checkout and populates the response after branch creation.
func finalizeCreateBranch(ctx context.Context, deps BranchDeps, resp *BranchCreateResponse, repoDir, name string, checkout bool) (*BranchCreateResponse, error) {
	if !checkout {
		resp.Success = true
		resp.Branch = &BranchInfo{Name: name}
		return resp, nil
	}
	if err := deps.Git.CheckoutBranch(ctx, repoDir, name); err != nil {
		resp.Success = false
		resp.Error = err.Error()
		return resp, nil
	}
	if st, err := GetRepoStatus(ctx, RepoStatusDeps{Git: deps.Git, RepoDir: repoDir}); err == nil {
		resp.Branch = branchInfoFromStatus(st)
	}
	resp.Success = true
	return resp, nil
}

// SwitchBranch changes the current branch.
func SwitchBranch(ctx context.Context, deps BranchDeps, req SwitchBranchRequest) (*BranchSwitchResponse, error) {
	resp := &BranchSwitchResponse{Timestamp: time.Now().UTC()}
	repoDir, err := validateBranchDeps(deps)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		resp.Success = false
		resp.Error = "branch name is required"
		return resp, nil
	}

	if warning, err := checkDirtyWorktree(ctx, deps, repoDir, req.AllowDirty); err != nil {
		return nil, err
	} else if warning != nil {
		resp.Success = false
		resp.Warning = warning
		return resp, nil
	}

	refs, err := listParsedBranches(ctx, deps, repoDir)
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

	return switchToRemoteBranch(ctx, deps, refs, name, req.TrackRemote, repoDir, resp)
}

// switchToRemoteBranch attempts to track and switch to a remote branch.
func switchToRemoteBranch(ctx context.Context, deps BranchDeps, refs []ParsedBranchRef, name string, trackRemote bool, repoDir string, resp *BranchSwitchResponse) (*BranchSwitchResponse, error) {
	remoteRef, ok := findRemoteBranch(refs, name)
	if !ok {
		resp.Success = false
		resp.Error = "branch not found"
		return resp, nil
	}
	if !trackRemote {
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

// Helper functions are in branch_service_helpers.go
