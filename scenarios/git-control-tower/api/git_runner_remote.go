package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func (r *ExecGitRunner) FetchRemote(ctx context.Context, repoDir string, remote string, cred *StoredCredential) error {
	if remote == "" {
		remote = "origin"
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), "-C", repoDir, "fetch", remote)
	env, cleanup, envErr := gitCredentialEnv(cred)
	if envErr != nil {
		return fmt.Errorf("git fetch credential setup failed: %w", envErr)
	}
	defer cleanup()
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git fetch failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git fetch failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) GetRemoteURL(ctx context.Context, repoDir string, remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), "-C", repoDir, "remote", "get-url", remote)
	out, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git remote get-url failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git remote get-url failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *ExecGitRunner) Push(ctx context.Context, repoDir string, remote string, branch string, setUpstream bool, cred *StoredCredential) error {
	if remote == "" {
		remote = "origin"
	}

	args := []string{"-C", repoDir, "push"}
	if setUpstream {
		args = append(args, "-u")
	}
	args = append(args, remote)
	if branch != "" {
		args = append(args, branch)
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	env, cleanup, envErr := gitCredentialEnv(cred)
	if envErr != nil {
		return fmt.Errorf("git push credential setup failed: %w", envErr)
	}
	defer cleanup()
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git push failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git push failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) Pull(ctx context.Context, repoDir string, remote string, branch string, cred *StoredCredential) error {
	if remote == "" {
		remote = "origin"
	}

	args := []string{"-C", repoDir, "pull", remote}
	if branch != "" {
		args = append(args, branch)
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	env, cleanup, envErr := gitCredentialEnv(cred)
	if envErr != nil {
		return fmt.Errorf("git pull credential setup failed: %w", envErr)
	}
	defer cleanup()
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git pull failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git pull failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) Clone(ctx context.Context, destination string, url string, cred *StoredCredential) error {
	args := []string{"clone", url}
	if strings.TrimSpace(destination) != "" {
		args = append(args, destination)
	}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	env, cleanup, envErr := gitCredentialEnv(cred)
	if envErr != nil {
		return fmt.Errorf("git clone credential setup failed: %w", envErr)
	}
	defer cleanup()
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git clone failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) SetRemoteURL(ctx context.Context, repoDir string, remote string, url string) error {
	remote = strings.TrimSpace(remote)
	url = strings.TrimSpace(url)
	if remote == "" {
		return fmt.Errorf("remote name is required")
	}
	if url == "" {
		return fmt.Errorf("URL is required")
	}

	args := []string{"-C", repoDir, "remote", "set-url", remote, url}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git remote set-url failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git remote set-url failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) LsRemote(ctx context.Context, repoDir string, remote string, cred *StoredCredential) error {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		remote = "origin"
	}

	args := []string{"-C", repoDir, "ls-remote", "--heads", "--exit-code", remote}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)

	env, cleanup, envErr := gitCredentialEnv(cred)
	if envErr != nil {
		return fmt.Errorf("git ls-remote credential setup failed: %w", envErr)
	}
	defer cleanup()
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git ls-remote failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git ls-remote failed: %w", err)
	}
	return nil
}
