// Package diff provides diff generation and patch application for sandboxes.
//
// # External Command Dependencies
//
// This package relies on external commands for diff generation and patch application:
//
//   - diff: Used for generating unified diffs of modified files (GNU diffutils).
//     Called via exec.CommandContext("diff", "-u", ...).
//     Required for modified file comparisons.
//
//   - git: Used for patch application in git repositories.
//     Called via exec.CommandContext("git", "apply", ...).
//     Only used when target directory is a git repository.
//
//   - patch: Fallback for patch application in non-git directories.
//     Called via exec.CommandContext("patch", "-p1", ...).
//     Used when target is not a git repository.
//
// # Error Handling
//
// When external commands fail, errors include:
//   - The command that was run
//   - The exit code
//   - Stderr output for debugging
//
// # Binary File Handling
//
// Binary files are detected by checking for null bytes in the first 8KB.
// Binary files are reported in the diff output but their content is not
// included (only a "Binary file <path>" marker).
//
// # Assumptions
//
//   - These commands are available in the system PATH
//   - The 'diff' command follows GNU diff behavior (exit code 1 = files differ)
//   - File paths do not contain newlines or special characters that would
//     break the diff format
package diff

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

// GeneratorConfig holds configuration for diff generation.
type GeneratorConfig struct {
	// BinaryDetectionThreshold is the number of bytes to scan for binary detection.
	// Default: 8000
	BinaryDetectionThreshold int
}

// DefaultGeneratorConfig returns sensible defaults.
func DefaultGeneratorConfig() GeneratorConfig {
	return GeneratorConfig{
		BinaryDetectionThreshold: 8000,
	}
}

// Generator creates unified diffs from sandbox changes.
type Generator struct {
	config GeneratorConfig
	runner CommandRunner
}

// NewGenerator creates a new diff generator with default config.
// Uses DefaultCommandRunner() for external command execution.
func NewGenerator() *Generator {
	return &Generator{
		config: DefaultGeneratorConfig(),
		runner: DefaultCommandRunner(),
	}
}

// NewGeneratorWithConfig creates a new diff generator with custom config.
// Uses DefaultCommandRunner() for external command execution.
func NewGeneratorWithConfig(cfg GeneratorConfig) *Generator {
	if cfg.BinaryDetectionThreshold <= 0 {
		cfg.BinaryDetectionThreshold = 8000
	}
	return &Generator{
		config: cfg,
		runner: DefaultCommandRunner(),
	}
}

// NewGeneratorWithRunner creates a diff generator with a custom command runner.
// This is the primary seam for test isolation - inject a MockCommandRunner
// to test diff generation without executing real commands.
func NewGeneratorWithRunner(runner CommandRunner) *Generator {
	return &Generator{
		config: DefaultGeneratorConfig(),
		runner: runner,
	}
}

// NewGeneratorWithConfigAndRunner creates a diff generator with custom config and runner.
func NewGeneratorWithConfigAndRunner(cfg GeneratorConfig, runner CommandRunner) *Generator {
	if cfg.BinaryDetectionThreshold <= 0 {
		cfg.BinaryDetectionThreshold = 8000
	}
	return &Generator{
		config: cfg,
		runner: runner,
	}
}

