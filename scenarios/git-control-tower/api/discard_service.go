package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DiscardDeps contains dependencies for discard operations.
type DiscardDeps struct {
	Git     GitRunner
	RepoDir string
}

// DiscardFiles discards changes for the specified files.
// For tracked files (untracked=false), this reverts modifications using git checkout.
// For untracked files (untracked=true), this deletes them using git clean.
func DiscardFiles(ctx context.Context, deps DiscardDeps, req DiscardRequest) (*DiscardResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	if len(req.Paths) == 0 {
		return &DiscardResponse{
			Success:   true,
			Discarded: []string{},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Validate paths are within repo (no directory traversal)
	validPaths := []string{}
	for _, p := range req.Paths {
		cleanPath := cleanFilePath(p)
		if cleanPath != "" && !strings.HasPrefix(cleanPath, "..") {
			validPaths = append(validPaths, cleanPath)
		}
	}

	if len(validPaths) == 0 {
		return &DiscardResponse{
			Success:   false,
			Failed:    req.Paths,
			Errors:    []string{"no valid paths provided"},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	err := deps.Git.Discard(ctx, repoDir, validPaths, req.Untracked)
	if err != nil {
		return &DiscardResponse{
			Success:   false,
			Failed:    validPaths,
			Errors:    []string{err.Error()},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	return &DiscardResponse{
		Success:   true,
		Discarded: validPaths,
		Timestamp: time.Now().UTC(),
	}, nil
}
