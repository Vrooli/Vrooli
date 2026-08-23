package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
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

// readArgs builds command arguments for read-only git operations.
// Prepends --no-optional-locks to prevent git from acquiring the index
// lock for optional stat-cache refreshes (primarily git status and git diff).
// This allows reads to run concurrently with index-modifying writes
// without contending for .git/index.lock.
func readArgs(repoDir string, args ...string) []string {
	result := make([]string, 0, len(args)+3)
	result = append(result, "--no-optional-locks", "-C", repoDir)
	result = append(result, args...)
	return result
}

func (r *ExecGitRunner) StatusPorcelainV2(ctx context.Context, repoDir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx,
		r.gitPath(),
		readArgs(repoDir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "-z")...,
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
	args := readArgs(repoDir, "diff", "--no-color")
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
	build := func() *exec.Cmd {
		args := []string{"-C", repoDir, "add", "--"}
		args = append(args, paths...)
		return exec.CommandContext(ctx, r.gitPath(), args...)
	}
	// Capture stderr separately so warnings can be parsed; the retry wrapper
	// inspects these bytes for index.lock contention.
	run := func(cmd *exec.Cmd) ([]byte, error) {
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		e := cmd.Run()
		return stderr.Bytes(), e
	}
	stderrBytes, err := execWithIndexLockRetry(ctx, build, run)

	// Parse stderr for warnings even if command succeeded
	var warnings []string
	stderrStr := strings.TrimSpace(string(stderrBytes))
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
	build := func() *exec.Cmd {
		args := []string{"-C", repoDir, "reset", "HEAD", "--"}
		args = append(args, paths...)
		return exec.CommandContext(ctx, r.gitPath(), args...)
	}
	out, err := execWithIndexLockRetry(ctx, build, runCombined)
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
	// Create the commit. index.lock contention fails before any commit is
	// written, so retrying on that error is safe (nothing was committed).
	build := func() *exec.Cmd {
		args := []string{"-C", repoDir, "commit"}
		if options.Amend {
			args = append(args, "--amend")
		}
		if options.NoVerify {
			args = append(args, "--no-verify")
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
		return cmd
	}
	out, err := execWithIndexLockRetry(ctx, build, runCombined)
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git commit failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return "", fmt.Errorf("git commit failed: %w", err)
	}

	// Get the commit hash using rev-parse HEAD
	hashCmd := exec.CommandContext(ctx, r.gitPath(), "-C", repoDir, "rev-parse", "HEAD")
	hashOut, err := hashCmd.Output()
	if err != nil {
		// Commit succeeded but couldn't get hash - return empty string
		return "", nil
	}

	return strings.TrimSpace(string(hashOut)), nil
}

func (r *ExecGitRunner) RevParse(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	cmdArgs := readArgs(repoDir, "rev-parse")
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
	cmd := exec.CommandContext(ctx, r.gitPath(), readArgs(repoDir, "log", "-1", "--pretty=%s")...)
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

func (r *ExecGitRunner) ResolveRepoRoot(ctx context.Context) string {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err == nil {
		return root
	}

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
	cmd := exec.CommandContext(ctx, r.gitPath(), readArgs(repoDir, "config", "--get", key)...)
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
		build := func() *exec.Cmd {
			args := []string{"-C", repoDir, "clean", "-f", "--"}
			args = append(args, paths...)
			return exec.CommandContext(ctx, r.gitPath(), args...)
		}
		out, err := execWithIndexLockRetry(ctx, build, runCombined)
		if err != nil {
			exitErr := &exec.ExitError{}
			if errors.As(err, &exitErr) {
				return fmt.Errorf("git clean failed: %w (%s)", err, strings.TrimSpace(string(out)))
			}
			return fmt.Errorf("git clean failed: %w", err)
		}
	} else {
		// For tracked files, use git checkout -- <paths>
		build := func() *exec.Cmd {
			args := []string{"-C", repoDir, "checkout", "--"}
			args = append(args, paths...)
			return exec.CommandContext(ctx, r.gitPath(), args...)
		}
		out, err := execWithIndexLockRetry(ctx, build, runCombined)
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

func (r *ExecGitRunner) DiffNumstat(ctx context.Context, repoDir string, staged bool, paths ...string) ([]byte, error) {
	args := readArgs(repoDir, "diff", "--numstat", "--no-color", "-z")
	if staged {
		args = append(args, "--cached")
	}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
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
	build := func() *exec.Cmd {
		args := []string{"-C", repoDir, "rm", "--cached", "--ignore-unmatch", "--"}
		args = append(args, paths...)
		return exec.CommandContext(ctx, r.gitPath(), args...)
	}
	out, err := execWithIndexLockRetry(ctx, build, runCombined)
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git rm --cached failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git rm --cached failed: %w", err)
	}
	return nil
}