// GenerateDiff creates a unified diff for all changes in a sandbox.
//
// # Preconditions
//
//   - s.UpperDir and s.LowerDir must be set (non-empty)
//   - UpperDir must exist (contains the changes)
//   - LowerDir must exist (contains the originals for comparison)
//
// # Assumptions Guarded
//
// This function validates its preconditions and returns clear errors if they
// are not met, rather than failing with confusing filesystem errors.
func (g *Generator) GenerateDiff(ctx context.Context, s *types.Sandbox, changes []*types.FileChange) (*types.DiffResult, error) {
	// ASSUMPTION: Paths are initialized
	// GUARD: Check for empty strings with clear message
	if s.UpperDir == "" {
		return nil, fmt.Errorf("sandbox upper directory path is empty (sandbox not properly initialized)")
	}
	if s.LowerDir == "" {
		return nil, fmt.Errorf("sandbox lower directory path is empty (sandbox not properly initialized)")
	}

	// ASSUMPTION: Directories actually exist
	// GUARD: Check filesystem to catch cleanup races or configuration errors
	if _, err := os.Stat(s.UpperDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("sandbox upper directory does not exist: %s (sandbox may have been cleaned up)", s.UpperDir)
	}
	if _, err := os.Stat(s.LowerDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("sandbox lower directory does not exist: %s (project root may have moved)", s.LowerDir)
	}

	// Sort changes for stable output
	sortedChanges := make([]*types.FileChange, len(changes))
	copy(sortedChanges, changes)
	sort.Slice(sortedChanges, func(i, j int) bool {
		return sortedChanges[i].FilePath < sortedChanges[j].FilePath
	})

	var diffBuilder strings.Builder
	var added, deleted, modified int

	for _, change := range sortedChanges {
		switch change.ChangeType {
		case types.ChangeTypeAdded:
			added++
			fileDiff, err := g.diffNewFile(ctx, s.UpperDir, change.FilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to diff new file %s: %w", change.FilePath, err)
			}
			diffBuilder.WriteString(fileDiff)

		case types.ChangeTypeDeleted:
			deleted++
			fileDiff, err := g.diffDeletedFile(ctx, s.LowerDir, change.FilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to diff deleted file %s: %w", change.FilePath, err)
			}
			diffBuilder.WriteString(fileDiff)

		case types.ChangeTypeModified:
			modified++
			fileDiff, err := g.diffModifiedFile(ctx, s.LowerDir, s.UpperDir, change.FilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to diff modified file %s: %w", change.FilePath, err)
			}
			diffBuilder.WriteString(fileDiff)
		}
	}

	return &types.DiffResult{
		SandboxID:     s.ID,
		Files:         sortedChanges,
		UnifiedDiff:   diffBuilder.String(),
		Generated:     time.Now(),
		TotalAdded:    added,
		TotalDeleted:  deleted,
		TotalModified: modified,
	}, nil
}

// diffNewFile generates a diff for a newly added file.
func (g *Generator) diffNewFile(ctx context.Context, upperDir, relPath string) (string, error) {
	filePath := filepath.Join(upperDir, relPath)

	// Check if it's a directory
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return fmt.Sprintf("diff --git a/%s b/%s\nnew file mode 040755\n--- /dev/null\n+++ b/%s\n",
			relPath, relPath, relPath), nil
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Check if binary
	if g.isBinary(content) {
		return fmt.Sprintf("diff --git a/%s b/%s\nnew file mode %06o\nBinary file %s\n",
			relPath, relPath, info.Mode().Perm(), relPath), nil
	}

	// Create unified diff header
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", relPath, relPath))
	builder.WriteString(fmt.Sprintf("new file mode %06o\n", info.Mode().Perm()))
	builder.WriteString("--- /dev/null\n")
	builder.WriteString(fmt.Sprintf("+++ b/%s\n", relPath))

	// Add lines
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 {
		builder.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
		for _, line := range lines {
			builder.WriteString("+" + line + "\n")
		}
	}

	return builder.String(), nil
}

