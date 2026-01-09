// Package diff provides diff generation and patch application for sandboxes.
//
// # Seam: GitOps Interface
//
// This file defines the GitOps interface and implementations for git operations
// that are used at the package level. This provides a seam for test isolation
// when testing code that needs to interact with git repositories.
//
// # Why This Seam Exists
//
// The service layer calls package-level functions like GetGitCommitHash,
// CheckForConflicts, and ReconcilePendingWithGit. These functions directly
// execute git commands, which makes testing dangerous and difficult.
//
// With GitOps, the service can inject a mock implementation in tests,
// completely avoiding real git operations.
//
// # Usage in Production
//
//	gitOps := diff.NewGitOps()
//	hash, err := gitOps.GetCommitHash(ctx, repoDir)
//
// # Usage in Tests
//
//	mock := diff.NewMockGitOps()
//	mock.CommitHash = "abc123"
//	mock.IsRepo = true
//	// inject mock into service
package diff

import (
	"context"
	"strings"

	"workspace-sandbox/internal/types"
)

// GitOperations defines the interface for git operations.
// This is the primary seam for test isolation of git-related functionality.
type GitOperations interface {
	// IsGitRepo checks if a directory is a git repository.
	IsGitRepo(ctx context.Context, dir string) bool

	// GetCommitHash returns the current HEAD commit hash for a git repository.
	// Returns empty string if the directory is not a git repo.
	GetCommitHash(ctx context.Context, repoDir string) (string, error)

	// CheckRepoChanged compares the current repo commit hash against a base hash.
	// Returns (hasChanged, currentHash, error).
	CheckRepoChanged(ctx context.Context, repoDir, baseHash string) (bool, string, error)

	// GetChangedFilesSince returns files changed in the repo since a specific commit.
	GetChangedFilesSince(ctx context.Context, repoDir, baseCommit string) ([]string, error)

	// GetUncommittedFiles returns all uncommitted files from git status.
	GetUncommittedFiles(ctx context.Context, repoDir string) ([]GitFileStatus, error)

	// GetUncommittedFilePaths returns just the paths of uncommitted files.
	GetUncommittedFilePaths(ctx context.Context, repoDir string) ([]string, error)

	// CheckForConflicts performs a comprehensive conflict detection check.
	CheckForConflicts(ctx context.Context, s *types.Sandbox, sandboxChanges []*types.FileChange) (*ConflictCheckResult, error)

	// ReconcilePendingWithGit compares database pending files with actual git status.
	ReconcilePendingWithGit(ctx context.Context, repoDir string, pendingPaths []string) (*ReconcileResult, error)
}

// GitOps is the production implementation of GitOperations.
// It uses a CommandRunner to execute git commands.
type GitOps struct {
	runner CommandRunner
}

// NewGitOps creates a GitOps with the default command runner.
func NewGitOps() *GitOps {
	return &GitOps{runner: DefaultCommandRunner()}
}

// NewGitOpsWithRunner creates a GitOps with a custom command runner.
// This is the primary seam for test isolation.
func NewGitOpsWithRunner(runner CommandRunner) *GitOps {
	return &GitOps{runner: runner}
}

// IsGitRepo checks if a directory is a git repository.
func (g *GitOps) IsGitRepo(ctx context.Context, dir string) bool {
	result := g.runner.Run(ctx, "", "", "git", "-C", dir, "rev-parse", "--git-dir")
	return result.Err == nil
}

// GetCommitHash returns the current HEAD commit hash for a git repository.
func (g *GitOps) GetCommitHash(ctx context.Context, repoDir string) (string, error) {
	if !g.IsGitRepo(ctx, repoDir) {
		return "", nil // Not a git repo, no commit hash available
	}

	result := g.runner.Run(ctx, "", "", "git", "-C", repoDir, "rev-parse", "HEAD")
	if result.Err != nil {
		return "", result.Err
	}

	return strings.TrimSpace(result.Stdout), nil
}

// CheckRepoChanged compares the current repo commit hash against a base hash.
func (g *GitOps) CheckRepoChanged(ctx context.Context, repoDir, baseHash string) (bool, string, error) {
	if baseHash == "" {
		return false, "", nil // No base hash to compare against
	}

	currentHash, err := g.GetCommitHash(ctx, repoDir)
	if err != nil {
		return false, "", err
	}

	if currentHash == "" {
		return false, "", nil // Not a git repo anymore
	}

	return currentHash != baseHash, currentHash, nil
}

