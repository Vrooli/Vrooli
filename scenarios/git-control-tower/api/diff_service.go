package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	if mode == ViewModeSource {
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

// getSourceContent returns just the file content without any diff information
func getSourceContent(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir string) (*DiffResponse, error) {
	cleanPath := cleanFilePath(req.Path)
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	var content string

	if req.Commit != "" {
		// Get file content at a specific commit
		out, err := deps.Git.ShowFileAtCommit(ctx, repoDir, req.Commit, cleanPath)
		if err != nil {
			return nil, fmt.Errorf("show file at commit: %w", err)
		}
		content = string(out)
	} else {
		// Get current file content from working directory
		absPath := filepath.Join(repoDir, cleanPath)
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
		content = string(data)
	}

	// Build annotated lines (all lines, no change markers)
	lines := strings.Split(content, "\n")
	annotatedLines := make([]AnnotatedLine, len(lines))
	for i, line := range lines {
		annotatedLines[i] = AnnotatedLine{
			Number:  i + 1,
			Content: line,
			Change:  LineChangeNone,
		}
	}

	return &DiffResponse{
		RepoDir:        repoDir,
		Path:           cleanPath,
		Staged:         req.Staged,
		Untracked:      req.Untracked,
		HasDiff:        false,
		Stats:          DiffStats{},
		FullContent:    content,
		AnnotatedLines: annotatedLines,
		Mode:           ViewModeSource,
		Timestamp:      time.Now().UTC(),
	}, nil
}

// getCommitDiff handles diff for a specific commit
func getCommitDiff(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir string, mode ViewMode) (*DiffResponse, error) {
	out, err := deps.Git.ShowCommitDiff(ctx, repoDir, req.Commit, req.Path)
	if err != nil {
		return nil, err
	}

	parsed := ParseDiffOutput(string(out))
	parsed.RepoDir = repoDir
	parsed.Path = req.Path
	parsed.Staged = false
	parsed.Untracked = false
	parsed.Base = req.Commit
	parsed.Mode = mode
	parsed.Timestamp = time.Now().UTC()

	// For full_diff mode, get the file content and annotate lines
	if mode == ViewModeFullDiff && req.Path != "" {
		cleanPath := cleanFilePath(req.Path)
		if cleanPath != "" && !strings.HasPrefix(cleanPath, "..") {
			content, err := deps.Git.ShowFileAtCommit(ctx, repoDir, req.Commit, cleanPath)
			if err == nil {
				parsed.FullContent = string(content)
				parsed.AnnotatedLines = buildAnnotatedLines(string(content), parsed.Hunks)
			}
		}
	}

	return parsed, nil
}

// getUntrackedContent handles untracked files
func getUntrackedContent(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir string, mode ViewMode) (*DiffResponse, error) {
	cleanPath := cleanFilePath(req.Path)
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}
	absPath := filepath.Join(repoDir, cleanPath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	fileText := string(content)
	lines := strings.Split(fileText, "\n")
	lineCount := len(lines)
	if lineCount > 0 && lines[lineCount-1] == "" {
		lineCount-- // Don't count empty trailing line
	}

	// For untracked files, all lines are "added"
	annotatedLines := make([]AnnotatedLine, len(lines))
	for i, line := range lines {
		change := LineChangeAdded
		if mode == ViewModeSource {
			change = LineChangeNone
		}
		annotatedLines[i] = AnnotatedLine{
			Number:  i + 1,
			Content: line,
			Change:  change,
		}
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
	}, nil
}

// getTrackedDiff handles diff for tracked files (staged or unstaged)
func getTrackedDiff(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir string, mode ViewMode) (*DiffResponse, error) {
	out, err := deps.Git.Diff(ctx, repoDir, req.Path, req.Staged)
	if err != nil {
		return nil, err
	}

	parsed := ParseDiffOutput(string(out))
	parsed.RepoDir = repoDir
	parsed.Path = req.Path
	parsed.Staged = req.Staged
	parsed.Untracked = req.Untracked
	parsed.Base = req.Base
	parsed.Mode = mode
	parsed.Timestamp = time.Now().UTC()

	// For full_diff mode, get the current file content and annotate lines
	if mode == ViewModeFullDiff && req.Path != "" {
		cleanPath := cleanFilePath(req.Path)
		if cleanPath != "" && !strings.HasPrefix(cleanPath, "..") {
			absPath := filepath.Join(repoDir, cleanPath)
			content, err := os.ReadFile(absPath)
			if err == nil {
				parsed.FullContent = string(content)
				parsed.AnnotatedLines = buildAnnotatedLines(string(content), parsed.Hunks)
			}
		}
	}

	return parsed, nil
}

// buildAnnotatedLines creates annotated lines from file content and diff hunks
func buildAnnotatedLines(content string, hunks []DiffHunk) []AnnotatedLine {
	lines := strings.Split(content, "\n")
	annotated := make([]AnnotatedLine, 0, len(lines)*2) // Extra capacity for deleted lines

	// Build a map of line numbers to their change status
	// Also track deleted lines that should be inserted
	type lineInfo struct {
		change      LineChange
		deletedAt   int    // Position where deleted lines should be inserted
		deletedLine string // Content of deleted line
	}

	addedLines := make(map[int]bool)
	deletedLines := make([]struct {
		afterLine int
		content   string
		oldNumber int
	}, 0)

	for _, hunk := range hunks {
		newLineNum := hunk.NewStart
		oldLineNum := hunk.OldStart

		for _, line := range hunk.Lines {
			if len(line) == 0 {
				continue
			}

			prefix := line[0]
			lineContent := ""
			if len(line) > 1 {
				lineContent = line[1:]
			}

			switch prefix {
			case '+':
				if !strings.HasPrefix(line, "+++") {
					addedLines[newLineNum] = true
					newLineNum++
				}
			case '-':
				if !strings.HasPrefix(line, "---") {
					deletedLines = append(deletedLines, struct {
						afterLine int
						content   string
						oldNumber int
					}{
						afterLine: newLineNum - 1, // Insert after the previous line
						content:   lineContent,
						oldNumber: oldLineNum,
					})
					oldLineNum++
				}
			default:
				// Context line
				newLineNum++
				oldLineNum++
			}
		}
	}

	// Now build the annotated lines, inserting deleted lines where appropriate
	deletedIdx := 0
	for i, line := range lines {
		lineNum := i + 1

		// Insert any deleted lines that should appear before this line
		for deletedIdx < len(deletedLines) && deletedLines[deletedIdx].afterLine < lineNum {
			annotated = append(annotated, AnnotatedLine{
				Number:    0, // Deleted lines don't have a current line number
				Content:   deletedLines[deletedIdx].content,
				Change:    LineChangeDeleted,
				OldNumber: deletedLines[deletedIdx].oldNumber,
			})
			deletedIdx++
		}

		// Add the current line
		change := LineChangeNone
		if addedLines[lineNum] {
			change = LineChangeAdded
		}
		annotated = append(annotated, AnnotatedLine{
			Number:  lineNum,
			Content: line,
			Change:  change,
		})
	}

	// Add any remaining deleted lines at the end
	for deletedIdx < len(deletedLines) {
		annotated = append(annotated, AnnotatedLine{
			Number:    0,
			Content:   deletedLines[deletedIdx].content,
			Change:    LineChangeDeleted,
			OldNumber: deletedLines[deletedIdx].oldNumber,
		})
		deletedIdx++
	}

	return annotated
}

// ParseDiffOutput parses raw git diff output into structured form
func ParseDiffOutput(raw string) *DiffResponse {
	resp := &DiffResponse{
		Raw:   raw,
		Hunks: []DiffHunk{},
		Stats: DiffStats{},
	}

	if strings.TrimSpace(raw) == "" {
		resp.HasDiff = false
		return resp
	}

	resp.HasDiff = true
	lines := strings.Split(raw, "\n")
	filesSet := make(map[string]bool)

	var currentHunk *DiffHunk
	hunkRegex := regexp.MustCompile(`^@@\s*-(\d+)(?:,(\d+))?\s*\+(\d+)(?:,(\d+))?\s*@@(.*)$`)

	for _, line := range lines {
		// Track file changes for stats
		if strings.HasPrefix(line, "diff --git") {
			filesSet[line] = true
			continue
		}

		// Parse hunk headers
		if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
			if currentHunk != nil {
				resp.Hunks = append(resp.Hunks, *currentHunk)
			}
			currentHunk = &DiffHunk{
				OldStart: atoi(matches[1]),
				OldCount: atoiWithDefault(matches[2], 1),
				NewStart: atoi(matches[3]),
				NewCount: atoiWithDefault(matches[4], 1),
				Header:   strings.TrimSpace(matches[5]),
				Lines:    []string{},
			}
			continue
		}

		// Collect hunk lines and track stats
		if currentHunk != nil {
			currentHunk.Lines = append(currentHunk.Lines, line)
		}

		// Count additions and deletions
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			resp.Stats.Additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			resp.Stats.Deletions++
		}
	}

	// Add last hunk
	if currentHunk != nil {
		resp.Hunks = append(resp.Hunks, *currentHunk)
	}

	resp.Stats.Files = len(filesSet)

	return resp
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func atoiWithDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