// diffDeletedFile generates a diff for a deleted file.
func (g *Generator) diffDeletedFile(ctx context.Context, lowerDir, relPath string) (string, error) {
	filePath := filepath.Join(lowerDir, relPath)

	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return fmt.Sprintf("diff --git a/%s b/%s\ndeleted file mode 040755\n",
			relPath, relPath), nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	if g.isBinary(content) {
		return fmt.Sprintf("diff --git a/%s b/%s\ndeleted file mode %06o\nBinary file %s\n",
			relPath, relPath, info.Mode().Perm(), relPath), nil
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", relPath, relPath))
	builder.WriteString(fmt.Sprintf("deleted file mode %06o\n", info.Mode().Perm()))
	builder.WriteString(fmt.Sprintf("--- a/%s\n", relPath))
	builder.WriteString("+++ /dev/null\n")

	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 {
		builder.WriteString(fmt.Sprintf("@@ -1,%d +0,0 @@\n", len(lines)))
		for _, line := range lines {
			builder.WriteString("-" + line + "\n")
		}
	}

	return builder.String(), nil
}

// diffModifiedFile generates a diff for a modified file.
// Uses the CommandRunner seam for external command execution, enabling test isolation.
func (g *Generator) diffModifiedFile(ctx context.Context, lowerDir, upperDir, relPath string) (string, error) {
	oldPath := filepath.Join(lowerDir, relPath)
	newPath := filepath.Join(upperDir, relPath)

	// Use the command runner for external diff command
	result := g.runner.Run(ctx, "", "", "diff", "-u", "--label", "a/"+relPath, "--label", "b/"+relPath, oldPath, newPath)

	// diff returns exit code 1 when files differ, which is expected
	if result.Err != nil {
		if result.ExitCode == 1 {
			// Files differ, output is in stdout
			return "diff --git a/" + relPath + " b/" + relPath + "\n" + result.Stdout, nil
		}
		// Actual error or files are the same (exit code 0)
		if result.ExitCode == 0 {
			return "", nil // No difference
		}
		return "", fmt.Errorf("diff command failed: %v: %s", result.Err, result.Stderr)
	}

	// No difference (exit code 0, no error)
	return "", nil
}

// isBinary checks if content appears to be binary.
func (g *Generator) isBinary(content []byte) bool {
	// Check for null bytes in first N bytes (configurable threshold)
	threshold := g.config.BinaryDetectionThreshold
	checkLen := len(content)
	if checkLen > threshold {
		checkLen = threshold
	}
	return bytes.Contains(content[:checkLen], []byte{0})
}

// isBinaryDefault is a package-level helper for code that doesn't have a Generator.
func isBinaryDefault(content []byte) bool {
	// Check for null bytes in first 8000 bytes (default threshold)
	checkLen := len(content)
	if checkLen > 8000 {
		checkLen = 8000
	}
	return bytes.Contains(content[:checkLen], []byte{0})
}

// Patcher applies diffs to the canonical repo.
type Patcher struct {
	runner CommandRunner
}

// NewPatcher creates a new patcher with the default command runner.
func NewPatcher() *Patcher {
	return &Patcher{runner: DefaultCommandRunner()}
}

// NewPatcherWithRunner creates a patcher with a custom command runner.
// This is the primary seam for test isolation - inject a MockCommandRunner
// to test patch application without executing real git/patch commands.
func NewPatcherWithRunner(runner CommandRunner) *Patcher {
	return &Patcher{runner: runner}
}

// ApplyOptions controls patch application behavior.
type ApplyOptions struct {
	DryRun     bool
	CommitMsg  string
	Author     string
	AllowEmpty bool

	// CreateCommit controls whether to create a git commit after applying changes.
	// If false (default), changes are applied to the working tree only.
	// If true and CommitMsg is set, a commit is created.
	CreateCommit bool

	// FilePaths contains the list of file paths to stage when creating a commit.
	// This prevents the `git add -A` behavior that stages all repository changes.
	// Only used when CreateCommit is true.
	FilePaths []string
}

// ApplyResult contains the outcome of a patch application.
type ApplyResult struct {
	Success      bool
	FilesApplied int
	CommitHash   string
	Errors       []string
}

// ApplyDiff applies a unified diff to the target directory.
// Uses the CommandRunner seam for external command execution, enabling test isolation.
func (p *Patcher) ApplyDiff(ctx context.Context, targetDir, diff string, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}

	if diff == "" {
		result.Success = true
		return result, nil
	}

	// Use git apply if in a git repo
	if p.isGitRepo(ctx, targetDir) {
		return p.applyWithGit(ctx, targetDir, diff, opts)
	}

	// Fall back to patch command
	return p.applyWithPatch(ctx, targetDir, diff, opts)
}