// GetChangedFilesSince returns files changed in the repo since a specific commit.
func (g *GitOps) GetChangedFilesSince(ctx context.Context, repoDir, baseCommit string) ([]string, error) {
	if !g.IsGitRepo(ctx, repoDir) || baseCommit == "" {
		return nil, nil
	}

	result := g.runner.Run(ctx, "", "", "git", "-C", repoDir, "diff", "--name-only", baseCommit+"..HEAD")
	if result.Err != nil {
		return nil, result.Err
	}

	output := strings.TrimSpace(result.Stdout)
	if output == "" {
		return nil, nil
	}

	return strings.Split(output, "\n"), nil
}

// GetUncommittedFiles returns all uncommitted files from git status.
func (g *GitOps) GetUncommittedFiles(ctx context.Context, repoDir string) ([]GitFileStatus, error) {
	if !g.IsGitRepo(ctx, repoDir) {
		return nil, nil
	}

	result := g.runner.Run(ctx, "", "", "git", "-C", repoDir, "status", "--porcelain", "-z")
	if result.Err != nil {
		return nil, result.Err
	}

	output := result.Stdout
	if output == "" {
		return nil, nil
	}

	var files []GitFileStatus

	// Split by null character (from -z flag)
	entries := strings.Split(output, "\x00")
	for _, entry := range entries {
		if len(entry) < 3 {
			continue
		}

		// First two chars are the status codes
		indexState := string(entry[0])
		workTree := string(entry[1])
		path := strings.TrimSpace(entry[3:])

		// Handle rename entries (have " -> " in path)
		if idx := strings.Index(path, " -> "); idx > 0 {
			path = path[idx+4:] // Use the new path
		}

		if path == "" {
			continue
		}

		status := GitFileStatus{
			Path:       path,
			IndexState: indexState,
			WorkTree:   workTree,
			IsStaged:   indexState != " " && indexState != "?",
			IsDirty:    workTree != " " && workTree != "?",
		}
		files = append(files, status)
	}

	return files, nil
}

