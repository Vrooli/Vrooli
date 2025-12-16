package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type GitRunner interface {
	StatusPorcelainV2(ctx context.Context, repoDir string) ([]byte, error)
	Diff(ctx context.Context, repoDir string, path string, staged bool) ([]byte, error)
	Stage(ctx context.Context, repoDir string, paths []string) error
	Unstage(ctx context.Context, repoDir string, paths []string) error
}

type ExecGitRunner struct {
	GitPath string
}

func (r *ExecGitRunner) StatusPorcelainV2(ctx context.Context, repoDir string) ([]byte, error) {
	gitPath := strings.TrimSpace(r.GitPath)
	if gitPath == "" {
		gitPath = "git"
	}

	cmd := exec.CommandContext(ctx,
		gitPath,
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
	gitPath := strings.TrimSpace(r.GitPath)
	if gitPath == "" {
		gitPath = "git"
	}

	args := []string{"-C", repoDir, "diff", "--no-color"}
	if staged {
		args = append(args, "--cached")
	}
	if path != "" {
		args = append(args, "--", path)
	}

	cmd := exec.CommandContext(ctx, gitPath, args...)
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
	gitPath := strings.TrimSpace(r.GitPath)
	if gitPath == "" {
		gitPath = "git"
	}

	args := []string{"-C", repoDir, "add", "--"}
	args = append(args, paths...)

	cmd := exec.CommandContext(ctx, gitPath, args...)
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
	gitPath := strings.TrimSpace(r.GitPath)
	if gitPath == "" {
		gitPath = "git"
	}

	args := []string{"-C", repoDir, "reset", "HEAD", "--"}
	args = append(args, paths...)

	cmd := exec.CommandContext(ctx, gitPath, args...)
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