// applyWithGit uses git apply for patch application.
// Uses the CommandRunner seam for external command execution.
func (p *Patcher) applyWithGit(ctx context.Context, targetDir, diff string, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}

	// First check with --stat --summary (and --check for dry run)
	args := []string{"git", "apply", "--stat", "--summary"}
	if opts.DryRun {
		args = append(args, "--check")
	}

	checkResult := p.runner.Run(ctx, targetDir, diff, args...)
	if checkResult.Err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("git apply failed: %v: %s", checkResult.Err, checkResult.Stderr))
		return result, nil
	}

	if opts.DryRun {
		result.Success = true
		return result, nil
	}

	// Actually apply the patch
	applyResult := p.runner.Run(ctx, targetDir, diff, "git", "apply")
	if applyResult.Err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("git apply failed: %v: %s", applyResult.Err, applyResult.Stderr))
		return result, nil
	}

	result.Success = true

	// Create commit only if explicitly requested AND message provided
	if opts.CreateCommit && opts.CommitMsg != "" {
		commitHash, err := p.createCommit(ctx, targetDir, opts)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("commit failed: %v", err))
		} else {
			result.CommitHash = commitHash
		}
	}

	return result, nil
}

// applyWithPatch uses the patch command for non-git directories.
// Uses the CommandRunner seam for external command execution.
func (p *Patcher) applyWithPatch(ctx context.Context, targetDir, diff string, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}

	args := []string{"patch", "-p1", "-d", targetDir}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	patchResult := p.runner.Run(ctx, "", diff, args...)
	if patchResult.Err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("patch failed: %v: %s", patchResult.Err, patchResult.Stderr))
		return result, nil
	}

	result.Success = true
	return result, nil
}

// createCommit creates a git commit with the applied changes.
// It stages only the specific files listed in opts.FilePaths to avoid
// accidentally committing unrelated changes in the repository.
// Uses the CommandRunner seam for external command execution.
func (p *Patcher) createCommit(ctx context.Context, targetDir string, opts ApplyOptions) (string, error) {
	// Stage only the specific files that were modified
	if len(opts.FilePaths) == 0 {
		return "", fmt.Errorf("no files specified for commit (FilePaths is empty)")
	}

	for _, filePath := range opts.FilePaths {
		absPath := filepath.Join(targetDir, filePath)

		// Check if file exists to determine if we're adding or removing
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			// File was deleted - use git rm to stage the deletion
			// Ignore errors for files that weren't tracked
			p.runner.Run(ctx, targetDir, "", "git", "rm", "--cached", "--ignore-unmatch", filePath)
		} else {
			// File exists - stage it
			result := p.runner.Run(ctx, targetDir, "", "git", "add", filePath)
			if result.Err != nil {
				return "", fmt.Errorf("git add %s failed: %w", filePath, result.Err)
			}
		}
	}

	// Create commit
	args := []string{"git", "commit", "-m", opts.CommitMsg}
	if opts.Author != "" {
		args = append(args, "--author", opts.Author)
	}
	if opts.AllowEmpty {
		args = append(args, "--allow-empty")
	}

	commitResult := p.runner.Run(ctx, targetDir, "", args...)
	if commitResult.Err != nil {
		return "", fmt.Errorf("git commit failed: %w: %s", commitResult.Err, commitResult.Stderr)
	}

	// Get commit hash
	hashResult := p.runner.Run(ctx, targetDir, "", "git", "rev-parse", "HEAD")
	if hashResult.Err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", hashResult.Err)
	}

	return strings.TrimSpace(hashResult.Stdout), nil
}

// CreateCommitFromFiles creates a git commit for a specific set of files.
// This is used for batch committing pending changes from multiple sandboxes.
func (p *Patcher) CreateCommitFromFiles(ctx context.Context, targetDir string, opts ApplyOptions) (string, error) {
	return p.createCommit(ctx, targetDir, opts)
}

