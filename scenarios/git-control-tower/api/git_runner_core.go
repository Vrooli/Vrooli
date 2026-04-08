package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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

func (r *ExecGitRunner) Stage(ctx context.Context, repoDir string, paths []string) ([]string, error) {
	args := []string{"-C", repoDir, "add", "--"}
	args = append(args, paths...)

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Parse stderr for warnings even if command succeeded
	var warnings []string
	stderrStr := strings.TrimSpace(stderr.String())
	if stderrStr != "" {
		// Check for known warning patterns that aren't fatal errors
		if strings.Contains(stderrStr, "ignored by one of your .gitignore files") ||
			strings.Contains(stderrStr, "hint:") ||
			strings.Contains(stderrStr, "warning:") {
			warnings = append(warnings, stderrStr)
		}
	}

	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return warnings, fmt.Errorf("git add failed: %w (%s)", err, stderrStr)
		}
		return warnings, fmt.Errorf("git add failed: %w", err)
	}
	return warnings, nil
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

func (r *ExecGitRunner) Commit(ctx context.Context, repoDir string, message string, options CommitOptions) (string, error) {
	// Create the commit
	args := []string{"-C", repoDir, "commit"}
	if options.Amend {
		args = append(args, "--amend")
	}
	if options.NoEdit {
		args = append(args, "--no-edit")
	} else {
		args = append(args, "-m", message)
	}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	if options.AuthorName != "" || options.AuthorEmail != "" {
		env := os.Environ()
		if options.AuthorName != "" {
			env = append(env,
				fmt.Sprintf("GIT_AUTHOR_NAME=%s", options.AuthorName),
				fmt.Sprintf("GIT_COMMITTER_NAME=%s", options.AuthorName),
			)
		}
		if options.AuthorEmail != "" {
			env = append(env,
				fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", options.AuthorEmail),
				fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", options.AuthorEmail),
			)
		}
		cmd.Env = env
	}
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

func (r *ExecGitRunner) LastCommitMessage(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, r.gitPath(), "-C", repoDir, "log", "-1", "--pretty=%s")
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git log failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return "", fmt.Errorf("git log failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
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

func (r *ExecGitRunner) ConfigGet(ctx context.Context, repoDir string, key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("config key is required")
	}
	cmd := exec.CommandContext(ctx, r.gitPath(), "-C", repoDir, "config", "--get", key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git config failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return "", fmt.Errorf("git config failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *ExecGitRunner) Discard(ctx context.Context, repoDir string, paths []string, untracked bool) error {
	if len(paths) == 0 {
		return nil
	}

	if untracked {
		// For untracked files, use git clean -f
		args := []string{"-C", repoDir, "clean", "-f", "--"}
		args = append(args, paths...)
		cmd := exec.CommandContext(ctx, r.gitPath(), args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			exitErr := &exec.ExitError{}
			if errors.As(err, &exitErr) {
				return fmt.Errorf("git clean failed: %w (%s)", err, strings.TrimSpace(string(out)))
			}
			return fmt.Errorf("git clean failed: %w", err)
		}
	} else {
		// For tracked files, use git checkout -- <paths>
		args := []string{"-C", repoDir, "checkout", "--"}
		args = append(args, paths...)
		cmd := exec.CommandContext(ctx, r.gitPath(), args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			exitErr := &exec.ExitError{}
			if errors.As(err, &exitErr) {
				return fmt.Errorf("git checkout failed: %w (%s)", err, strings.TrimSpace(string(out)))
			}
			return fmt.Errorf("git checkout failed: %w", err)
		}
	}

	return nil
}

func (r *ExecGitRunner) DiffNumstat(ctx context.Context, repoDir string, staged bool) ([]byte, error) {
	args := []string{"-C", repoDir, "diff", "--numstat", "--no-color"}
	if staged {
		args = append(args, "--cached")
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("git diff --numstat failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git diff --numstat failed: %w", err)
}

func (r *ExecGitRunner) RemoveFromIndex(ctx context.Context, repoDir string, paths []string) error {
	args := []string{"-C", repoDir, "rm", "--cached", "--ignore-unmatch", "--"}
	args = append(args, paths...)

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git rm --cached failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git rm --cached failed: %w", err)
	}
	return nil
}
