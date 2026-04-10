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

// GetDirectoryContents returns immediate children of a directory
func GetDirectoryContents(ctx context.Context, deps FileDeps, dirPath string) (*DirListResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	fullPath, err := resolveDirectoryPath(repoDir, dirPath)
	if err != nil {
		return nil, err
	}

	dirEntries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory not found: %s", dirPath)
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	trackedFiles, trackedPrefixes := buildTrackedSets(ctx, deps, repoDir)
	entries := buildDirEntries(ctx, dirEntries, dirPath, trackedFiles, trackedPrefixes)

	// Sort entries: folders first, then alphabetically
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	return &DirListResponse{
		Path:      dirPath,
		Entries:   entries,
		Timestamp: time.Now().UTC(),
	}, nil
}

// resolveDirectoryPath validates and resolves a relative directory path within a repo.
func resolveDirectoryPath(repoDir, dirPath string) (string, error) {
	if dirPath == "" {
		return repoDir, nil
	}
	depth := strings.Count(dirPath, "/") + 1
	if depth > MaxDirDepth {
		return "", fmt.Errorf("path exceeds maximum depth of %d", MaxDirDepth)
	}
	cleaned := filepath.Clean(dirPath)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("invalid path: %s", dirPath)
	}
	return filepath.Join(repoDir, cleaned), nil
}

// buildTrackedSets returns sets for tracked file paths and tracked directory prefixes.
func buildTrackedSets(ctx context.Context, deps FileDeps, repoDir string) (map[string]bool, map[string]bool) {
	trackedFiles := make(map[string]bool)
	trackedPrefixes := make(map[string]bool)
	trackedList, err := deps.Git.ListTrackedFiles(ctx, repoDir)
	if err != nil {
		return trackedFiles, trackedPrefixes
	}
	for _, f := range trackedList {
		trackedFiles[f] = true
		parts := strings.Split(f, "/")
		for i := 1; i < len(parts); i++ {
			prefix := strings.Join(parts[:i], "/")
			trackedPrefixes[prefix] = true
		}
	}
	return trackedFiles, trackedPrefixes
}

// buildDirEntries converts os.DirEntry items into DirEntry structs with tracking info.
func buildDirEntries(ctx context.Context, dirEntries []os.DirEntry, dirPath string, trackedFiles, trackedPrefixes map[string]bool) []DirEntry {
	entries := make([]DirEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if ctx.Err() != nil {
			break
		}
		name := de.Name()
		if name == ".git" {
			continue
		}

		entryPath := name
		if dirPath != "" {
			entryPath = dirPath + "/" + name
		}

		entry := DirEntry{
			Name:  name,
			Path:  entryPath,
			IsDir: de.IsDir(),
		}
		if de.IsDir() {
			entry.Tracked = trackedPrefixes[entryPath]
		} else {
			entry.Language = LanguageFromExtension(name)
			entry.Tracked = trackedFiles[entryPath]
		}
		entries = append(entries, entry)
	}
	return entries
}

// DeletePath removes a file or directory from the filesystem
// This is a filesystem delete, NOT a git rm. Tracked files will show as "deleted" in git status.
func DeletePath(ctx context.Context, deps FileDeps, req DeletePathRequest) (*DeletePathResponse, error) {
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	reqPath := strings.TrimSpace(req.Path)
	if errMsg := validateDeletePath(reqPath); errMsg != "" {
		return deleteFailResponse(reqPath, false, errMsg), nil
	}

	cleaned := filepath.Clean(reqPath)
	fullPath := filepath.Join(repoDir, cleaned)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return deleteFailResponse(reqPath, false, "path does not exist"), nil
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	isDir := info.IsDir()
	if isDir {
		err = os.RemoveAll(fullPath)
	} else {
		err = os.Remove(fullPath)
	}
	if err != nil {
		return deleteFailResponse(reqPath, isDir, fmt.Sprintf("failed to delete: %v", err)), nil
	}

	return &DeletePathResponse{
		Success:   true,
		Path:      reqPath,
		IsDir:     isDir,
		Timestamp: time.Now().UTC(),
	}, nil
}

func deleteFailResponse(path string, isDir bool, errMsg string) *DeletePathResponse {
	return &DeletePathResponse{
		Success:   false,
		Path:      path,
		IsDir:     isDir,
		Error:     errMsg,
		Timestamp: time.Now().UTC(),
	}
}

func validateDeletePath(reqPath string) string {
	if reqPath == "" {
		return "path is required"
	}
	cleaned := filepath.Clean(reqPath)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "invalid path: potential directory traversal"
	}
	cleanedLower := strings.ToLower(cleaned)
	for _, dangerous := range []string{".git", ".gitignore", ".gitattributes"} {
		if cleanedLower == dangerous || strings.HasPrefix(cleanedLower, dangerous+"/") {
			return "cannot delete protected path"
		}
	}
	return ""
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