// isGitRepo checks if a directory is a git repository.
// This is a method to use the CommandRunner seam for test isolation.
func (p *Patcher) isGitRepo(ctx context.Context, dir string) bool {
	result := p.runner.Run(ctx, "", "", "git", "-C", dir, "rev-parse", "--git-dir")
	return result.Err == nil
}

// isGitRepo is a package-level helper that uses the default command runner.
// For testable code, use Patcher.isGitRepo instead.
func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// CopyChanges copies changes from the sandbox upper layer to the target directory.
func CopyChanges(ctx context.Context, s *types.Sandbox, changes []*types.FileChange, targetDir string) error {
	for _, change := range changes {
		srcPath := filepath.Join(s.UpperDir, change.FilePath)
		dstPath := filepath.Join(targetDir, change.FilePath)

		switch change.ChangeType {
		case types.ChangeTypeAdded, types.ChangeTypeModified:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", change.FilePath, err)
			}

			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy %s: %w", change.FilePath, err)
			}

		case types.ChangeTypeDeleted:
			// Remove file
			if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete %s: %w", change.FilePath, err)
			}
		}
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// GenerateFileDiff creates a diff for a single file given its ID.
func GenerateFileDiff(ctx context.Context, s *types.Sandbox, change *types.FileChange) (string, error) {
	gen := NewGenerator()

	switch change.ChangeType {
	case types.ChangeTypeAdded:
		return gen.diffNewFile(ctx, s.UpperDir, change.FilePath)
	case types.ChangeTypeDeleted:
		return gen.diffDeletedFile(ctx, s.LowerDir, change.FilePath)
	case types.ChangeTypeModified:
		return gen.diffModifiedFile(ctx, s.LowerDir, s.UpperDir, change.FilePath)
	default:
		return "", fmt.Errorf("unknown change type: %s", change.ChangeType)
	}
}

// FilterChanges returns only the changes with matching IDs.
func FilterChanges(changes []*types.FileChange, ids []uuid.UUID) []*types.FileChange {
	idSet := make(map[uuid.UUID]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	var filtered []*types.FileChange
	for _, c := range changes {
		if idSet[c.ID] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// ParsedHunk represents a parsed diff hunk with metadata.
type ParsedHunk struct {
	Header    string   // Full header line including context
	OldStart  int      // Starting line number in original file
	OldCount  int      // Number of lines from original file
	NewStart  int      // Starting line number in new file
	NewCount  int      // Number of lines in new file
	Lines     []string // The actual diff lines (including +/-/space prefixes)
	FileID    uuid.UUID
	HunkIndex int
}

// ParsedFileDiff represents a parsed file diff with its hunks.
type ParsedFileDiff struct {
	Path       string
	ChangeType types.ChangeType
	Header     string // Full git diff header
	Hunks      []*ParsedHunk
}

// ParseUnifiedDiff parses a unified diff string into structured format.
// This enables hunk-level selection and filtering.
func ParseUnifiedDiff(diff string) []*ParsedFileDiff {
	var files []*ParsedFileDiff
	lines := strings.Split(diff, "\n")
	var currentFile *ParsedFileDiff
	var currentHunk *ParsedHunk
	hunkIdx := 0

	for _, line := range lines {
		// File header: diff --git a/path b/path
		if strings.HasPrefix(line, "diff --git") {
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, currentHunk)
				}
				files = append(files, currentFile)
			}
			currentFile = &ParsedFileDiff{
				Header:     line,
				ChangeType: types.ChangeTypeModified,
				Hunks:      []*ParsedHunk{},
			}
			currentHunk = nil
			hunkIdx = 0

			// Extract path from header
			if idx := strings.Index(line, " b/"); idx > 0 {
				currentFile.Path = line[idx+3:]
			}
			continue
		}

		if currentFile == nil {
			continue
		}

		// Append header lines
		if strings.HasPrefix(line, "new file mode") {
			currentFile.ChangeType = types.ChangeTypeAdded
			currentFile.Header += "\n" + line
			continue
		}
		if strings.HasPrefix(line, "deleted file mode") {
			currentFile.ChangeType = types.ChangeTypeDeleted
			currentFile.Header += "\n" + line
			continue
		}
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "index ") {
			currentFile.Header += "\n" + line
			continue
		}

		// Hunk header: @@ -old,count +new,count @@
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, currentHunk)
			}
			currentHunk = parseHunkHeader(line, hunkIdx)
			hunkIdx++
			continue
		}

		// Content lines
		if currentHunk != nil {
			currentHunk.Lines = append(currentHunk.Lines, line)
		}
	}

	// Push final file and hunk
	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, currentHunk)
		}
		files = append(files, currentFile)
	}

	return files
}

