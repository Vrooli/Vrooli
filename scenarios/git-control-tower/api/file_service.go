package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	tracked, err := deps.Git.ListTrackedFiles(ctx, deps.RepoDir)
	if err == nil {
		for _, path := range tracked {
			if ctx.Err() != nil {
				return files, false, true // Cancelled
			}
			if len(files) >= limit {
				return files, true, false // Truncated
			}
			if matchesPattern(path, pattern) && !seenPaths[path] {
				seenPaths[path] = true
				files = append(files, FileInfo{
					Path:     path,
					Language: LanguageFromExtension(path),
					Status:   FileStatusTracked,
				})
			}
		}
	}

	// Get untracked files
	untracked, err := deps.Git.ListUntrackedFiles(ctx, deps.RepoDir)
	if err == nil {
		for _, path := range untracked {
			if ctx.Err() != nil {
				return files, false, true // Cancelled
			}
			if len(files) >= limit {
				return files, true, false // Truncated
			}
			if matchesPattern(path, pattern) && !seenPaths[path] {
				seenPaths[path] = true
				files = append(files, FileInfo{
					Path:     path,
					Language: LanguageFromExtension(path),
					Status:   FileStatusUntracked,
				})
			}
		}
	}

	// Sort files by path for consistent output
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, false, false
}

// getDeepFileTree walks the entire directory tree
func getDeepFileTree(ctx context.Context, repoDir string, pattern string, limit int) ([]FileInfo, bool, bool) {
	files := make([]FileInfo, 0, limit)

	// Directories to exclude from deep search
	excludeDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"__pycache__":  true,
		".next":        true,
		".nuxt":        true,
		"dist":         true,
		"build":        true,
		"target":       true,
		".cache":       true,
	}

	err := filepath.WalkDir(repoDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		// Check for cancellation
		if ctx.Err() != nil {
			return filepath.SkipAll
		}

		// Skip excluded directories
		if d.IsDir() {
			if excludeDirs[d.Name()] {
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

// matchesPattern checks if a path matches a glob pattern
func matchesPattern(path, pattern string) bool {
	if pattern == "" {
		return true
	}

	// Support simple glob patterns
	matched, err := filepath.Match(pattern, filepath.Base(path))
	if err == nil && matched {
		return true
	}

	// Also try matching against the full path
	matched, err = filepath.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// Support partial string matching for fuzzy search
	return strings.Contains(strings.ToLower(path), strings.ToLower(pattern))
}
