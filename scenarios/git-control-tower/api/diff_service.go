package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// DiffDeps contains dependencies for diff operations
type DiffDeps struct {
	Git     GitRunner
	RepoDir string
}

// GetDiff retrieves and parses a git diff
func GetDiff(ctx context.Context, deps DiffDeps, req DiffRequest) (*DiffResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	// Normalize the view mode - default to diff if not specified
	mode := req.Mode
	if mode == "" {
		mode = ViewModeDiff
	}

	// Handle source mode - just return the file content without diff
	// Also use source mode when any=true (viewing non-changed files)
	if mode == ViewModeSource || req.Any {
		return getSourceContent(ctx, deps, req, repoDir)
	}

	// Handle commit-specific diff (history mode)
	if req.Commit != "" {
		return getCommitDiff(ctx, deps, req, repoDir, mode)
	}

	// Handle untracked files
	if req.Untracked {
		return getUntrackedContent(ctx, deps, req, repoDir, mode)
	}

	// Standard diff for tracked files
	return getTrackedDiff(ctx, deps, req, repoDir, mode)
}

// loadFileContent reads file content from a commit or working directory and
// returns the text (or base64 for images) along with its binary classification.
func loadFileContent(ctx context.Context, deps DiffDeps, repoDir, cleanPath, commit string) (string, binaryKind, error) {
	var data []byte
	if commit != "" {
		out, err := deps.Git.ShowFileAtCommit(ctx, repoDir, commit, cleanPath)
		if err != nil {
			return "", binaryNone, fmt.Errorf("show file at commit: %w", err)
		}
		if int64(len(out)) > maxDiffFileBytes {
			return "", binaryNone, &FileTooLargeError{Path: cleanPath, Size: int64(len(out)), Limit: maxDiffFileBytes}
		}
		data = out
	} else {
		absPath := filepath.Join(repoDir, cleanPath)
		d, _, err := readFileForDisplay(absPath, cleanPath)
		if err != nil {
			return "", binaryNone, fmt.Errorf("read file: %w", err)
		}
		data = d
	}

	kind := detectBinaryKind(cleanPath, data)
	if kind == binaryUnsupported {
		return "", kind, &UnsupportedBinaryError{Path: cleanPath}
	}
	if kind == binaryImage {
		return base64.StdEncoding.EncodeToString(data), kind, nil
	}
	return string(data), kind, nil
}

// buildSourceAnnotatedLines creates annotated lines for source view (no change markers).
// Returns the annotated lines and the effective line count (excluding trailing empty line).
func buildSourceAnnotatedLines(content string) ([]AnnotatedLine, int) {
	lines := strings.Split(content, "\n")
	annotated := make([]AnnotatedLine, len(lines))
	for i, line := range lines {
		annotated[i] = AnnotatedLine{
			Number:  i + 1,
			Content: line,
			Change:  LineChangeNone,
		}
	}
	lineCount := len(lines)
	if lineCount > 0 && lines[lineCount-1] == "" {
		lineCount--
	}
	return annotated, lineCount
}

// computeSourceDiffStats returns diff stats for source mode, either from untracked status or by running git diff.
func computeSourceDiffStats(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir, cleanPath string, bk binaryKind, lineCount int) (bool, DiffStats, error) {
	if req.Untracked {
		stats := DiffStats{Files: 1}
		if bk == binaryNone {
			stats.Additions = lineCount
		}
		return true, stats, nil
	}

	var diffOut []byte
	var diffErr error
	if req.Commit != "" {
		diffOut, diffErr = deps.Git.ShowCommitDiff(ctx, repoDir, req.Commit, cleanPath)
	} else {
		diffOut, diffErr = deps.Git.Diff(ctx, repoDir, cleanPath, req.Staged)
	}
	if diffErr != nil {
		return false, DiffStats{}, fmt.Errorf("check diff for source content: %w", diffErr)
	}
	parsed := ParseDiffOutput(string(diffOut))
	enrichCommentStats(parsed, cleanPath)
	return parsed.HasDiff, parsed.Stats, nil
}

// getSourceContent returns just the file content without any diff information
func getSourceContent(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir string) (*DiffResponse, error) {
	cleanPath := cleanFilePath(req.Path)
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	content, bk, err := loadFileContent(ctx, deps, repoDir, cleanPath, req.Commit)
	if err != nil {
		return nil, err
	}

	var annotatedLines []AnnotatedLine
	lineCount := 0
	if bk == binaryNone {
		annotatedLines, lineCount = buildSourceAnnotatedLines(content)
	}

	hasDiff, stats, err := computeSourceDiffStats(ctx, deps, req, repoDir, cleanPath, bk, lineCount)
	if err != nil {
		return nil, err
	}

	return &DiffResponse{
		RepoDir:        repoDir,
		Path:           cleanPath,
		Staged:         req.Staged,
		Untracked:      req.Untracked,
		HasDiff:        hasDiff,
		Stats:          stats,
		FullContent:    content,
		AnnotatedLines: annotatedLines,
		Mode:           ViewModeSource,
		Timestamp:      time.Now().UTC(),
		ContentHash: func() string {
			if bk != binaryNone {
				return ""
			}
			return hashContentBytes([]byte(content))
		}(),
	}, nil
}

