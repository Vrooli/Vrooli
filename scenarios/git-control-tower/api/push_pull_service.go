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

	remote := req.Remote
	if remote == "" {
		remote = "origin"
	}

	// Get current branch if not specified
	branch := req.Branch
	if branch == "" {
		status, err := GetRepoStatus(ctx, RepoStatusDeps{
			Git:     deps.Git,
			RepoDir: repoDir,
		})
		if err == nil {
			branch = status.Branch.Head
		}
	}

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

	return &PushResponse{
		Success:   true,
		Remote:    remote,
		Branch:    branch,
		Timestamp: time.Now().UTC(),
	}, nil
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
