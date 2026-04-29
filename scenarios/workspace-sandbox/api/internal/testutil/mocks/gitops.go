package mocks

import (
	"context"
	"strings"
	"sync"

	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/types"
)

// FakeGitOps is the canonical diff.GitOperations implementation for
// tests. It records calls, supports per-method state and error
// injection, and produces sensible default results so tests that only
// care about a subset of git interactions don't need to fill out the
// whole surface.
//
// Note: an older diff.MockGitOps lives inside the diff package
// itself. That impl is unused outside diff's own tests; new tests
// (and the consolidation in Round 4) use FakeGitOps from testutil.
type FakeGitOps struct {
	mu sync.Mutex

	// Default responses
	IsRepo           bool
	CommitHash       string
	CurrentHash      string
	RepoChanged      bool
	ChangedFiles     []string
	UncommittedFiles []diff.GitFileStatus
	UncommittedPaths []string
	ConflictResult   *diff.ConflictCheckResult
	ReconcileResult  *diff.ReconcileResult

	// Per-method error injection
	GetCommitHashErr    error
	CheckRepoChangedErr error
	GetChangedFilesErr  error
	GetUncommittedErr   error
	CheckConflictsErr   error
	ReconcilePendingErr error

	// Calls records every call as `Method:dir[:arg]`. Tests use
	// WasCalled / Reset / Calls() for verification.
	calls []string
}

// NewFakeGitOps returns a FakeGitOps with sane defaults: IsRepo=true,
// CommitHash=abc123, no errors. The default ConflictResult/ReconcileResult
// are populated lazily so tests that don't override them get
// "nothing changed" semantics.
func NewFakeGitOps() *FakeGitOps {
	return &FakeGitOps{
		IsRepo:           true,
		CommitHash:       "abc123",
		CurrentHash:      "abc123",
		ChangedFiles:     []string{},
		UncommittedFiles: []diff.GitFileStatus{},
		UncommittedPaths: []string{},
		ConflictResult:   &diff.ConflictCheckResult{HasChanged: false},
		ReconcileResult:  &diff.ReconcileResult{StillPending: []string{}},
	}
}

// Calls returns a copy of the recorded call list.
func (m *FakeGitOps) Calls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.calls))
	copy(out, m.calls)
	return out
}

// WasCalled reports whether the named method was invoked.
func (m *FakeGitOps) WasCalled(method string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.calls {
		if strings.HasPrefix(c, method+":") || c == method {
			return true
		}
	}
	return false
}

// Reset clears the recorded call list (state and error knobs are
// untouched).
func (m *FakeGitOps) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = m.calls[:0]
}

func (m *FakeGitOps) record(call string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, call)
}

// --- diff.GitOperations implementation ---

func (m *FakeGitOps) IsGitRepo(ctx context.Context, dir string) bool {
	m.record("IsGitRepo:" + dir)
	return m.IsRepo
}

func (m *FakeGitOps) GetCommitHash(ctx context.Context, repoDir string) (string, error) {
	m.record("GetCommitHash:" + repoDir)
	if m.GetCommitHashErr != nil {
		return "", m.GetCommitHashErr
	}
	return m.CommitHash, nil
}

func (m *FakeGitOps) CheckRepoChanged(ctx context.Context, repoDir, baseHash string) (bool, string, error) {
	m.record("CheckRepoChanged:" + repoDir + ":" + baseHash)
	if m.CheckRepoChangedErr != nil {
		return false, "", m.CheckRepoChangedErr
	}
	return m.RepoChanged, m.CurrentHash, nil
}

func (m *FakeGitOps) GetChangedFilesSince(ctx context.Context, repoDir, baseCommit string) ([]string, error) {
	m.record("GetChangedFilesSince:" + repoDir + ":" + baseCommit)
	if m.GetChangedFilesErr != nil {
		return nil, m.GetChangedFilesErr
	}
	return m.ChangedFiles, nil
}

func (m *FakeGitOps) GetUncommittedFiles(ctx context.Context, repoDir string) ([]diff.GitFileStatus, error) {
	m.record("GetUncommittedFiles:" + repoDir)
	if m.GetUncommittedErr != nil {
		return nil, m.GetUncommittedErr
	}
	return m.UncommittedFiles, nil
}

func (m *FakeGitOps) GetUncommittedFilePaths(ctx context.Context, repoDir string) ([]string, error) {
	m.record("GetUncommittedFilePaths:" + repoDir)
	if m.GetUncommittedErr != nil {
		return nil, m.GetUncommittedErr
	}
	if len(m.UncommittedPaths) > 0 {
		return m.UncommittedPaths, nil
	}
	out := make([]string, 0, len(m.UncommittedFiles))
	for _, f := range m.UncommittedFiles {
		out = append(out, f.Path)
	}
	return out, nil
}

func (m *FakeGitOps) CheckForConflicts(ctx context.Context, s *types.Sandbox, sandboxChanges []*types.FileChange) (*diff.ConflictCheckResult, error) {
	id := "<nil>"
	if s != nil {
		id = s.ID.String()
	}
	m.record("CheckForConflicts:" + id)
	if m.CheckConflictsErr != nil {
		return nil, m.CheckConflictsErr
	}
	if m.ConflictResult != nil {
		return m.ConflictResult, nil
	}
	return &diff.ConflictCheckResult{HasChanged: false}, nil
}

func (m *FakeGitOps) ReconcilePendingWithGit(ctx context.Context, repoDir string, pendingPaths []string) (*diff.ReconcileResult, error) {
	m.record("ReconcilePendingWithGit:" + repoDir)
	if m.ReconcilePendingErr != nil {
		return nil, m.ReconcilePendingErr
	}
	if m.ReconcileResult != nil {
		return m.ReconcileResult, nil
	}
	out := make([]string, len(pendingPaths))
	copy(out, pendingPaths)
	return &diff.ReconcileResult{StillPending: out}, nil
}

var _ diff.GitOperations = (*FakeGitOps)(nil)
