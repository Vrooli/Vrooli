package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// StagingDeps contains dependencies for staging operations
type StagingDeps struct {
	Git     GitRunner
	RepoDir string
}

// StageFiles stages the specified files
func StageFiles(ctx context.Context, deps StagingDeps, req StageRequest) (*StageResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	paths := req.Paths
	if req.Scope != "" && len(paths) == 0 {
		paths = expandScope(req.Scope)
	}

	if len(paths) == 0 {
		return &StageResponse{
			Success:   true,
			Staged:    []string{},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Validate paths are within repo
	validPaths := []string{}
	for _, p := range paths {
		cleanPath := cleanFilePath(p)
		if cleanPath != "" && !strings.HasPrefix(cleanPath, "..") {
			validPaths = append(validPaths, cleanPath)
		}
	}

	if len(validPaths) == 0 {
		return &StageResponse{
			Success:   false,
			Failed:    paths,
			Errors:    []string{"no valid paths provided"},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	err := deps.Git.Stage(ctx, repoDir, validPaths)
	if err != nil {
		return &StageResponse{
			Success:   false,
			Failed:    validPaths,
			Errors:    []string{err.Error()},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	return &StageResponse{
		Success:   true,
		Staged:    validPaths,
		Timestamp: time.Now().UTC(),
	}, nil
}

// UnstageFiles unstages the specified files
func UnstageFiles(ctx context.Context, deps StagingDeps, req UnstageRequest) (*UnstageResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	paths := req.Paths
	if req.Scope != "" && len(paths) == 0 {
		paths = expandScope(req.Scope)
	}

	if len(paths) == 0 {
		return &UnstageResponse{
			Success:   true,
			Unstaged:  []string{},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Validate paths are within repo
	validPaths := []string{}
	for _, p := range paths {
		cleanPath := cleanFilePath(p)
		if cleanPath != "" && !strings.HasPrefix(cleanPath, "..") {
			validPaths = append(validPaths, cleanPath)
		}
	}

	if len(validPaths) == 0 {
		return &UnstageResponse{
			Success:   false,
			Failed:    paths,
			Errors:    []string{"no valid paths provided"},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	err := deps.Git.Unstage(ctx, repoDir, validPaths)
	if err != nil {
		return &UnstageResponse{
			Success:   false,
			Failed:    validPaths,
			Errors:    []string{err.Error()},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	return &UnstageResponse{
		Success:   true,
		Unstaged:  validPaths,
		Timestamp: time.Now().UTC(),
	}, nil
}

// expandScope converts a scope (scenario:name, resource:name) to a glob pattern
func expandScope(scope string) []string {
	parts := strings.SplitN(scope, ":", 2)
	if len(parts) != 2 {
		return nil
	}

	scopeType := parts[0]
	scopeName := parts[1]

	switch scopeType {
	case "scenario":
		return []string{fmt.Sprintf("scenarios/%s/", scopeName)}
	case "resource":
		return []string{fmt.Sprintf("resources/%s/", scopeName)}
	case "package":
		return []string{fmt.Sprintf("packages/%s/", scopeName)}
	default:
		return nil
	}
}

// cleanFilePath sanitizes a file path for git operations
func cleanFilePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	// Clean the path to remove any .. traversals
	cleaned := filepath.Clean(path)
	// Remove leading slashes for relative paths
	cleaned = strings.TrimPrefix(cleaned, "/")
	return cleaned
}
