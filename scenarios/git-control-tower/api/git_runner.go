package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
	// Returns warnings (e.g., for ignored files) and an error if staging failed.
	Stage(ctx context.Context, repoDir string, paths []string) (warnings []string, err error)

	// Unstage removes the specified paths from the git index (git reset HEAD).
	Unstage(ctx context.Context, repoDir string, paths []string) error

	// Commit creates a new commit with the given message.
	// Returns the commit hash (short OID) on success.
	Commit(ctx context.Context, repoDir string, message string, options CommitOptions) (string, error)

	// RevParse runs git rev-parse with the given arguments.
	// Used for repository validation (e.g., --is-inside-work-tree).
	RevParse(ctx context.Context, repoDir string, args ...string) ([]byte, error)

	// LastCommitMessage returns the subject of the last commit.
	LastCommitMessage(ctx context.Context, repoDir string) (string, error)

	// LookPath checks if the git binary is available.
	// Returns the full path to git if found, or an error if not.
	LookPath() (string, error)

	// ResolveRepoRoot determines the git repository root directory.
	// Returns the absolute path to the repo root, or empty string if not in a repo.
	// This centralizes repo resolution so it can be mocked in tests.
	ResolveRepoRoot(ctx context.Context) string

	// FetchRemote fetches updates from the remote without merging.
	// Used to get accurate ahead/behind counts.
	// If cred is provided, uses it for authentication (SSH key or HTTPS token).
	FetchRemote(ctx context.Context, repoDir string, remote string, cred *StoredCredential) error

	// GetRemoteURL returns the URL for the specified remote (e.g., "origin").
	GetRemoteURL(ctx context.Context, repoDir string, remote string) (string, error)

	// ConfigGet returns the git config value for the given key.
	ConfigGet(ctx context.Context, repoDir string, key string) (string, error)

	// Discard discards changes in the specified paths.
	// For tracked files: git checkout -- <paths>
	// For untracked files: removes them from the working tree
	Discard(ctx context.Context, repoDir string, paths []string, untracked bool) error

	// Push pushes commits to the remote repository.
	// If cred is provided, uses it for authentication.
	Push(ctx context.Context, repoDir string, remote string, branch string, setUpstream bool, cred *StoredCredential) error

	// Pull pulls commits from the remote repository.
	// If cred is provided, uses it for authentication.
	Pull(ctx context.Context, repoDir string, remote string, branch string, cred *StoredCredential) error

	// Clone clones a remote repository into the destination directory.
	// If cred is provided, uses it for authentication.
	Clone(ctx context.Context, destination string, url string, cred *StoredCredential) error

	// LogGraph returns a git log graph for recent commits.
	// Use a limit to cap the number of log entries.
	// When grep is non-empty, only commits whose message contains the string are returned.
	LogGraph(ctx context.Context, repoDir string, limit int, grep string) ([]byte, error)

	// LogDetails returns structured log details including file lists.
	// When grep is non-empty, only commits whose message contains the string are returned.
	LogDetails(ctx context.Context, repoDir string, limit int, grep string) ([]byte, error)

	// DiffNumstat returns numstat output for changes.
	// If staged is true, returns staged stats (--cached).
	DiffNumstat(ctx context.Context, repoDir string, staged bool) ([]byte, error)

	// RemoveFromIndex removes paths from the git index without deleting working files.
	RemoveFromIndex(ctx context.Context, repoDir string, paths []string) error

	// ShowCommitDiff returns the diff for a specific commit.
	// If path is provided, returns only that file's diff from the commit.
	// Uses git show <commit> or git show <commit> -- <path>.
	ShowCommitDiff(ctx context.Context, repoDir string, commit string, path string) ([]byte, error)

	// ShowFileAtCommit returns the content of a file at a specific commit.
	// Uses git show <commit>:<path>.
	ShowFileAtCommit(ctx context.Context, repoDir string, commit string, path string) ([]byte, error)

	// Branches returns branch metadata for local and remote refs.
	Branches(ctx context.Context, repoDir string) ([]byte, error)

	// CreateBranch creates a new branch from the specified base (or HEAD if empty).
	CreateBranch(ctx context.Context, repoDir string, name string, from string) error

	// CheckoutBranch checks out the specified branch.
	CheckoutBranch(ctx context.Context, repoDir string, name string) error

	// TrackRemoteBranch creates a local branch tracking a remote branch.
	TrackRemoteBranch(ctx context.Context, repoDir string, remote string, name string) error

	// CheckRefFormat validates a branch name using git check-ref-format.
	CheckRefFormat(ctx context.Context, repoDir string, name string) error

	// SetUpstream sets the upstream tracking branch for a local branch.
	SetUpstream(ctx context.Context, repoDir string, branch string, upstream string) error

	// ListStagedFiles returns file paths that are currently staged in the index.
	// Uses git diff --cached --name-only.
	ListStagedFiles(ctx context.Context, repoDir string) ([]string, error)

	// ListTrackedFiles returns all tracked files in the repository.
	// Uses git ls-files --cached.
	ListTrackedFiles(ctx context.Context, repoDir string) ([]string, error)

	// ListUntrackedFiles returns all untracked files (respects .gitignore).
	// Uses git ls-files --others --exclude-standard.
	ListUntrackedFiles(ctx context.Context, repoDir string) ([]string, error)

	// CatFile returns the content of a file in the working tree.
	// Used for reading file content for import scanning.
	CatFile(ctx context.Context, repoDir string, path string) ([]byte, error)

	// SetRemoteURL updates the URL for a remote.
	// Uses git remote set-url <remote> <url>.
	SetRemoteURL(ctx context.Context, repoDir string, remote string, url string) error

	// LsRemote lists references from a remote repository.
	// Used to test connectivity and authentication.
	// If cred is provided, uses GIT_ASKPASS for authentication.
	LsRemote(ctx context.Context, repoDir string, remote string, cred *StoredCredential) error

	// GrepContent searches file contents using git grep.
	// Returns raw output in format: file:line:content
	GrepContent(ctx context.Context, repoDir string, opts GrepOptions) ([]byte, error)

	// LogFileFrequency returns a map of file paths to the number of commits
	// they appeared in over the last commitLimit commits.
	LogFileFrequency(ctx context.Context, repoDir string, commitLimit int) (map[string]int, error)
}

