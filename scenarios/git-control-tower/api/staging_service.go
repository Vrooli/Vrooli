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

// validateStagingDeps validates common staging operation preconditions and resolves paths.
// Returns (repoDir, validPaths, originalPaths, error). If originalPaths is empty,
// the caller should return a success-with-empty-list response.
func validateStagingDeps(deps StagingDeps, paths []string, scope string) (string, []string, []string, error) {
	if deps.Git == nil {
		return "", nil, nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return "", nil, nil, fmt.Errorf("repo dir is required")
	}

	if scope != "" && len(paths) == 0 {
		paths = expandScope(scope)
	}

	if len(paths) == 0 {
		return repoDir, nil, nil, nil
	}

	validPaths := filterValidPaths(paths)
	return repoDir, validPaths, paths, nil
}

func filterValidPaths(paths []string) []string {
	var valid []string
	for _, p := range paths {
		cleanPath := cleanFilePath(p)
		if cleanPath != "" && !strings.HasPrefix(cleanPath, "..") {
			valid = append(valid, cleanPath)
		}
	}
	return valid
}

// StageFiles stages the specified files
func StageFiles(ctx context.Context, deps StagingDeps, req StageRequest) (*StageResponse, error) {
	repoDir, validPaths, origPaths, err := validateStagingDeps(deps, req.Paths, req.Scope)
	if err != nil {
		return nil, err
	}

	if origPaths == nil {
		return &StageResponse{Success: true, Staged: []string{}, Timestamp: time.Now().UTC()}, nil
	}

	if len(validPaths) == 0 {
		return &StageResponse{Success: false, Failed: origPaths, Errors: []string{"no valid paths provided"}, Timestamp: time.Now().UTC()}, nil
	}

	warnings, stageErr := deps.Git.Stage(ctx, repoDir, validPaths)
	if stageErr != nil {
		return &StageResponse{Success: false, Failed: validPaths, Errors: []string{stageErr.Error()}, Warnings: warnings, Timestamp: time.Now().UTC()}, nil
	}

	return &StageResponse{Success: true, Staged: validPaths, Warnings: warnings, Timestamp: time.Now().UTC()}, nil
}

// UnstageFiles unstages the specified files
func UnstageFiles(ctx context.Context, deps StagingDeps, req UnstageRequest) (*UnstageResponse, error) {
	repoDir, validPaths, origPaths, err := validateStagingDeps(deps, req.Paths, req.Scope)
	if err != nil {
		return nil, err
	}

	if origPaths == nil {
		return &UnstageResponse{Success: true, Unstaged: []string{}, Timestamp: time.Now().UTC()}, nil
	}

	if len(validPaths) == 0 {
		return &UnstageResponse{Success: false, Failed: origPaths, Errors: []string{"no valid paths provided"}, Timestamp: time.Now().UTC()}, nil
	}

	unstageErr := deps.Git.Unstage(ctx, repoDir, validPaths)
	if unstageErr != nil {
		return &UnstageResponse{Success: false, Failed: validPaths, Errors: []string{unstageErr.Error()}, Timestamp: time.Now().UTC()}, nil
	}

	return &UnstageResponse{Success: true, Unstaged: validPaths, Timestamp: time.Now().UTC()}, nil
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
