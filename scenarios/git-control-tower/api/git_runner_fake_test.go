package main

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
)

// FakeGitRunner implements GitRunner without touching the filesystem or running git.
// Use this in tests to exercise domain logic quickly and safely.
//
// The fake maintains in-memory state representing a simulated git repository:
//   - Branch: current branch info
//   - Staged: files in the index
//   - Unstaged: modified files not staged
//   - Untracked: new files not tracked
//   - Conflicts: files with merge conflicts
//
// This enables testing service logic without risk of affecting real repositories.
type FakeGitRunner struct {
	// Repository simulation state
	Branch       FakeBranchState
	Staged       map[string]string // path -> content
	Unstaged     map[string]string // path -> content diff
	Untracked    []string
	Conflicts    []string
	IsRepository bool   // Whether this is a valid git repo
	GitAvailable bool   // Whether git binary is "installed"
	RepoRoot     string // Configured repository root path

	// Error injection for testing error paths
	StatusError   error
	DiffError     error
	StageError    error
	UnstageError  error
	CommitError   error
	RevParseError error
	LookPathError error

	// Commit tracking
	LastCommitMessage string
	CommitCount       int

	// Call tracking for verification
	Calls []FakeGitCall
}

// FakeBranchState represents the simulated branch state.
type FakeBranchState struct {
	Head     string
	Upstream string
	OID      string
	Ahead    int
	Behind   int
}

// FakeGitCall records a call made to the fake for test verification.
type FakeGitCall struct {
	Method string
	Args   []string
}

// NewFakeGitRunner creates a FakeGitRunner with sensible defaults.
// By default, it simulates a valid git repository with git available.
func NewFakeGitRunner() *FakeGitRunner {
	return &FakeGitRunner{
		Branch: FakeBranchState{
			Head: "main",
			OID:  "abc123def456",
		},
		Staged:       make(map[string]string),
		Unstaged:     make(map[string]string),
		Untracked:    []string{},
		Conflicts:    []string{},
		IsRepository: true,
		GitAvailable: true,
		RepoRoot:     "/fake/repo",
		Calls:        []FakeGitCall{},
	}
}

func (f *FakeGitRunner) recordCall(method string, args ...string) {
	f.Calls = append(f.Calls, FakeGitCall{Method: method, Args: args})
}

// StatusPorcelainV2 returns simulated git status output in porcelain v2 format.
func (f *FakeGitRunner) StatusPorcelainV2(ctx context.Context, repoDir string) ([]byte, error) {
	f.recordCall("StatusPorcelainV2", repoDir)

	if f.StatusError != nil {
		return nil, f.StatusError
	}

	var buf bytes.Buffer

	// Branch headers
	buf.WriteString(fmt.Sprintf("# branch.oid %s\x00", f.Branch.OID))
	buf.WriteString(fmt.Sprintf("# branch.head %s\x00", f.Branch.Head))
	if f.Branch.Upstream != "" {
		buf.WriteString(fmt.Sprintf("# branch.upstream %s\x00", f.Branch.Upstream))
		buf.WriteString(fmt.Sprintf("# branch.ab +%d -%d\x00", f.Branch.Ahead, f.Branch.Behind))
	}

	// Staged files (index modified, worktree unchanged)
	for path := range f.Staged {
		// "1 M. N... 100644 100644 100644 hash1 hash2 path"
		buf.WriteString(fmt.Sprintf("1 M. N... 100644 100644 100644 abc123 def456 %s\x00", path))
	}

	// Unstaged files (index unchanged, worktree modified)
	for path := range f.Unstaged {
		buf.WriteString(fmt.Sprintf("1 .M N... 100644 100644 100644 abc123 def456 %s\x00", path))
	}

	// Untracked files
	for _, path := range f.Untracked {
		buf.WriteString(fmt.Sprintf("? %s\x00", path))
	}

	// Conflict files
	for _, path := range f.Conflicts {
		buf.WriteString(fmt.Sprintf("u UU N... 100644 100644 100644 100644 abc def ghi %s\x00", path))
	}

	return buf.Bytes(), nil
}

// Diff returns simulated diff output for the specified path.
func (f *FakeGitRunner) Diff(ctx context.Context, repoDir string, path string, staged bool) ([]byte, error) {
	f.recordCall("Diff", repoDir, path, fmt.Sprintf("staged=%v", staged))

	if f.DiffError != nil {
		return nil, f.DiffError
	}

	var diffs []string

	if staged {
		// Return diffs for staged files
		for p, content := range f.Staged {
			if path == "" || path == p {
				diffs = append(diffs, f.generateDiff(p, content))
			}
		}
	} else {
		// Return diffs for unstaged files
		for p, content := range f.Unstaged {
			if path == "" || path == p {
				diffs = append(diffs, f.generateDiff(p, content))
			}
		}
	}

	// Sort for deterministic output
	sort.Strings(diffs)
	return []byte(strings.Join(diffs, "\n")), nil
}

func (f *FakeGitRunner) generateDiff(path, content string) string {
	if content == "" {
		content = "+added line"
	}
	return fmt.Sprintf(`diff --git a/%s b/%s
index 1234567..abcdef0 100644
--- a/%s
+++ b/%s
@@ -1,1 +1,2 @@
 existing line
%s`, path, path, path, path, content)
}

