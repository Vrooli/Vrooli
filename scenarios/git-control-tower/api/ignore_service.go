package main

import (
	"context"
	"fmt"
	"os"
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

func findNearestGitignore(repoDir string, path string) (string, string, error) {
	repoRoot := filepath.Clean(repoDir)
	absPath := filepath.Join(repoRoot, path)
	dir := filepath.Dir(absPath)

	for {
		gitignorePath := filepath.Join(dir, ".gitignore")
		if _, err := os.Stat(gitignorePath); err == nil {
			return gitignorePath, dir, nil
		} else if !os.IsNotExist(err) {
			return "", "", fmt.Errorf("stat .gitignore: %w", err)
		}

		if dir == repoRoot {
			return gitignorePath, dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", "", fmt.Errorf("gitignore resolution failed")
}

func ignoreEntryForPath(repoDir string, gitignoreDir string, path string) (string, error) {
	repoRoot := filepath.Clean(repoDir)
	absPath := filepath.Join(repoRoot, path)
	rel, err := filepath.Rel(gitignoreDir, absPath)
	if err != nil {
		return "", fmt.Errorf("resolve ignore entry: %w", err)
	}
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid ignore entry for %q", path)
	}
	return rel, nil
}

func ensureIgnoreEntries(gitignorePath string, entries []string) error {
	var content string
	raw, err := os.ReadFile(gitignorePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read .gitignore: %w", err)
		}
	} else {
		content = string(raw)
	}

	existing := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		normalized := strings.TrimPrefix(trimmed, "/")
		existing[normalized] = true
	}

	var toAdd []string
	for _, entry := range entries {
		normalized := strings.TrimPrefix(strings.TrimSpace(entry), "/")
		if normalized == "" {
			continue
		}
		if existing[normalized] {
			continue
		}
		existing[normalized] = true
		toAdd = append(toAdd, normalized)
	}

	if len(toAdd) == 0 {
		return nil
	}

	var builder strings.Builder
	builder.WriteString(content)
	if content != "" && !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}
	for _, entry := range toAdd {
		builder.WriteString(entry)
		builder.WriteString("\n")
	}

	if err := os.WriteFile(gitignorePath, []byte(builder.String()), 0o644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}
