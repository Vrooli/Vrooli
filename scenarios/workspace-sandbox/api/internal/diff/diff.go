// Package diff provides diff generation and patch application for sandboxes.
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

// Generator creates unified diffs from sandbox changes.
type Generator struct{}

// NewGenerator creates a new diff generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateDiff creates a unified diff for all changes in a sandbox.
func (g *Generator) GenerateDiff(ctx context.Context, s *types.Sandbox, changes []*types.FileChange) (*types.DiffResult, error) {
	if s.UpperDir == "" || s.LowerDir == "" {
		return nil, fmt.Errorf("sandbox paths not initialized")
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
	if isBinary(content) {
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

	if isBinary(content) {
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
func (g *Generator) diffModifiedFile(ctx context.Context, lowerDir, upperDir, relPath string) (string, error) {
	oldPath := filepath.Join(lowerDir, relPath)
	newPath := filepath.Join(upperDir, relPath)

	// Try using the system diff command for better output
	cmd := exec.CommandContext(ctx, "diff", "-u", "--label", "a/"+relPath, "--label", "b/"+relPath, oldPath, newPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	// diff returns exit code 1 when files differ, which is expected
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Files differ, output is in stdout
			return "diff --git a/" + relPath + " b/" + relPath + "\n" + stdout.String(), nil
		}
		// Actual error or files are the same (exit code 0)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 0 {
			return "", nil // No difference
		}
		return "", fmt.Errorf("diff command failed: %v: %s", err, stderr.String())
	}

	// No difference
	return "", nil
}

// isBinary checks if content appears to be binary.
func isBinary(content []byte) bool {
	// Check for null bytes in first 8000 bytes
	checkLen := len(content)
	if checkLen > 8000 {
		checkLen = 8000
	}
	return bytes.Contains(content[:checkLen], []byte{0})
}

// Patcher applies diffs to the canonical repo.
type Patcher struct{}

// NewPatcher creates a new patcher.
func NewPatcher() *Patcher {
	return &Patcher{}
}

// ApplyOptions controls patch application behavior.
type ApplyOptions struct {
	DryRun     bool
	CommitMsg  string
	Author     string
	AllowEmpty bool
}

// ApplyResult contains the outcome of a patch application.
type ApplyResult struct {
	Success      bool
	FilesApplied int
	CommitHash   string
	Errors       []string
}

// ApplyDiff applies a unified diff to the target directory.
func (p *Patcher) ApplyDiff(ctx context.Context, targetDir, diff string, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}

	if diff == "" {
		result.Success = true
		return result, nil
	}

	// Use git apply if in a git repo
	if isGitRepo(targetDir) {
		return p.applyWithGit(ctx, targetDir, diff, opts)
	}

	// Fall back to patch command
	return p.applyWithPatch(ctx, targetDir, diff, opts)
}

// applyWithGit uses git apply for patch application.
func (p *Patcher) applyWithGit(ctx context.Context, targetDir, diff string, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}

	args := []string{"apply", "--stat", "--summary"}
	if opts.DryRun {
		args = append(args, "--check")
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = targetDir
	cmd.Stdin = strings.NewReader(diff)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("git apply failed: %v: %s", err, stderr.String()))
		return result, nil
	}

	if opts.DryRun {
		result.Success = true
		return result, nil
	}

	// Actually apply the patch
	cmd = exec.CommandContext(ctx, "git", "apply")
	cmd.Dir = targetDir
	cmd.Stdin = strings.NewReader(diff)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("git apply failed: %v: %s", err, stderr.String()))
		return result, nil
	}

	result.Success = true

	// Create commit if message provided
	if opts.CommitMsg != "" {
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
func (p *Patcher) applyWithPatch(ctx context.Context, targetDir, diff string, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}

	args := []string{"-p1", "-d", targetDir}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	cmd := exec.CommandContext(ctx, "patch", args...)
	cmd.Stdin = strings.NewReader(diff)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("patch failed: %v: %s", err, stderr.String()))
		return result, nil
	}

	result.Success = true
	return result, nil
}

// createCommit creates a git commit with the applied changes.
func (p *Patcher) createCommit(ctx context.Context, targetDir string, opts ApplyOptions) (string, error) {
	// Stage all changes
	cmd := exec.CommandContext(ctx, "git", "add", "-A")
	cmd.Dir = targetDir
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git add failed: %w", err)
	}

	// Create commit
	args := []string{"commit", "-m", opts.CommitMsg}
	if opts.Author != "" {
		args = append(args, "--author", opts.Author)
	}
	if opts.AllowEmpty {
		args = append(args, "--allow-empty")
	}

	cmd = exec.CommandContext(ctx, "git", args...)
	cmd.Dir = targetDir
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git commit failed: %w", err)
	}

	// Get commit hash
	cmd = exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = targetDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// isGitRepo checks if a directory is a git repository.
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