// GrepOptions configures the git grep search.
type GrepOptions struct {
	Pattern       string   // Search pattern (required)
	CaseSensitive bool     // Match case exactly (-i flag inverted)
	WholeWord     bool     // Match whole words only (-w)
	ExtendedRegex bool     // Treat pattern as extended regex (-E)
	IncludeGlobs  []string // Glob patterns to include (--include)
	ExcludeGlobs  []string // Glob patterns to exclude (--exclude)
	ContextLines  int      // Lines of context (-C)
	MaxCount      int      // Max matches per file (-m)
}

// CommitOptions configures author overrides for commit operations.
type CommitOptions struct {
	AuthorName  string
	AuthorEmail string
	Amend       bool
	NoEdit      bool
}

// gitCredentialEnv builds environment variables and a cleanup function for
// authenticated git operations. It supports both SSH and HTTPS credentials.
//
//   - nil cred: returns os.Environ() + GIT_TERMINAL_PROMPT=0, noop cleanup
//   - SSH cred: sets GIT_SSH_COMMAND with -i <key> -o IdentitiesOnly=yes
//   - HTTPS cred: creates a temp askpass script, returns cleanup that removes it
func gitCredentialEnv(cred *StoredCredential) (env []string, cleanup func(), err error) {
	base := os.Environ()
	noop := func() {}

	if cred == nil {
		return append(base, "GIT_TERMINAL_PROMPT=0"), noop, nil
	}

	switch cred.Type {
	case CredentialTypeSSH:
		keyPath := strings.TrimSpace(cred.SSHKeyPath)
		if keyPath == "" {
			return append(base, "GIT_TERMINAL_PROMPT=0"), noop, nil
		}
		if _, statErr := os.Stat(keyPath); statErr != nil {
			return nil, noop, fmt.Errorf("SSH key not found: %s", keyPath)
		}
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new", keyPath)
		env = append(base,
			fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd),
			"GIT_TERMINAL_PROMPT=0",
		)
		return env, noop, nil

	case CredentialTypeHTTPS:
		if cred.Username == "" || cred.Token == "" {
			return append(base, "GIT_TERMINAL_PROMPT=0"), noop, nil
		}
		env = append(base,
			fmt.Sprintf("GIT_USERNAME=%s", cred.Username),
			fmt.Sprintf("GIT_PASSWORD=%s", cred.Token),
			"GIT_TERMINAL_PROMPT=0",
		)
		askpassScript := "#!/bin/sh\ncase \"$1\" in\n  *[Uu]sername*) echo \"$GIT_USERNAME\" ;;\n  *[Pp]assword*) echo \"$GIT_PASSWORD\" ;;\nesac"
		tmpFile, tmpErr := os.CreateTemp("", "git-askpass-*.sh")
		if tmpErr != nil {
			return nil, noop, fmt.Errorf("failed to create askpass script: %w", tmpErr)
		}
		tmpPath := tmpFile.Name()
		if _, wErr := tmpFile.WriteString(askpassScript); wErr != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return nil, noop, fmt.Errorf("failed to write askpass script: %w", wErr)
		}
		tmpFile.Close()
		if chErr := os.Chmod(tmpPath, 0o700); chErr != nil {
			os.Remove(tmpPath)
			return nil, noop, fmt.Errorf("failed to make askpass script executable: %w", chErr)
		}
		env = append(env, fmt.Sprintf("GIT_ASKPASS=%s", tmpPath))
		return env, func() { os.Remove(tmpPath) }, nil

	default:
		return append(base, "GIT_TERMINAL_PROMPT=0"), noop, nil
	}
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

