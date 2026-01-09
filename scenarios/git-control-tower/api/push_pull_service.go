package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PushPullDeps contains dependencies for push/pull operations.
type PushPullDeps struct {
	Git     GitRunner
	RepoDir string
}

// PushToRemote pushes commits to the remote repository.
func PushToRemote(ctx context.Context, deps PushPullDeps, req PushRequest) (*PushResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	remote := strings.TrimSpace(req.Remote)

	// Get current branch if not specified
	branch := req.Branch
	remoteFromUpstream := ""
	branchFromUpstream := ""
	if branch == "" {
		status, err := GetRepoStatus(ctx, RepoStatusDeps{
			Git:     deps.Git,
			RepoDir: repoDir,
		})
		if err == nil {
			remoteFromUpstream, branchFromUpstream = parseUpstream(status.Branch.Upstream)
			if branchFromUpstream != "" {
				branch = branchFromUpstream
			} else {
				branch = status.Branch.Head
			}
		}
	}

	branch = strings.TrimSpace(branch)
	if remote == "" && remoteFromUpstream != "" {
		remote = remoteFromUpstream
	}
	if branch == "" && branchFromUpstream != "" {
		branch = branchFromUpstream
	}
	if remote == "" {
		remote = "origin"
	}
	if branch == "" {
		return &PushResponse{
			Success:   false,
			Remote:    remote,
			Branch:    branch,
			Error:     "branch could not be resolved for push",
			Timestamp: time.Now().UTC(),
		}, nil
	}

	headOID, headErr := resolveRefOID(ctx, deps, repoDir, "HEAD")
	if headErr != nil {
		return &PushResponse{
			Success:   false,
			Remote:    remote,
			Branch:    branch,
			Error:     headErr.Error(),
			Timestamp: time.Now().UTC(),
		}, nil
	}
	preRemoteOID, preRemoteKnown := resolveRemoteOID(ctx, deps, repoDir, remote, branch)

	err := deps.Git.Push(ctx, repoDir, remote, branch, req.SetUpstream)
	if err != nil {
		return &PushResponse{
			Success:   false,
			Remote:    remote,
			Branch:    branch,
			Error:     err.Error(),
			Timestamp: time.Now().UTC(),
		}, nil
	}

	resp := &PushResponse{
		Success:   true,
		Remote:    remote,
		Branch:    branch,
		Timestamp: time.Now().UTC(),
	}

	verifyPushResult(ctx, deps, repoDir, remote, branch, headOID, preRemoteOID, preRemoteKnown, resp)

	return resp, nil
}

// PullFromRemote pulls commits from the remote repository.
func PullFromRemote(ctx context.Context, deps PushPullDeps, req PullRequest) (*PullResponse, error) {
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

	branch := req.Branch

	err := deps.Git.Pull(ctx, repoDir, remote, branch)
	if err != nil {
		errStr := err.Error()
		hasConflicts := strings.Contains(errStr, "CONFLICT") || strings.Contains(errStr, "conflict")

		return &PullResponse{
			Success:      false,
			Remote:       remote,
			Branch:       branch,
			Error:        errStr,
			HasConflicts: hasConflicts,
			Timestamp:    time.Now().UTC(),
		}, nil
	}

	return &PullResponse{
		Success:   true,
		Remote:    remote,
		Branch:    branch,
		Timestamp: time.Now().UTC(),
	}, nil
}

func resolveRefOID(ctx context.Context, deps PushPullDeps, repoDir string, ref string) (string, error) {
	out, err := deps.Git.RevParse(ctx, repoDir, ref)
	if err != nil {
		return "", fmt.Errorf("git rev-parse %s failed: %w", ref, err)
	}
	oid := strings.TrimSpace(string(out))
	if oid == "" {
		return "", fmt.Errorf("git rev-parse %s returned empty output", ref)
	}
	return oid, nil
}

func resolveRemoteOID(
	ctx context.Context,
	deps PushPullDeps,
	repoDir string,
	remote string,
	branch string,
) (string, bool) {
	ref := buildRemoteRef(remote, branch)
	if ref == "" {
		return "", false
	}
	out, err := deps.Git.RevParse(ctx, repoDir, ref)
	if err != nil {
		return "", false
	}
	oid := strings.TrimSpace(string(out))
	if oid == "" {
		return "", false
	}
	return oid, true
}

func buildRemoteRef(remote string, branch string) string {
	remote = strings.TrimSpace(remote)
	branch = strings.TrimSpace(branch)
	branch = strings.TrimPrefix(branch, "refs/heads/")
	if remote == "" || branch == "" {
		return ""
	}
	if strings.HasPrefix(branch, "refs/remotes/") {
		return strings.TrimPrefix(branch, "refs/remotes/")
	}
	if strings.HasPrefix(branch, remote+"/") {
		return branch
	}
	return fmt.Sprintf("%s/%s", remote, branch)
}

func parseUpstream(upstream string) (string, string) {
	trimmed := strings.TrimSpace(upstream)
	if trimmed == "" {
		return "", ""
	}
	trimmed = strings.TrimPrefix(trimmed, "refs/remotes/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", ""
	}
	remote := parts[0]
	branch := strings.Join(parts[1:], "/")
	return remote, branch
}

func verifyPushResult(
	ctx context.Context,
	deps PushPullDeps,
	repoDir string,
	remote string,
	branch string,
	headOID string,
	preRemoteOID string,
	preRemoteKnown bool,
	resp *PushResponse,
) {
	if resp == nil {
		return
	}

	if err := deps.Git.FetchRemote(ctx, repoDir, remote); err != nil {
		resp.VerificationError = err.Error()
		resp.Verified = false
		return
	}

	postRemoteOID, postRemoteKnown := resolveRemoteOID(ctx, deps, repoDir, remote, branch)
	if !postRemoteKnown {
		resp.VerificationError = "unable to resolve remote ref after push"
		resp.Verified = false
		return
	}

	resp.Verified = true
	if postRemoteOID == headOID {
		resp.UpToDate = preRemoteKnown && preRemoteOID == headOID
		resp.Pushed = !resp.UpToDate
		return
	}

	resp.Success = false
	resp.Error = "push completed but remote ref did not update"
}
