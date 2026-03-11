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
		return &IgnoreResponse{
			Success:   false,
			Failed:    []string{req.Path},
			Errors:    []string{"no valid path provided"},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	var gitignorePath string
	var entry string

	switch req.Level {
	case "group":
		groupDir := strings.TrimSpace(req.GroupDir)
		if groupDir == "" {
			return &IgnoreResponse{
				Success:   false,
				Failed:    []string{cleanPath},
				Errors:    []string{"group_dir is required when level is group"},
				Timestamp: time.Now().UTC(),
			}, nil
		}
		if !isCleanSubpath(groupDir) {
			return &IgnoreResponse{
				Success:   false,
				Failed:    []string{cleanPath},
				Errors:    []string{"invalid group_dir"},
				Timestamp: time.Now().UTC(),
			}, nil
		}
		gitignorePath = filepath.Join(repoDir, groupDir, ".gitignore")

		// Strip the groupDir prefix from the path to get the relative entry.
		normalizedGroup := normalizePrefix(groupDir)
		if strings.HasPrefix(cleanPath, normalizedGroup) {
			entry = cleanPath[len(normalizedGroup):]
		} else {
			entry = cleanPath
		}
		if entry == "" {
			return &IgnoreResponse{
				Success:   false,
				Failed:    []string{cleanPath},
				Errors:    []string{"path does not fall under group_dir"},
				Timestamp: time.Now().UTC(),
			}, nil
		}

	default:
		// project level: write to root .gitignore
		gitignorePath = filepath.Join(repoDir, ".gitignore")
		entry = cleanPath
	}

	fs := deps.FS
	if fs == nil {
		fs = OSFileIO{}
	}

	if err := ensureIgnoreEntriesFS(fs, gitignorePath, []string{entry}); err != nil {
		return &IgnoreResponse{
			Success:   false,
			Failed:    []string{cleanPath},
			Errors:    []string{err.Error()},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	if err := deps.Git.RemoveFromIndex(ctx, repoDir, []string{cleanPath}); err != nil {
		return &IgnoreResponse{
			Success:   false,
			Failed:    []string{cleanPath},
			Errors:    []string{err.Error()},
			Timestamp: time.Now().UTC(),
		}, nil
	}

	return &IgnoreResponse{
		Success:       true,
		Ignored:       []string{cleanPath},
		GitignorePath: gitignorePath,
		Timestamp:     time.Now().UTC(),
	}, nil
}
