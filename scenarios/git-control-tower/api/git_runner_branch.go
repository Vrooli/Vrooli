package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (r *ExecGitRunner) Branches(ctx context.Context, repoDir string) ([]byte, error) {
	args := []string{
		"-C", repoDir,
		"for-each-ref",
		"--format=" + branchRefFormat,
		"refs/heads",
		"refs/remotes",
	}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}

	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("git for-each-ref failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git for-each-ref failed: %w", err)
}

func (r *ExecGitRunner) CreateBranch(ctx context.Context, repoDir string, name string, from string) error {
	args := []string{"-C", repoDir, "branch", name}
	if strings.TrimSpace(from) != "" {
		args = append(args, from)
	}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git branch failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git branch failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) CheckoutBranch(ctx context.Context, repoDir string, name string) error {
	args := []string{"-C", repoDir, "checkout", name}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git checkout failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git checkout failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) TrackRemoteBranch(ctx context.Context, repoDir string, remote string, name string) error {
	remote = strings.TrimSpace(remote)
	branch := strings.TrimSpace(name)
	if remote == "" || branch == "" {
		return fmt.Errorf("remote and branch are required")
	}

	args := []string{"-C", repoDir, "checkout", "-b", branch, "--track", fmt.Sprintf("%s/%s", remote, branch)}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git checkout --track failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git checkout --track failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) CheckRefFormat(ctx context.Context, repoDir string, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("branch name is required")
	}
	args := []string{"-C", repoDir, "check-ref-format", "--branch", name}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git check-ref-format failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git check-ref-format failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) SetUpstream(ctx context.Context, repoDir string, branch string, upstream string) error {
	branch = strings.TrimSpace(branch)
	upstream = strings.TrimSpace(upstream)
	if branch == "" || upstream == "" {
		return fmt.Errorf("branch and upstream are required")
	}
	args := []string{"-C", repoDir, "branch", "--set-upstream-to", upstream, branch}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return fmt.Errorf("git branch --set-upstream-to failed: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("git branch --set-upstream-to failed: %w", err)
	}
	return nil
}

func (r *ExecGitRunner) ListStagedFiles(ctx context.Context, repoDir string) ([]string, error) {
	args := []string{"-C", repoDir, "diff", "--cached", "--name-only"}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git diff --cached --name-only failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git diff --cached --name-only failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

func (r *ExecGitRunner) ListTrackedFiles(ctx context.Context, repoDir string) ([]string, error) {
	// git ls-files --cached returns all tracked files
	args := []string{"-C", repoDir, "ls-files", "--cached"}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git ls-files failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

func (r *ExecGitRunner) ListUntrackedFiles(ctx context.Context, repoDir string) ([]string, error) {
	// git ls-files --others --exclude-standard returns untracked files (respects .gitignore)
	args := []string{"-C", repoDir, "ls-files", "--others", "--exclude-standard"}
	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git ls-files --others failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git ls-files --others failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

func (r *ExecGitRunner) CatFile(ctx context.Context, repoDir string, path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("file path is required")
	}
	// Read file directly from working tree
	absPath := repoDir + "/" + path
	return os.ReadFile(absPath)
}

func (r *ExecGitRunner) GrepContent(ctx context.Context, repoDir string, opts GrepOptions) ([]byte, error) {
	if strings.TrimSpace(opts.Pattern) == "" {
		return nil, fmt.Errorf("search pattern is required")
	}

	args := buildGrepArgs(repoDir, opts)

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	if err != nil {
		return handleGrepError(err)
	}

	return out, nil
}

// buildGrepArgs constructs the git grep argument list from options.
func buildGrepArgs(repoDir string, opts GrepOptions) []string {
	args := []string{"-C", repoDir, "grep", "-n", "--no-color"}

	if !opts.CaseSensitive {
		args = append(args, "-i")
	}
	if opts.WholeWord {
		args = append(args, "-w")
	}
	if opts.ExtendedRegex {
		args = append(args, "-E")
	}
	if opts.ContextLines > 0 {
		args = append(args, fmt.Sprintf("-C%d", opts.ContextLines))
	}
	if opts.MaxCount > 0 {
		args = append(args, fmt.Sprintf("-m%d", opts.MaxCount))
	}

	args = append(args, "-e", opts.Pattern)
	args = appendPathspecs(args, opts.IncludeGlobs, opts.ExcludeGlobs)

	return args
}

// appendPathspecs adds include/exclude glob pathspecs to the argument list.
func appendPathspecs(args []string, includeGlobs, excludeGlobs []string) []string {
	if len(includeGlobs) == 0 && len(excludeGlobs) == 0 {
		return args
	}

	args = append(args, "--")
	for _, glob := range includeGlobs {
		if strings.TrimSpace(glob) != "" {
			args = append(args, glob)
		}
	}
	for _, glob := range excludeGlobs {
		if strings.TrimSpace(glob) != "" {
			args = append(args, ":^"+glob)
		}
	}
	return args
}

// handleGrepError interprets git grep exit codes, treating exit code 1 (no matches) as success.
func handleGrepError(err error) ([]byte, error) {
	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 1 {
			return []byte{}, nil
		}
		return nil, fmt.Errorf("git grep failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git grep failed: %w", err)
}
