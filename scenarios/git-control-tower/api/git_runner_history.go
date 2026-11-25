package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func (r *ExecGitRunner) LogGraph(ctx context.Context, repoDir string, limit int, grep string) ([]byte, error) {
	if limit <= 0 {
		limit = 30
	}
	args := readArgs(repoDir,
		"log",
		"--graph",
		"--oneline",
		"--decorate",
		"--color=never",
		"-n", fmt.Sprintf("%d", limit),
	)
	if grep != "" {
		args = append(args, "--fixed-strings", "--grep="+grep)
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("git log failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git log failed: %w", err)
}

func (r *ExecGitRunner) LogDetails(ctx context.Context, repoDir string, limit int, grep string) ([]byte, error) {
	if limit <= 0 {
		limit = 30
	}
	args := readArgs(repoDir,
		"log",
		"--name-only",
		"-z",
		"--pretty=format:%x1e%H%x00%an%x00%ad%x00%B%x00",
		"--date=iso",
		"-n", fmt.Sprintf("%d", limit),
	)
	if grep != "" {
		args = append(args, "--fixed-strings", "--grep="+grep)
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("git log --name-only failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git log --name-only failed: %w", err)
}

func (r *ExecGitRunner) ShowCommitDiff(ctx context.Context, repoDir string, commit string, path string) ([]byte, error) {
	if strings.TrimSpace(commit) == "" {
		return nil, fmt.Errorf("commit hash is required")
	}

	// diff-tree handles root commits and compares merge commits with their
	// first parent. This keeps the history diff to one Git subprocess instead
	// of probing for a parent before running the diff.
	args := readArgs(repoDir,
		"diff-tree",
		"--root",
		"--no-commit-id",
		"-p",
		"--diff-merges=first-parent",
		"-r",
		"--no-color",
		commit,
	)

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
		return nil, fmt.Errorf("git diff/show failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git diff/show failed: %w", err)
}

func (r *ExecGitRunner) ShowFileAtCommit(ctx context.Context, repoDir string, commit string, path string) ([]byte, error) {
	if strings.TrimSpace(commit) == "" {
		return nil, fmt.Errorf("commit hash is required")
	}
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Use git show <commit>:<path> to get file content at that commit
	object := fmt.Sprintf("%s:%s", commit, path)
	if size, err := r.catFileSize(ctx, repoDir, object); err == nil {
		if size > maxDiffFileBytes {
			return nil, &FileTooLargeError{Path: path, Size: size, Limit: maxDiffFileBytes}
		}
	} else {
		return nil, err
	}

	args := readArgs(repoDir, "show", object)

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("git show failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git show failed: %w", err)
}

func (r *ExecGitRunner) catFileSize(ctx context.Context, repoDir string, object string) (int64, error) {
	args := readArgs(repoDir, "cat-file", "-s", object)
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return 0, fmt.Errorf("git cat-file -s failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return 0, fmt.Errorf("git cat-file -s failed: %w", err)
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse git cat-file size failed: %w", err)
	}
	return size, nil
}

func (r *ExecGitRunner) LogFileFrequency(ctx context.Context, repoDir string, commitLimit int) (map[string]int, error) {
	if commitLimit <= 0 {
		commitLimit = 50
	}
	args := readArgs(repoDir,
		"log",
		"--name-only",
		"--pretty=format:",
		"-n", fmt.Sprintf("%d", commitLimit),
	)
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log --name-only failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log --name-only failed: %w", err)
	}

	freq := map[string]int{}
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			freq[name]++
		}
	}
	return freq, nil
}