// Stage simulates adding files to the index.
func (f *FakeGitRunner) Stage(ctx context.Context, repoDir string, paths []string) error {
	f.recordCall("Stage", append([]string{repoDir}, paths...)...)

	if f.StageError != nil {
		return f.StageError
	}

	// Move files from unstaged/untracked to staged
	for _, path := range paths {
		// Check if it's an unstaged modification
		if content, ok := f.Unstaged[path]; ok {
			f.Staged[path] = content
			delete(f.Unstaged, path)
			continue
		}

		// Check if it's untracked
		for i, u := range f.Untracked {
			if u == path {
				f.Staged[path] = "+new file"
				f.Untracked = append(f.Untracked[:i], f.Untracked[i+1:]...)
				break
			}
		}
	}

	return nil
}

// Unstage simulates removing files from the index.
func (f *FakeGitRunner) Unstage(ctx context.Context, repoDir string, paths []string) error {
	f.recordCall("Unstage", append([]string{repoDir}, paths...)...)

	if f.UnstageError != nil {
		return f.UnstageError
	}

	// Move files from staged to unstaged
	for _, path := range paths {
		if content, ok := f.Staged[path]; ok {
			f.Unstaged[path] = content
			delete(f.Staged, path)
		}
	}

	return nil
}

// Commit simulates creating a commit.
func (f *FakeGitRunner) Commit(ctx context.Context, repoDir string, message string) (string, error) {
	f.recordCall("Commit", repoDir, message)

	if f.CommitError != nil {
		return "", f.CommitError
	}

	// Check if there are staged files
	if len(f.Staged) == 0 {
		return "", fmt.Errorf("nothing to commit")
	}

	// Clear staged files (they're now committed)
	f.Staged = make(map[string]string)

	// Track commit
	f.LastCommitMessage = message
	f.CommitCount++

	// Return a fake commit hash
	return fmt.Sprintf("fake%03d", f.CommitCount), nil
}

// RevParse simulates git rev-parse operations.
func (f *FakeGitRunner) RevParse(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	f.recordCall("RevParse", append([]string{repoDir}, args...)...)

	if f.RevParseError != nil {
		return nil, f.RevParseError
	}

	// Handle common rev-parse queries
	for _, arg := range args {
		switch arg {
		case "--is-inside-work-tree":
			if f.IsRepository {
				return []byte("true\n"), nil
			}
			return nil, fmt.Errorf("fatal: not a git repository")
		case "--show-toplevel":
			if f.IsRepository {
				return []byte(repoDir + "\n"), nil
			}
			return nil, fmt.Errorf("fatal: not a git repository")
		}
	}

	return []byte(""), nil
}

// LookPath simulates checking for the git binary.
func (f *FakeGitRunner) LookPath() (string, error) {
	f.recordCall("LookPath")

	if f.LookPathError != nil {
		return "", f.LookPathError
	}
	if !f.GitAvailable {
		return "", fmt.Errorf("executable file not found in $PATH")
	}
	return "/usr/bin/git", nil
}

// ResolveRepoRoot returns the configured repository root.
func (f *FakeGitRunner) ResolveRepoRoot(_ context.Context) string {
	f.recordCall("ResolveRepoRoot")

	if !f.IsRepository {
		return ""
	}
	return f.RepoRoot
}

// --- Test helpers ---

// AddStagedFile adds a file to the staged state.
func (f *FakeGitRunner) AddStagedFile(path string) *FakeGitRunner {
	f.Staged[path] = "+staged content"
	return f
}

// AddUnstagedFile adds a file to the unstaged (modified) state.
func (f *FakeGitRunner) AddUnstagedFile(path string) *FakeGitRunner {
	f.Unstaged[path] = "+modified content"
	return f
}

// AddUntrackedFile adds an untracked file.
func (f *FakeGitRunner) AddUntrackedFile(path string) *FakeGitRunner {
	f.Untracked = append(f.Untracked, path)
	return f
}

// AddConflictFile adds a file with merge conflicts.
func (f *FakeGitRunner) AddConflictFile(path string) *FakeGitRunner {
	f.Conflicts = append(f.Conflicts, path)
	return f
}

// WithBranch sets the branch state.
func (f *FakeGitRunner) WithBranch(head, upstream string, ahead, behind int) *FakeGitRunner {
	f.Branch.Head = head
	f.Branch.Upstream = upstream
	f.Branch.Ahead = ahead
	f.Branch.Behind = behind
	return f
}

// WithNotARepository simulates a non-git directory.
func (f *FakeGitRunner) WithNotARepository() *FakeGitRunner {
	f.IsRepository = false
	return f
}

// WithGitUnavailable simulates git not being installed.
func (f *FakeGitRunner) WithGitUnavailable() *FakeGitRunner {
	f.GitAvailable = false
	return f
}

// WithRepoRoot sets the repository root path.
func (f *FakeGitRunner) WithRepoRoot(root string) *FakeGitRunner {
	f.RepoRoot = root
	return f
}

// AssertCalled verifies a method was called.
func (f *FakeGitRunner) AssertCalled(method string) bool {
	for _, call := range f.Calls {
		if call.Method == method {
			return true
		}
	}
	return false
}

// AssertNotCalled verifies a method was not called.
func (f *FakeGitRunner) AssertNotCalled(method string) bool {
	return !f.AssertCalled(method)
}

// CallCount returns the number of times a method was called.
func (f *FakeGitRunner) CallCount(method string) int {
	count := 0
	for _, call := range f.Calls {
		if call.Method == method {
			count++
		}
	}
	return count
}