// parseHunkHeader parses a hunk header like "@@ -1,5 +1,7 @@ context"
func parseHunkHeader(line string, idx int) *ParsedHunk {
	hunk := &ParsedHunk{
		Header:    line,
		HunkIndex: idx,
		OldStart:  1,
		OldCount:  1,
		NewStart:  1,
		NewCount:  1,
	}

	// Parse @@ -old,count +new,count @@
	// Format: @@ -startLine,count +startLine,count @@ optional context
	parts := strings.SplitN(line, "@@", 3)
	if len(parts) < 2 {
		return hunk
	}

	rangeStr := strings.TrimSpace(parts[1])
	// rangeStr looks like: "-1,5 +1,7"
	fields := strings.Fields(rangeStr)
	for _, f := range fields {
		if strings.HasPrefix(f, "-") {
			parseRange(f[1:], &hunk.OldStart, &hunk.OldCount)
		} else if strings.HasPrefix(f, "+") {
			parseRange(f[1:], &hunk.NewStart, &hunk.NewCount)
		}
	}

	return hunk
}

// parseRange parses "start,count" or "start" into line numbers
func parseRange(s string, start, count *int) {
	parts := strings.Split(s, ",")
	if len(parts) >= 1 {
		fmt.Sscanf(parts[0], "%d", start)
	}
	if len(parts) >= 2 {
		fmt.Sscanf(parts[1], "%d", count)
	}
}

// FilterHunks filters a unified diff to only include selected hunks.
// hunkRanges specifies which hunks to include for each file.
func FilterHunks(diff string, hunkRanges []types.HunkRange, fileChanges []*types.FileChange) string {
	if len(hunkRanges) == 0 {
		return diff
	}

	// Build a map of file ID -> file path for lookup
	filePathMap := make(map[uuid.UUID]string)
	for _, fc := range fileChanges {
		filePathMap[fc.ID] = fc.FilePath
	}

	// Build a map of file path -> set of hunk line ranges to include
	type hunkKey struct {
		filePath  string
		startLine int
		endLine   int
	}
	selectedHunks := make(map[string][]hunkKey)
	for _, hr := range hunkRanges {
		filePath, ok := filePathMap[hr.FileID]
		if !ok {
			continue
		}
		selectedHunks[filePath] = append(selectedHunks[filePath], hunkKey{
			filePath:  filePath,
			startLine: hr.StartLine,
			endLine:   hr.EndLine,
		})
	}

	// Parse the diff and filter
	parsedFiles := ParseUnifiedDiff(diff)
	var result strings.Builder

	for _, file := range parsedFiles {
		selectedForFile, hasFile := selectedHunks[file.Path]
		if !hasFile {
			continue
		}

		// Check which hunks in this file are selected
		var selectedFileHunks []*ParsedHunk
		for _, hunk := range file.Hunks {
			for _, sel := range selectedForFile {
				// Match by hunk line range (newStart is in the range)
				if hunk.NewStart >= sel.startLine && hunk.NewStart <= sel.endLine {
					selectedFileHunks = append(selectedFileHunks, hunk)
					break
				}
			}
		}

		if len(selectedFileHunks) == 0 {
			continue
		}

		// Reconstruct the diff for this file with only selected hunks
		result.WriteString(file.Header)
		result.WriteString("\n")

		for _, hunk := range selectedFileHunks {
			result.WriteString(hunk.Header)
			result.WriteString("\n")
			for _, line := range hunk.Lines {
				result.WriteString(line)
				result.WriteString("\n")
			}
		}
	}

	return result.String()
}

