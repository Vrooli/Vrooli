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
