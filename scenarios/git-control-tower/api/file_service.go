package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/pathfilter"
)

// FileDeps contains dependencies for file operations
type FileDeps struct {
	Git     GitRunner
	RepoDir string
}

// GetFileTree returns a list of files in the repository
func GetFileTree(ctx context.Context, deps FileDeps, req FileTreeRequest) (*FileTreeResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	// Apply defaults and limits
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultFileTreeLimit
	}
	if limit > MaxFileTreeLimit {
		limit = MaxFileTreeLimit
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = DefaultFileTreeTimeout
	}
	if timeout > MaxFileTreeTimeout {
		timeout = MaxFileTreeTimeout
	}

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	var files []FileInfo
	var truncated, cancelled bool
	searchMode := "default"

	if req.Deep {
		searchMode = "deep"
		files, truncated, cancelled = getDeepFileTree(timeoutCtx, repoDir, req.Pattern, limit)
	} else {
		files, truncated, cancelled = getDefaultFileTree(timeoutCtx, deps, req.Pattern, limit)
	}

	return &FileTreeResponse{
		Files:      files,
		Truncated:  truncated,
		Cancelled:  cancelled,
		SearchMode: searchMode,
		Timestamp:  time.Now().UTC(),
	}, nil
}

// getDefaultFileTree uses git ls-files to get tracked + untracked files
func getDefaultFileTree(ctx context.Context, deps FileDeps, pattern string, limit int) ([]FileInfo, bool, bool) {
	files := make([]FileInfo, 0, limit)
	seenPaths := make(map[string]bool)

	// Get tracked files
	tracked, _ := deps.Git.ListTrackedFiles(ctx, deps.RepoDir)
	files, truncated, cancelled := collectFileInfos(ctx, files, seenPaths, tracked, pattern, limit, FileStatusTracked)
	if truncated || cancelled {
		return files, truncated, cancelled
	}

	// Get untracked files
	untracked, _ := deps.Git.ListUntrackedFiles(ctx, deps.RepoDir)
	files, truncated, cancelled = collectFileInfos(ctx, files, seenPaths, untracked, pattern, limit, FileStatusUntracked)
	if truncated || cancelled {
		return files, truncated, cancelled
	}

	// Sort files by path for consistent output
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, false, false
}

// collectFileInfos appends matching paths to files, returning early on limit or cancellation.
func collectFileInfos(ctx context.Context, files []FileInfo, seen map[string]bool, paths []string, pattern string, limit int, status FileStatus) ([]FileInfo, bool, bool) {
	for _, path := range paths {
		if ctx.Err() != nil {
			return files, false, true
		}
		if len(files) >= limit {
			return files, true, false
		}
		if matchesPattern(path, pattern) && !seen[path] {
			seen[path] = true
			files = append(files, FileInfo{
				Path:     path,
				Language: LanguageFromExtension(path),
				Status:   status,
			})
		}
	}
	return files, false, false
}

// getDeepFileTree walks the entire directory tree
func getDeepFileTree(ctx context.Context, repoDir string, pattern string, limit int) ([]FileInfo, bool, bool) {
	files := make([]FileInfo, 0, limit)

	err := filepath.WalkDir(repoDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		if ctx.Err() != nil {
			return filepath.SkipAll
		}

		if d.IsDir() {
			if pathfilter.SkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check limit
		if len(files) >= limit {
			return filepath.SkipAll
		}

		// Get relative path
		relPath, err := filepath.Rel(repoDir, path)
		if err != nil {
			return nil
		}

		// Skip if doesn't match pattern
		if !matchesPattern(relPath, pattern) {
			return nil
		}

		files = append(files, FileInfo{
			Path:     relPath,
			Language: LanguageFromExtension(relPath),
			Status:   FileStatusTracked, // We don't know the exact status in deep mode
		})

		return nil
	})

	cancelled := ctx.Err() != nil
	truncated := len(files) >= limit && err == nil && !cancelled

	// Sort files by path
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, truncated, cancelled
}