// GetHunksForFile extracts all hunks from a diff for a specific file path.
func GetHunksForFile(diff string, filePath string) []*ParsedHunk {
	parsedFiles := ParseUnifiedDiff(diff)
	for _, file := range parsedFiles {
		if file.Path == filePath {
			return file.Hunks
		}
	}
	return nil
}

// --- Conflict Detection (OT-P2-002) ---

// GetGitCommitHash returns the current HEAD commit hash for a git repository.
// Returns empty string if the directory is not a git repo or git is unavailable.
func GetGitCommitHash(ctx context.Context, repoDir string) (string, error) {
	if !isGitRepo(repoDir) {
		return "", nil // Not a git repo, no commit hash available
	}

	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "rev-parse", "HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get git commit hash: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// CheckRepoChanged compares the current repo commit hash against a base hash.
// Returns true if the repo has changed since the base hash was recorded.
// Returns false if either hash is empty (non-git repo) or they match.
func CheckRepoChanged(ctx context.Context, repoDir, baseHash string) (bool, string, error) {
	if baseHash == "" {
		return false, "", nil // No base hash to compare against
	}

	currentHash, err := GetGitCommitHash(ctx, repoDir)
	if err != nil {
		return false, "", err
	}

	if currentHash == "" {
		return false, "", nil // Not a git repo anymore
	}

	return currentHash != baseHash, currentHash, nil
}

// GetChangedFilesSinceCommit returns files changed in the repo since a specific commit.
// This helps identify which files in a sandbox might have conflicts.
func GetChangedFilesSinceCommit(ctx context.Context, repoDir, baseCommit string) ([]string, error) {
	if !isGitRepo(repoDir) || baseCommit == "" {
		return nil, nil
	}

	// Get list of files changed since base commit
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "diff", "--name-only", baseCommit+"..HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return nil, nil
	}

	return strings.Split(output, "\n"), nil
}

// FindConflictingFiles identifies files that have been modified both in the sandbox
// and in the canonical repo since sandbox creation.
func FindConflictingFiles(sandboxChanges []*types.FileChange, repoChangedFiles []string) []string {
	sandboxFilePaths := make(map[string]bool)
	for _, change := range sandboxChanges {
		sandboxFilePaths[change.FilePath] = true
	}

	var conflicts []string
	for _, repoFile := range repoChangedFiles {
		if sandboxFilePaths[repoFile] {
			conflicts = append(conflicts, repoFile)
		}
	}

	return conflicts
}

// ConflictCheckResult contains the result of a conflict detection check.
type ConflictCheckResult struct {
	HasChanged       bool     // True if repo has changed since sandbox creation
	BaseCommitHash   string   // Original commit hash at sandbox creation
	CurrentHash      string   // Current commit hash in canonical repo
	RepoChangedFiles []string // Files changed in repo since sandbox creation
	ConflictingFiles []string // Files changed in both sandbox and repo
}

