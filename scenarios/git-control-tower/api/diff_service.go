package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxDiffFileBytes  int64 = 4 * 1024 * 1024
	binarySampleBytes       = 8 * 1024
)

type FileTooLargeError struct {
	Path  string
	Size  int64
	Limit int64
}

func (e *FileTooLargeError) Error() string {
	path := e.Path
	if path == "" {
		path = "file"
	}
	return fmt.Sprintf("file too large to display: %s (%s > %s)", path, formatBytes(e.Size), formatBytes(e.Limit))
}

type UnsupportedBinaryError struct {
	Path string
}

func (e *UnsupportedBinaryError) Error() string {
	path := e.Path
	if path == "" {
		path = "file"
	}
	return fmt.Sprintf("binary file not supported for display: %s", path)
}

type binaryKind int

const (
	binaryNone binaryKind = iota
	binaryImage
	binaryUnsupported
)

var binaryImageExtensions = map[string]struct{}{
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".gif":  {},
	".svg":  {},
	".webp": {},
	".ico":  {},
	".bmp":  {},
	".tiff": {},
}

var binaryUnsupportedExtensions = map[string]struct{}{
	".pdf": {},
}

func binaryKindForPath(path string) binaryKind {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := binaryImageExtensions[ext]; ok {
		return binaryImage
	}
	if _, ok := binaryUnsupportedExtensions[ext]; ok {
		return binaryUnsupported
	}
	return binaryNone
}

func detectBinaryKind(path string, data []byte) binaryKind {
	kind := binaryKindForPath(path)
	if kind != binaryNone {
		return kind
	}
	sample := data
	if len(sample) > binarySampleBytes {
		sample = sample[:binarySampleBytes]
	}
	if isBinaryData(sample) {
		return binaryUnsupported
	}
	return binaryNone
}

func isBinaryData(sample []byte) bool {
	if len(sample) == 0 {
		return false
	}
	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}
	if !utf8.Valid(sample) {
		return true
	}
	return false
}

func readFileForDisplay(absPath, displayPath string) ([]byte, int64, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, 0, err
	}
	if info.IsDir() {
		return nil, 0, fmt.Errorf("path is a directory")
	}
	size := info.Size()
	if size > maxDiffFileBytes {
		return nil, size, &FileTooLargeError{Path: displayPath, Size: size, Limit: maxDiffFileBytes}
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, size, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxDiffFileBytes+1))
	if err != nil {
		return nil, size, err
	}
	if int64(len(data)) > maxDiffFileBytes {
		return nil, size, &FileTooLargeError{Path: displayPath, Size: int64(len(data)), Limit: maxDiffFileBytes}
	}
	return data, size, nil
}

func ensureFileWithinLimit(repoDir, cleanPath string) error {
	absPath := filepath.Join(repoDir, cleanPath)
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory")
	}
	if info.Size() > maxDiffFileBytes {
		return &FileTooLargeError{Path: cleanPath, Size: info.Size(), Limit: maxDiffFileBytes}
	}
	return nil
}

func formatBytes(size int64) string {
	const kb = 1024
	const mb = 1024 * 1024
	if size >= mb {
		return fmt.Sprintf("%.1f MB", float64(size)/float64(mb))
	}
	if size >= kb {
		return fmt.Sprintf("%.1f KB", float64(size)/float64(kb))
	}
	return fmt.Sprintf("%d B", size)
}

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

