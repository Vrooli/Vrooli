package main

import (
	"context"
	"fmt"
	"os"
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
	defaultEnv := append(base, "GIT_TERMINAL_PROMPT=0")

	if cred == nil {
		return defaultEnv, noop, nil
	}

	switch cred.Type {
	case CredentialTypeSSH:
		return sshCredentialEnv(base, cred.SSHKeyPath)
	case CredentialTypeHTTPS:
		return httpsCredentialEnv(base, cred.Username, cred.Token)
	default:
		return defaultEnv, noop, nil
	}
}

// sshCredentialEnv builds environment variables for SSH-based git authentication.
func sshCredentialEnv(base []string, keyPath string) ([]string, func(), error) {
	noop := func() {}
	keyPath = strings.TrimSpace(keyPath)
	if keyPath == "" {
		return append(base, "GIT_TERMINAL_PROMPT=0"), noop, nil
	}
	if _, statErr := os.Stat(keyPath); statErr != nil {
		return nil, noop, fmt.Errorf("SSH key not found: %s", keyPath)
	}
	sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new", keyPath)
	env := append(base,
		fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd),
		"GIT_TERMINAL_PROMPT=0",
	)
	return env, noop, nil
}

// httpsCredentialEnv builds environment variables and a temp askpass script for HTTPS-based git authentication.
func httpsCredentialEnv(base []string, username, token string) ([]string, func(), error) {
	noop := func() {}
	if username == "" || token == "" {
		return append(base, "GIT_TERMINAL_PROMPT=0"), noop, nil
	}
	env := append(base,
		fmt.Sprintf("GIT_USERNAME=%s", username),
		fmt.Sprintf("GIT_PASSWORD=%s", token),
		"GIT_TERMINAL_PROMPT=0",
	)
	askpassPath, err := createAskpassScript()
	if err != nil {
		return nil, noop, err
	}
	env = append(env, fmt.Sprintf("GIT_ASKPASS=%s", askpassPath))
	return env, func() { os.Remove(askpassPath) }, nil
}

// createAskpassScript writes a temporary shell script that echoes GIT_USERNAME/GIT_PASSWORD.
func createAskpassScript() (string, error) {
	askpassScript := "#!/bin/sh\ncase \"$1\" in\n  *[Uu]sername*) echo \"$GIT_USERNAME\" ;;\n  *[Pp]assword*) echo \"$GIT_PASSWORD\" ;;\nesac"
	tmpFile, err := os.CreateTemp("", "git-askpass-*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create askpass script: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, wErr := tmpFile.WriteString(askpassScript); wErr != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to write askpass script: %w", wErr)
	}
	tmpFile.Close()
	if chErr := os.Chmod(tmpPath, 0o700); chErr != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to make askpass script executable: %w", chErr)
	}
	return tmpPath, nil
}
