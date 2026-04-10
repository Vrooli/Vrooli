package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// IgnoreDeps contains dependencies for ignore operations.
type IgnoreDeps struct {
	Git     GitRunner
	FS      FileIO
	RepoDir string
}

func ignoreFailure(path string, errMsg string) *IgnoreResponse {
	return &IgnoreResponse{
		Success:   false,
		Failed:    []string{path},
		Errors:    []string{errMsg},
		Timestamp: time.Now().UTC(),
	}
}

// resolveGroupIgnore resolves the gitignore path and entry for group-level ignores.
func resolveGroupIgnore(repoDir, cleanPath, groupDirRaw string) (string, string, *IgnoreResponse) {
	groupDir := strings.TrimSpace(groupDirRaw)
	if groupDir == "" {
		return "", "", ignoreFailure(cleanPath, "group_dir is required when level is group")
	}
	if !isCleanSubpath(groupDir) {
		return "", "", ignoreFailure(cleanPath, "invalid group_dir")
	}

	gitignorePath := filepath.Join(repoDir, groupDir, ".gitignore")
	normalizedGroup := normalizePrefix(groupDir)
	entry := cleanPath
	if strings.HasPrefix(cleanPath, normalizedGroup) {
		entry = cleanPath[len(normalizedGroup):]
	}
	if entry == "" {
		return "", "", ignoreFailure(cleanPath, "path does not fall under group_dir")
	}
	return gitignorePath, entry, nil
}

// IgnorePath adds the path to the appropriate .gitignore and removes it from the index if tracked.
// When Level is "group", the entry is written to <RepoDir>/<GroupDir>/.gitignore.
// Otherwise it writes to the root .gitignore.
func IgnorePath(ctx context.Context, deps IgnoreDeps, req IgnoreRequest) (*IgnoreResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	cleanPath := cleanFilePath(req.Path)
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
		return ignoreFailure(req.Path, "no valid path provided"), nil
	}

	var gitignorePath, entry string
	if req.Level == "group" {
		var fail *IgnoreResponse
		gitignorePath, entry, fail = resolveGroupIgnore(repoDir, cleanPath, req.GroupDir)
		if fail != nil {
			return fail, nil
		}
	} else {
		gitignorePath = filepath.Join(repoDir, ".gitignore")
		entry = cleanPath
	}

	fs := deps.FS
	if fs == nil {
		fs = OSFileIO{}
	}

	if err := ensureIgnoreEntriesFS(fs, gitignorePath, []string{entry}); err != nil {
		return ignoreFailure(cleanPath, err.Error()), nil
	}

	if err := deps.Git.RemoveFromIndex(ctx, repoDir, []string{cleanPath}); err != nil {
		return ignoreFailure(cleanPath, err.Error()), nil
	}

	return &IgnoreResponse{
		Success:       true,
		Ignored:       []string{cleanPath},
		GitignorePath: gitignorePath,
		Timestamp:     time.Now().UTC(),
	}, nil
}