// applyFullDiffContent sets FullContent, ContentHash, and AnnotatedLines on the
// response using raw file bytes. Returns an error for unsupported binary files.
func applyFullDiffContent(resp *DiffResponse, cleanPath string, content []byte) error {
	if int64(len(content)) > maxDiffFileBytes {
		return &FileTooLargeError{Path: cleanPath, Size: int64(len(content)), Limit: maxDiffFileBytes}
	}
	bk := detectBinaryKind(cleanPath, content)
	if bk == binaryUnsupported {
		return &UnsupportedBinaryError{Path: cleanPath}
	}
	if bk == binaryImage {
		resp.FullContent = base64.StdEncoding.EncodeToString(content)
	} else {
		resp.FullContent = string(content)
		resp.ContentHash = hashContentBytes(content)
		resp.AnnotatedLines = buildAnnotatedLines(resp.FullContent, resp.Hunks)
	}
	return nil
}

// getCommitDiff handles diff for a specific commit
func getCommitDiff(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir string, mode ViewMode) (*DiffResponse, error) {
	cleanPath := cleanFilePath(req.Path)
	if req.Path != "" && (cleanPath == "" || strings.HasPrefix(cleanPath, "..")) {
		return nil, fmt.Errorf("invalid path")
	}

	out, err := deps.Git.ShowCommitDiff(ctx, repoDir, req.Commit, cleanPath)
	if err != nil {
		return nil, err
	}

	parsed := ParseDiffOutput(string(out))
	enrichCommentStats(parsed, cleanPath)
	parsed.RepoDir = repoDir
	parsed.Path = cleanPath
	parsed.Staged = false
	parsed.Untracked = false
	parsed.Base = req.Commit
	parsed.Mode = mode
	parsed.Timestamp = time.Now().UTC()

	if mode.needsFullContent() && req.Path != "" {
		content, err := deps.Git.ShowFileAtCommit(ctx, repoDir, req.Commit, cleanPath)
		if err != nil {
			return nil, err
		}
		if err := applyFullDiffContent(parsed, cleanPath, content); err != nil {
			return nil, err
		}
	}

	return parsed, nil
}

// buildUntrackedAnnotatedLines creates annotated lines for untracked files where all lines are "added".
func buildUntrackedAnnotatedLines(lines []string, mode ViewMode) []AnnotatedLine {
	change := LineChangeAdded
	if mode == ViewModeSource {
		change = LineChangeNone
	}
	annotated := make([]AnnotatedLine, len(lines))
	for i, line := range lines {
		annotated[i] = AnnotatedLine{
			Number:  i + 1,
			Content: line,
			Change:  change,
		}
	}
	return annotated
}

// getUntrackedContent handles untracked files
func getUntrackedContent(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir string, mode ViewMode) (*DiffResponse, error) {
	cleanPath := cleanFilePath(req.Path)
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}
	absPath := filepath.Join(repoDir, cleanPath)
	content, _, err := readFileForDisplay(absPath, cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	bk := detectBinaryKind(cleanPath, content)
	if bk == binaryUnsupported {
		return nil, &UnsupportedBinaryError{Path: cleanPath}
	}

	fileText, lines, lineCount := textOrBase64(content, cleanPath, bk)

	var annotatedLines []AnnotatedLine
	if bk == binaryNone {
		annotatedLines = buildUntrackedAnnotatedLines(lines, mode)
	}

	return &DiffResponse{
		RepoDir:        repoDir,
		Path:           cleanPath,
		Staged:         false,
		Untracked:      true,
		HasDiff:        true,
		Stats:          DiffStats{Additions: lineCount, Deletions: 0, Files: 1},
		FullContent:    fileText,
		AnnotatedLines: annotatedLines,
		Mode:           mode,
		Timestamp:      time.Now().UTC(),
		ContentHash: func() string {
			if bk != binaryNone {
				return ""
			}
			return hashContentBytes(content)
		}(),
	}, nil
}

// textOrBase64 converts raw bytes to display text. For images it returns base64.
// Returns the display text, split lines (nil for images), and effective line count.
func textOrBase64(data []byte, _ string, bk binaryKind) (string, []string, int) {
	if bk == binaryImage {
		return base64.StdEncoding.EncodeToString(data), nil, 0
	}
	text := string(data)
	lines := strings.Split(text, "\n")
	lineCount := len(lines)
	if lineCount > 0 && lines[lineCount-1] == "" {
		lineCount--
	}
	return text, lines, lineCount
}

// getTrackedDiff handles diff for tracked files (staged or unstaged)
func getTrackedDiff(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir string, mode ViewMode) (*DiffResponse, error) {
	pathForGit := strings.TrimSpace(req.Path)
	if pathForGit != "" {
		cleanPath := cleanFilePath(pathForGit)
		if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
			return nil, fmt.Errorf("invalid path")
		}
		if err := ensureFileWithinLimit(repoDir, cleanPath); err != nil {
			return nil, err
		}
		pathForGit = cleanPath
	}

	out, err := deps.Git.Diff(ctx, repoDir, pathForGit, req.Staged)
	if err != nil {
		return nil, err
	}

	parsed := ParseDiffOutput(string(out))
	enrichCommentStats(parsed, pathForGit)
	parsed.RepoDir = repoDir
	parsed.Path = pathForGit
	parsed.Staged = req.Staged
	parsed.Untracked = req.Untracked
	parsed.Base = req.Base
	parsed.Mode = mode
	parsed.Timestamp = time.Now().UTC()

	if mode.needsFullContent() && req.Path != "" {
		absPath := filepath.Join(repoDir, pathForGit)
		content, _, err := readFileForDisplay(absPath, pathForGit)
		if err != nil {
			return nil, err
		}
		if err := applyFullDiffContent(parsed, pathForGit, content); err != nil {
			return nil, err
		}
	}

	return parsed, nil
}
