package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// remoteCommandError shapes the error from a network git command. A deadline that
// expires mid-transfer kills git with a bare "signal: killed", which tells the operator
// nothing; naming the timeout is what makes the failure actionable.
func remoteCommandError(ctx context.Context, operation string, out []byte, err error) error {
	detail := strings.TrimSpace(string(out))
	if ctxErr := ctx.Err(); ctxErr != nil {
		if detail == "" {
			return fmt.Errorf("git %s timed out before the transfer finished: %w", operation, ctxErr)
		}
		return fmt.Errorf("git %s timed out before the transfer finished: %w (%s)", operation, ctxErr, detail)
	}
	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		return fmt.Errorf("git %s failed: %w (%s)", operation, err, detail)
	}
	return fmt.Errorf("git %s failed: %w", operation, err)
}

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
		return remoteCommandError(ctx, "fetch", out, err)
	}
	return nil
}

func (r *ExecGitRunner) FetchRemoteBranch(ctx context.Context, repoDir string, remote string, branch string, cred *StoredCredential) error {
	if remote == "" {
		remote = "origin"
	}
	branch = strings.TrimPrefix(strings.TrimSpace(branch), "refs/heads/")
	if branch == "" {
		return r.FetchRemote(ctx, repoDir, remote, cred)
	}

	// An explicit refspec keeps the remote-tracking ref authoritative even when the
	// remote has no configured fetch refspec covering this branch.
	refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", branch, remote, branch)
	cmd := exec.CommandContext(ctx, r.gitPath(), "-C", repoDir, "fetch", remote, refspec)
	env, cleanup, envErr := gitCredentialEnv(cred)
	if envErr != nil {
		return fmt.Errorf("git fetch credential setup failed: %w", envErr)
	}
	defer cleanup()
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		return remoteCommandError(ctx, "fetch", out, err)
	}
	return nil
}

func (r *ExecGitRunner) GetRemoteURL(ctx context.Context, repoDir string, remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), readArgs(repoDir, "remote", "get-url", remote)...)
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

func (r *ExecGitRunner) Push(ctx context.Context, repoDir string, remote string, branch string, sourceOID string, setUpstream bool, cred *StoredCredential) error {
	if remote == "" {
		remote = "origin"
	}

	if branch == "" || strings.HasPrefix(remote, "-") || len(sourceOID) != 40 && len(sourceOID) != 64 {
		return fmt.Errorf("exact source commit and destination are required")
	}
	localBranch := ""
	if setUpstream {
		local, e := r.RevParse(ctx, repoDir, "--symbolic-full-name", "HEAD")
		if e != nil {
			return e
		}
		localBranch = strings.TrimSpace(string(local))
		if !strings.HasPrefix(localBranch, "refs/heads/") {
			return fmt.Errorf("upstream setup requires a local branch")
		}
	}
	// Pin the inspected commit even if another writer advances HEAD during push.
	args := []string{"-C", repoDir, "push", "--", remote, sourceOID + ":refs/heads/" + branch}

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	env, cleanup, envErr := gitCredentialEnv(cred)
	if envErr != nil {
		return fmt.Errorf("git push credential setup failed: %w", envErr)
	}
	defer cleanup()
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		return remoteCommandError(ctx, "push", out, err)
	}
	if setUpstream {
		current, e := r.RevParse(ctx, repoDir, "--verify", localBranch)
		if e != nil || strings.TrimSpace(string(current)) != sourceOID {
			return fmt.Errorf("push completed, but source changed; upstream setup needs review")
		}
		if e := r.SetUpstream(ctx, repoDir, strings.TrimPrefix(localBranch, "refs/heads/"), remote+"/"+branch); e != nil {
			return fmt.Errorf("push completed but upstream setup failed: %w", e)
		}
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
		return remoteCommandError(ctx, "pull", out, err)
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

	args := readArgs(repoDir, "ls-remote", "--heads", "--exit-code", remote)
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