// GetUncommittedFilePaths returns just the paths of uncommitted files.
func (g *GitOps) GetUncommittedFilePaths(ctx context.Context, repoDir string) ([]string, error) {
	files, err := g.GetUncommittedFiles(ctx, repoDir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	return paths, nil
}

// CheckForConflicts performs a comprehensive conflict detection check.
func (g *GitOps) CheckForConflicts(ctx context.Context, s *types.Sandbox, sandboxChanges []*types.FileChange) (*ConflictCheckResult, error) {
	result := &ConflictCheckResult{
		BaseCommitHash: s.BaseCommitHash,
	}

	// If no base commit hash, we can't detect conflicts
	if s.BaseCommitHash == "" {
		return result, nil
	}

	// Check if repo has changed
	changed, currentHash, err := g.CheckRepoChanged(ctx, s.ProjectRoot, s.BaseCommitHash)
	if err != nil {
		return nil, err
	}

	result.HasChanged = changed
	result.CurrentHash = currentHash

	if !changed {
		return result, nil
	}

	// Get list of files changed in repo
	repoChangedFiles, err := g.GetChangedFilesSince(ctx, s.ProjectRoot, s.BaseCommitHash)
	if err != nil {
		return nil, err
	}
	result.RepoChangedFiles = repoChangedFiles

	// Find conflicting files
	result.ConflictingFiles = FindConflictingFiles(sandboxChanges, repoChangedFiles)

	return result, nil
}

// ReconcilePendingWithGit compares database pending files with actual git status.
func (g *GitOps) ReconcilePendingWithGit(ctx context.Context, repoDir string, pendingPaths []string) (*ReconcileResult, error) {
	uncommitted, err := g.GetUncommittedFilePaths(ctx, repoDir)
	if err != nil {
		return nil, err
	}

	// Build set of uncommitted paths
	uncommittedSet := make(map[string]bool)
	for _, p := range uncommitted {
		uncommittedSet[p] = true
	}

	result := &ReconcileResult{}
	for _, pending := range pendingPaths {
		if uncommittedSet[pending] {
			result.StillPending = append(result.StillPending, pending)
		} else {
			// File is not in uncommitted list - either already committed or deleted
			result.AlreadyCommitted = append(result.AlreadyCommitted, pending)
		}
	}

	return result, nil
}

// --- Mock Implementation for Tests ---

// MockGitOps is a test implementation of GitOperations.
// Configure the fields to control the behavior.
type MockGitOps struct {
	// IsRepo controls IsGitRepo return value
	IsRepo bool

	// CommitHash is returned by GetCommitHash
	CommitHash string

	// CommitHashError is returned as the error from GetCommitHash
	CommitHashError error

	// RepoChanged controls CheckRepoChanged return
	RepoChanged bool

	// CurrentHash is returned by CheckRepoChanged
	CurrentHash string

	// ChangedFiles is returned by GetChangedFilesSince
	ChangedFiles []string

	// ChangedFilesError is returned as error from GetChangedFilesSince
	ChangedFilesError error

	// UncommittedFiles is returned by GetUncommittedFiles
	UncommittedFiles []GitFileStatus

	// UncommittedError is returned as error from GetUncommittedFiles
	UncommittedError error

	// ConflictResult is returned by CheckForConflicts
	ConflictResult *ConflictCheckResult

	// ConflictError is returned as error from CheckForConflicts
	ConflictError error

	// ReconcileResult is returned by ReconcilePendingWithGit
	ReconcileResult *ReconcileResult

	// ReconcileError is returned as error from ReconcilePendingWithGit
	ReconcileError error

	// Calls records all method calls for verification
	Calls []string
}

// NewMockGitOps creates a new mock for testing.
func NewMockGitOps() *MockGitOps {
	return &MockGitOps{
		Calls: []string{},
	}
}

func (m *MockGitOps) IsGitRepo(ctx context.Context, dir string) bool {
	m.Calls = append(m.Calls, "IsGitRepo:"+dir)
	return m.IsRepo
}

func (m *MockGitOps) GetCommitHash(ctx context.Context, repoDir string) (string, error) {
	m.Calls = append(m.Calls, "GetCommitHash:"+repoDir)
	return m.CommitHash, m.CommitHashError
}

func (m *MockGitOps) CheckRepoChanged(ctx context.Context, repoDir, baseHash string) (bool, string, error) {
	m.Calls = append(m.Calls, "CheckRepoChanged:"+repoDir+":"+baseHash)
	return m.RepoChanged, m.CurrentHash, nil
}

func (m *MockGitOps) GetChangedFilesSince(ctx context.Context, repoDir, baseCommit string) ([]string, error) {
	m.Calls = append(m.Calls, "GetChangedFilesSince:"+repoDir+":"+baseCommit)
	return m.ChangedFiles, m.ChangedFilesError
}

func (m *MockGitOps) GetUncommittedFiles(ctx context.Context, repoDir string) ([]GitFileStatus, error) {
	m.Calls = append(m.Calls, "GetUncommittedFiles:"+repoDir)
	return m.UncommittedFiles, m.UncommittedError
}

func (m *MockGitOps) GetUncommittedFilePaths(ctx context.Context, repoDir string) ([]string, error) {
	m.Calls = append(m.Calls, "GetUncommittedFilePaths:"+repoDir)
	files, err := m.GetUncommittedFiles(ctx, repoDir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	return paths, nil
}

func (m *MockGitOps) CheckForConflicts(ctx context.Context, s *types.Sandbox, sandboxChanges []*types.FileChange) (*ConflictCheckResult, error) {
	m.Calls = append(m.Calls, "CheckForConflicts:"+s.ID.String())
	if m.ConflictResult != nil {
		return m.ConflictResult, m.ConflictError
	}
	// Return empty result if not configured
	return &ConflictCheckResult{BaseCommitHash: s.BaseCommitHash}, m.ConflictError
}

func (m *MockGitOps) ReconcilePendingWithGit(ctx context.Context, repoDir string, pendingPaths []string) (*ReconcileResult, error) {
	m.Calls = append(m.Calls, "ReconcilePendingWithGit:"+repoDir)
	if m.ReconcileResult != nil {
		return m.ReconcileResult, m.ReconcileError
	}
	// Return all paths as still pending if not configured
	return &ReconcileResult{StillPending: pendingPaths}, m.ReconcileError
}

// WasCalled returns true if the method was called.
func (m *MockGitOps) WasCalled(method string) bool {
	for _, call := range m.Calls {
		if strings.HasPrefix(call, method) {
			return true
		}
	}
	return false
}

// Reset clears all recorded calls.
func (m *MockGitOps) Reset() {
	m.Calls = []string{}
}

// Verify that implementations satisfy the interface
var _ GitOperations = (*GitOps)(nil)
var _ GitOperations = (*MockGitOps)(nil)