// getSourceContent returns just the file content without any diff information
func getSourceContent(ctx context.Context, deps DiffDeps, req DiffRequest, repoDir string) (*DiffResponse, error) {
	cleanPath := cleanFilePath(req.Path)
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	var content string
	var annotatedLines []AnnotatedLine
	lineCount := 0
	var binaryKind binaryKind

	if req.Commit != "" {
		// Get file content at a specific commit
		out, err := deps.Git.ShowFileAtCommit(ctx, repoDir, req.Commit, cleanPath)
		if err != nil {
			return nil, fmt.Errorf("show file at commit: %w", err)
		}
		if int64(len(out)) > maxDiffFileBytes {
			return nil, &FileTooLargeError{Path: cleanPath, Size: int64(len(out)), Limit: maxDiffFileBytes}
		}
		binaryKind = detectBinaryKind(cleanPath, out)
		if binaryKind == binaryUnsupported {
			return nil, &UnsupportedBinaryError{Path: cleanPath}
		}
		if binaryKind == binaryImage {
			// Return base64 encoded for binary files
			content = base64.StdEncoding.EncodeToString(out)
		} else {
			content = string(out)
		}
	} else {
		// Get current file content from working directory
		absPath := filepath.Join(repoDir, cleanPath)
		data, _, err := readFileForDisplay(absPath, cleanPath)
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
		binaryKind = detectBinaryKind(cleanPath, data)
		if binaryKind == binaryUnsupported {
			return nil, &UnsupportedBinaryError{Path: cleanPath}
		}
		if binaryKind == binaryImage {
			// Return base64 encoded for binary files
			content = base64.StdEncoding.EncodeToString(data)
		} else {
			content = string(data)
		}
	}

	// Build annotated lines (all lines, no change markers) - only for text files
	if binaryKind == binaryNone {
		lines := strings.Split(content, "\n")
		annotatedLines = make([]AnnotatedLine, len(lines))
		for i, line := range lines {
			annotatedLines[i] = AnnotatedLine{
				Number:  i + 1,
				Content: line,
				Change:  LineChangeNone,
			}
		}
		lineCount = len(lines)
		if lineCount > 0 && lines[lineCount-1] == "" {
			lineCount-- // Don't count empty trailing line
		}
	}

	hasDiff := false
	stats := DiffStats{}

	if req.Untracked {
		hasDiff = true
		stats.Files = 1
		if binaryKind == binaryNone {
			stats.Additions = lineCount
		}
	} else {
		var diffOut []byte
		var diffErr error
		if req.Commit != "" {
			diffOut, diffErr = deps.Git.ShowCommitDiff(ctx, repoDir, req.Commit, cleanPath)
		} else {
			diffOut, diffErr = deps.Git.Diff(ctx, repoDir, cleanPath, req.Staged)
		}
		if diffErr != nil {
			return nil, fmt.Errorf("check diff for source content: %w", diffErr)
		}
		parsed := ParseDiffOutput(string(diffOut))
		enrichCommentStats(parsed, cleanPath)
		hasDiff = parsed.HasDiff
		stats = parsed.Stats
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
			if binaryKind != binaryNone {
				return ""
			}
			return hashContentBytes([]byte(content))
		}(),
	}, nil
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

	// For full_diff mode, get the file content and annotate lines
	if mode == ViewModeFullDiff && req.Path != "" {
		content, err := deps.Git.ShowFileAtCommit(ctx, repoDir, req.Commit, cleanPath)
		if err != nil {
			return nil, err
		}
		if int64(len(content)) > maxDiffFileBytes {
			return nil, &FileTooLargeError{Path: cleanPath, Size: int64(len(content)), Limit: maxDiffFileBytes}
		}
		binaryKind := detectBinaryKind(cleanPath, content)
		if binaryKind == binaryUnsupported {
			return nil, &UnsupportedBinaryError{Path: cleanPath}
		}
		if binaryKind == binaryImage {
			parsed.FullContent = base64.StdEncoding.EncodeToString(content)
		} else {
			parsed.FullContent = string(content)
			parsed.ContentHash = hashContentBytes(content)
			parsed.AnnotatedLines = buildAnnotatedLines(parsed.FullContent, parsed.Hunks)
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
	content, _, err := readFileForDisplay(absPath, cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	binaryKind := detectBinaryKind(cleanPath, content)
	if binaryKind == binaryUnsupported {
		return nil, &UnsupportedBinaryError{Path: cleanPath}
	}

	fileText := ""
	lines := []string{}
	lineCount := 0
	if binaryKind == binaryImage {
		fileText = base64.StdEncoding.EncodeToString(content)
	} else {
		fileText = string(content)
		lines = strings.Split(fileText, "\n")
		lineCount = len(lines)
		if lineCount > 0 && lines[lineCount-1] == "" {
			lineCount-- // Don't count empty trailing line
		}
	}

	// For untracked files, all lines are "added"
	var annotatedLines []AnnotatedLine
	if binaryKind == binaryNone {
		annotatedLines = make([]AnnotatedLine, len(lines))
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
			if binaryKind != binaryNone {
				return ""
			}
			return hashContentBytes(content)
		}(),
	}, nil
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

	// For full_diff mode, get the current file content and annotate lines
	if mode == ViewModeFullDiff && req.Path != "" {
		absPath := filepath.Join(repoDir, pathForGit)
		content, _, err := readFileForDisplay(absPath, pathForGit)
		if err != nil {
			return nil, err
		}
		binaryKind := detectBinaryKind(pathForGit, content)
		if binaryKind == binaryUnsupported {
			return nil, &UnsupportedBinaryError{Path: pathForGit}
		}
		if binaryKind == binaryImage {
			parsed.FullContent = base64.StdEncoding.EncodeToString(content)
		} else {
			parsed.FullContent = string(content)
			parsed.ContentHash = hashContentBytes(content)
			parsed.AnnotatedLines = buildAnnotatedLines(parsed.FullContent, parsed.Hunks)
		}
	}

	return parsed, nil
}

// buildAnnotatedLines creates annotated lines from file content and diff hunks
func buildAnnotatedLines(content string, hunks []DiffHunk) []AnnotatedLine {
	lines := strings.Split(content, "\n")
	annotated := make([]AnnotatedLine, 0, len(lines)*2) // Extra capacity for deleted lines

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

		// Detect renames in diff header metadata
		if strings.HasPrefix(line, "rename from ") {
			resp.Stats.IsRename = true
			resp.Stats.OldPath = strings.TrimPrefix(line, "rename from ")
			continue
		}
		if strings.HasPrefix(line, "rename to ") {
			continue
		}
		if strings.HasPrefix(line, "new file mode") {
			resp.Stats.IsNewFile = true
			continue
		}
		if strings.HasPrefix(line, "deleted file mode") {
			resp.Stats.IsDeletedFile = true
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

	// Compute enhanced metrics from parsed hunks
	resp.Stats.NetLines = resp.Stats.Additions - resp.Stats.Deletions
	resp.Stats.HunkCount = len(resp.Hunks)
	for _, h := range resp.Hunks {
		if n := countHunkChangedLines(h); n > resp.Stats.LargestHunk {
			resp.Stats.LargestHunk = n
		}
	}
	if total := resp.Stats.Additions + resp.Stats.Deletions; total > 0 && resp.Stats.HunkCount > 0 {
		resp.Stats.Density = float64(resp.Stats.HunkCount) / float64(total)
	}

	return resp
}

// countHunkChangedLines counts added/deleted lines in a single hunk.
func countHunkChangedLines(h DiffHunk) int {
	count := 0
	for _, line := range h.Lines {
		if len(line) > 0 && (line[0] == '+' || line[0] == '-') {
			if !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
				count++
			}
		}
	}
	return count
}

// isCommentLine checks if a trimmed line (without +/- prefix) is a comment
// based on the file extension.
func isCommentLine(trimmedContent string, ext string) bool {
	if trimmedContent == "" {
		return false
	}
	ext = strings.ToLower(ext)
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".java", ".c", ".cpp", ".rs", ".swift":
		return strings.HasPrefix(trimmedContent, "//") ||
			strings.HasPrefix(trimmedContent, "/*") ||
			strings.HasPrefix(trimmedContent, "*/") ||
			strings.HasPrefix(trimmedContent, "*")
	case ".py", ".rb", ".sh", ".bash", ".yaml", ".yml", ".toml":
		return strings.HasPrefix(trimmedContent, "#")
	case ".html", ".xml", ".svg":
		return strings.HasPrefix(trimmedContent, "<!--")
	}
	return false
}

// enrichCommentStats counts comment additions/deletions in the parsed diff.
func enrichCommentStats(resp *DiffResponse, path string) {
	if path == "" || resp == nil {
		return
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return
	}
	for _, hunk := range resp.Hunks {
		for _, line := range hunk.Lines {
			if len(line) == 0 {
				continue
			}
			prefix := line[0]
			if prefix == '+' && !strings.HasPrefix(line, "+++") {
				content := strings.TrimSpace(line[1:])
				if isCommentLine(content, ext) {
					resp.Stats.CommentAdditions++
				}
			} else if prefix == '-' && !strings.HasPrefix(line, "---") {
				content := strings.TrimSpace(line[1:])
				if isCommentLine(content, ext) {
					resp.Stats.CommentDeletions++
				}
			}
		}
	}
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