// CheckForConflicts performs a comprehensive conflict detection check.
// This should be called before approving changes to detect potential issues.
func CheckForConflicts(ctx context.Context, s *types.Sandbox, sandboxChanges []*types.FileChange) (*ConflictCheckResult, error) {
	result := &ConflictCheckResult{
		BaseCommitHash: s.BaseCommitHash,
	}

	// If no base commit hash, we can't detect conflicts
	if s.BaseCommitHash == "" {
		return result, nil
	}

	// Check if repo has changed
	changed, currentHash, err := CheckRepoChanged(ctx, s.ProjectRoot, s.BaseCommitHash)
	if err != nil {
		return nil, err
	}

	result.HasChanged = changed
	result.CurrentHash = currentHash

	if !changed {
		return result, nil
	}

	// Get list of files changed in repo
	repoChangedFiles, err := GetChangedFilesSinceCommit(ctx, s.ProjectRoot, s.BaseCommitHash)
	if err != nil {
		return nil, err
	}
	result.RepoChangedFiles = repoChangedFiles

	// Find conflicting files
	result.ConflictingFiles = FindConflictingFiles(sandboxChanges, repoChangedFiles)

	return result, nil
}

// --- Git Status Reconciliation ---

// GitFileStatus represents the status of a file in git's working tree.
type GitFileStatus struct {
	Path       string // Relative path from repo root
	IndexState string // State in index (staged): M, A, D, R, C, U, or ?
	WorkTree   string // State in working tree: M, D, U, or ?
	IsStaged   bool   // True if file has staged changes
	IsDirty    bool   // True if file has unstaged changes
}

// GetUncommittedFiles returns all uncommitted files from git status.
// This includes both staged and unstaged changes.
func GetUncommittedFiles(ctx context.Context, repoDir string) ([]GitFileStatus, error) {
	if !isGitRepo(repoDir) {
		return nil, nil
	}

	// Use porcelain format for machine-readable output
	// Format: XY PATH (or XY PATH -> PATH for renames)
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "status", "--porcelain", "-z")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	output := stdout.String()
	if output == "" {
		return nil, nil
	}

	var files []GitFileStatus

	// Split by null character (from -z flag)
	entries := strings.Split(output, "\x00")
	for _, entry := range entries {
		if len(entry) < 3 {
			continue
		}

		// First two chars are the status codes
		indexState := string(entry[0])
		workTree := string(entry[1])
		path := strings.TrimSpace(entry[3:])

		// Handle rename entries (have " -> " in path)
		if idx := strings.Index(path, " -> "); idx > 0 {
			path = path[idx+4:] // Use the new path
		}

		if path == "" {
			continue
		}

		status := GitFileStatus{
			Path:       path,
			IndexState: indexState,
			WorkTree:   workTree,
			IsStaged:   indexState != " " && indexState != "?",
			IsDirty:    workTree != " " && workTree != "?",
		}
		files = append(files, status)
	}

	return files, nil
}

// GetUncommittedFilePaths returns just the paths of uncommitted files.
// This is a convenience wrapper around GetUncommittedFiles.
func GetUncommittedFilePaths(ctx context.Context, repoDir string) ([]string, error) {
	files, err := GetUncommittedFiles(ctx, repoDir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	return paths, nil
}

// ReconcileResult contains the result of reconciling pending changes with git status.
type ReconcileResult struct {
	// StillPending are files in DB that are still uncommitted in git
	StillPending []string

	// AlreadyCommitted are files in DB marked pending but already committed externally
	AlreadyCommitted []string

	// NotFound are files in DB that don't exist in git status at all
	// (either committed or never existed)
	NotFound []string
}

// ReconcilePendingWithGit compares database pending files with actual git status.
// This detects files that were committed outside of workspace-sandbox.
func ReconcilePendingWithGit(ctx context.Context, repoDir string, pendingPaths []string) (*ReconcileResult, error) {
	uncommitted, err := GetUncommittedFilePaths(ctx, repoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get uncommitted files: %w", err)
	}

	// Build set of uncommitted paths
	uncommittedSet := make(map[string]bool)
	for _, p := range uncommitted {
		uncommittedSet[p] = true
	}

	result := &ReconcileResult{}
	for _, pending := range pendingPaths {
		if uncommittedSet[pending] {
			result.StillPending = append(result.StillPending, pending)
		} else {
			// File is not in uncommitted list - either already committed or deleted
			result.AlreadyCommitted = append(result.AlreadyCommitted, pending)
		}
	}

	return result, nil
}