func (r *ExecGitRunner) LogGraph(ctx context.Context, repoDir string, limit int, grep string) ([]byte, error) {
	if limit <= 0 {
		limit = 30
	}
	args := []string{
		"-C", repoDir,
		"log",
		"--graph",
		"--oneline",
		"--decorate",
		"--color=never",
		"-n", fmt.Sprintf("%d", limit),
	}
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
	args := []string{
		"-C", repoDir,
		"log",
		"--name-only",
		"--pretty=format:%H%x00%an%x00%ad%x00%s",
		"--date=iso",
		"-n", fmt.Sprintf("%d", limit),
	}
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

func (r *ExecGitRunner) ShowCommitDiff(ctx context.Context, repoDir string, commit string, path string) ([]byte, error) {
	if strings.TrimSpace(commit) == "" {
		return nil, fmt.Errorf("commit hash is required")
	}

	// Use git diff with explicit parent reference instead of git show.
	// git show produces "combined diffs" for merge commits which often show
	// empty diffs for files that came cleanly from one parent.
	// git diff <commit>^..<commit> explicitly shows changes from first parent.

	// First, check if this commit has a parent
	parentCheck := exec.CommandContext(ctx, r.gitPath(), "-C", repoDir, "rev-parse", "--verify", "--quiet", commit+"^")
	hasParent := parentCheck.Run() == nil

	var args []string
	if hasParent {
		// Normal commit with parent - use git diff
		args = []string{"-C", repoDir, "diff", "--no-color", commit + "^", commit}
	} else {
		// Root commit (no parent) - use git show
		args = []string{"-C", repoDir, "show", "--no-color", "--format=", commit}
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

	args := []string{"-C", repoDir, "show", object}

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
	args := []string{"-C", repoDir, "cat-file", "-s", object}
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

func (r *ExecGitRunner) GrepContent(ctx context.Context, repoDir string, opts GrepOptions) ([]byte, error) {
	if strings.TrimSpace(opts.Pattern) == "" {
		return nil, fmt.Errorf("search pattern is required")
	}

	// Build git grep command
	// Output format: file:line:content (using -n for line numbers)
	args := []string{"-C", repoDir, "grep", "-n", "--no-color"}

	// Case sensitivity (git grep is case-sensitive by default)
	if !opts.CaseSensitive {
		args = append(args, "-i")
	}

	// Whole word matching
	if opts.WholeWord {
		args = append(args, "-w")
	}

	// Extended regex
	if opts.ExtendedRegex {
		args = append(args, "-E")
	}

	// Context lines
	if opts.ContextLines > 0 {
		args = append(args, fmt.Sprintf("-C%d", opts.ContextLines))
	}

	// Max matches per file
	if opts.MaxCount > 0 {
		args = append(args, fmt.Sprintf("-m%d", opts.MaxCount))
	}

	// Add the search pattern (must come before pathspecs)
	args = append(args, "-e", opts.Pattern)

	// Add pathspecs (-- separator then patterns)
	hasPathspecs := len(opts.IncludeGlobs) > 0 || len(opts.ExcludeGlobs) > 0
	if hasPathspecs {
		args = append(args, "--")

		// Add include globs
		for _, glob := range opts.IncludeGlobs {
			if strings.TrimSpace(glob) != "" {
				args = append(args, glob)
			}
		}

		// Add exclude globs using :^pattern syntax
		for _, glob := range opts.ExcludeGlobs {
			if strings.TrimSpace(glob) != "" {
				args = append(args, ":^"+glob)
			}
		}
	}

	cmd := exec.CommandContext(ctx, r.gitPath(), args...)
	out, err := cmd.Output()
	// git grep returns exit code 1 when no matches found - this is not an error
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			// Exit code 1 = no matches, which is valid
			if exitErr.ExitCode() == 1 {
				return []byte{}, nil
			}
			return nil, fmt.Errorf("git grep failed: %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git grep failed: %w", err)
	}

	return out, nil
}

func (r *ExecGitRunner) LogFileFrequency(ctx context.Context, repoDir string, commitLimit int) (map[string]int, error) {
	if commitLimit <= 0 {
		commitLimit = 50
	}
	args := []string{
		"-C", repoDir,
		"log",
		"--name-only",
		"--pretty=format:",
		"-n", fmt.Sprintf("%d", commitLimit),
	}
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
