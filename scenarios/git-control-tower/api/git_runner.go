package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GitRunner abstracts git operations to enable testing without real git.
// This is the primary seam for isolating git side effects.
//
// Production code uses ExecGitRunner which shells out to the git binary.
// Test code can use FakeGitRunner (in git_runner_test.go) to exercise
// domain logic without touching the filesystem or running real git commands.
//
// SEAM BOUNDARY: All git operations must flow through this interface.
// Do not call exec.Command("git", ...) directly outside of implementations.
type GitRunner interface {
	// StatusPorcelainV2 returns git status in porcelain v2 format (-z for NUL-separated).
	StatusPorcelainV2(ctx context.Context, repoDir string) ([]byte, error)

	// Diff returns the diff output for the given path (or all files if empty).
	// If staged is true, returns the staged diff (--cached).
	Diff(ctx context.Context, repoDir string, path string, staged bool) ([]byte, error)

	// Stage adds the specified paths to the git index.
	Stage(ctx context.Context, repoDir string, paths []string) error

	// Unstage removes the specified paths from the git index (git reset HEAD).
	Unstage(ctx context.Context, repoDir string, paths []string) error

	// Commit creates a new commit with the given message.
	// Returns the commit hash (short OID) on success.
	Commit(ctx context.Context, repoDir string, message string) (string, error)

	// RevParse runs git rev-parse with the given arguments.
	// Used for repository validation (e.g., --is-inside-work-tree).
	RevParse(ctx context.Context, repoDir string, args ...string) ([]byte, error)

	// LookPath checks if the git binary is available.
	// Returns the full path to git if found, or an error if not.
	LookPath() (string, error)

	// ResolveRepoRoot determines the git repository root directory.
	// Returns the absolute path to the repo root, or empty string if not in a repo.
	// This centralizes repo resolution so it can be mocked in tests.
	ResolveRepoRoot(ctx context.Context) string
}

// ExecGitRunner implements GitRunner by executing the real git binary.
// This is the production implementation used when the API is running.
type ExecGitRunner struct {
	// GitPath is the path to the git binary. Defaults to "git" if empty.
	GitPath string
}

// gitPath returns the configured git path or "git" as default.
func (r *ExecGitRunner) gitPath() string {
	p := strings.TrimSpace(r.GitPath)
	if p == "" {
		return "git"
	}
	return p
}

func (r *ExecGitRunner) StatusPorcelainV2(ctx context.Context, repoDir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx,
		r.gitPath(),
		"-C", repoDir,
		"status",
		"--porcelain=v2",
		"--branch",
		"--untracked-files=all",
		"-z",
	)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("git status failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git status failed: %w", err)
}

func (r *ExecGitRunner) Diff(ctx context.Context, repoDir string, path string, staged bool) ([]byte, error) {
	args := []string{"-C", repoDir, "diff", "--no-color"}
	if staged {
		args = append(args, "--cached")
	}
	if path != "" {
		args = append(args, "--", path)
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("git diff failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git diff failed: %w", err)
}

func (r *ExecGitRunner) Stage(ctx context.Context, repoDir string, paths []string) error {
	args := []string{"-C", repoDir, "add", "--"}
	args = append(args, paths...)

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git add failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git add failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) Unstage(ctx context.Context, repoDir string, paths []string) error {
	args := []string{"-C", repoDir, "reset", "HEAD", "--"}
	args = append(args, paths...)

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git reset failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git reset failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) Commit(ctx context.Context, repoDir string, message string) (string, error) {
	// Create the commit
	args := []string{"-C", repoDir, "commit", "-m", message}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git commit failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return "", fmt.Errorf("git commit failed: %w", err)
	}

	// Get the commit hash using rev-parse HEAD
	hashCmd := exec.CommandContext(ctx, r.gitPath(), "-C", repoDir, "rev-parse", "--short", "HEAD")
	hashOut, err := hashCmd.Output()
	if err != nil {
		// Commit succeeded but couldn't get hash - return empty string
		return "", nil
	}

	return strings.TrimSpace(string(hashOut)), nil
}

func (r *ExecGitRunner) RevParse(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	cmdArgs := []string{"-C", repoDir, "rev-parse"}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, r.gitPath(), cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git rev-parse failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return nil, fmt.Errorf("git rev-parse failed: %w", err)
	}
	return out, nil
}

func (r *ExecGitRunner) LookPath() (string, error) {
	return exec.LookPath(r.gitPath())
}

// ResolveRepoRoot returns the repository root directory.
// Priority: VROOLI_ROOT env var > git rev-parse --show-toplevel > empty string.
// DECISION BOUNDARY: This determines which repository the API operates on.
func (r *ExecGitRunner) ResolveRepoRoot(ctx context.Context) string {
	// First, check for explicit VROOLI_ROOT configuration
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root
	}

	// Fall back to git's repository detection
	cmd := exec.CommandContext(ctx, r.gitPath(), "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}

	return ""
}
