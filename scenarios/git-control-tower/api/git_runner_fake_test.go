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
	Branch         FakeBranchState
	LocalBranches  map[string]FakeBranchRef
	RemoteBranches map[string]FakeBranchRef
	Staged         map[string]string // path -> content
	Unstaged       map[string]string // path -> content diff
	Untracked      []string
	Conflicts      []string
	IsRepository   bool   // Whether this is a valid git repo
	GitAvailable   bool   // Whether git binary is "installed"
	RepoRoot       string // Configured repository root path
	RemoteURL      string // Configured remote URL (for GetRemoteURL)

	// Error injection for testing error paths
	StatusError          error
	DiffError            error
	StageError           error
	UnstageError         error
	CommitError          error
	RevParseError        error
	LookPathError        error
	FetchError           error
	RemoteURLError       error
	ConfigError          error
	DiscardError         error
	PushError            error
	PullError            error
	LogError             error
	RemoveFromIndexError error
	BranchesError        error
	CreateBranchError    error
	CheckoutBranchError  error
	TrackRemoteError     error
	CheckRefFormatError  error

	// Commit tracking
	LastCommitMessage     string
	LastCommitAuthorName  string
	LastCommitAuthorEmail string
	CommitCount           int

	// History tracking
	HistoryLines   []string
	HistoryDetails []RepoHistoryEntry
	NumstatLines   []string

	// Config values
	ConfigValues map[string]string

	// Fetch tracking
	FetchCount int

	// Push/Pull tracking
	PushCount int
	PullCount int

	// Push behavior controls
	PushUpdatesRemote bool

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

type FakeBranchRef struct {
	Name         string
	Upstream     string
	OID          string
	LastCommitAt string
}

// FakeGitCall records a call made to the fake for test verification.
type FakeGitCall struct {
	Method string
	Args   []string
}

// NewFakeGitRunner creates a FakeGitRunner with sensible defaults.
// By default, it simulates a valid git repository with git available.
func NewFakeGitRunner() *FakeGitRunner {
	now := "2025-01-01 00:00:00 +0000"
	return &FakeGitRunner{
		Branch: FakeBranchState{
			Head: "main",
			OID:  "abc123def456",
		},
		LocalBranches: map[string]FakeBranchRef{
			"main": {
				Name:         "main",
				Upstream:     "origin/main",
				OID:          "abc123def456",
				LastCommitAt: now,
			},
		},
		RemoteBranches: map[string]FakeBranchRef{
			"origin/main": {
				Name:         "origin/main",
				OID:          "abc123def456",
				LastCommitAt: now,
			},
		},
		Staged:       make(map[string]string),
		Unstaged:     make(map[string]string),
		Untracked:    []string{},
		Conflicts:    []string{},
		IsRepository: true,
		GitAvailable: true,
		RepoRoot:     "/fake/repo",
		ConfigValues: map[string]string{},
		Calls:        []FakeGitCall{},
		PushUpdatesRemote: true,
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
func (f *FakeGitRunner) Commit(ctx context.Context, repoDir string, message string, options CommitOptions) (string, error) {
	f.recordCall("Commit", repoDir, message, options.AuthorName, options.AuthorEmail)

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
	f.LastCommitAuthorName = options.AuthorName
	f.LastCommitAuthorEmail = options.AuthorEmail
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
		case "HEAD":
			if f.Branch.OID == "" {
				return nil, fmt.Errorf("fatal: ambiguous argument 'HEAD'")
			}
			return []byte(f.Branch.OID + "\n"), nil
		case "@{u}":
			upstream := strings.TrimSpace(f.Branch.Upstream)
			if upstream == "" {
				return nil, fmt.Errorf("fatal: no upstream configured")
			}
			if ref, ok := f.RemoteBranches[upstream]; ok {
				return []byte(ref.OID + "\n"), nil
			}
			return nil, fmt.Errorf("fatal: unknown upstream ref")
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

		if ref, ok := f.RemoteBranches[arg]; ok {
			return []byte(ref.OID + "\n"), nil
		}
		if strings.HasPrefix(arg, "refs/remotes/") {
			key := strings.TrimPrefix(arg, "refs/remotes/")
			if ref, ok := f.RemoteBranches[key]; ok {
				return []byte(ref.OID + "\n"), nil
			}
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

// ConfigGet returns a configured git config value.
func (f *FakeGitRunner) ConfigGet(ctx context.Context, repoDir string, key string) (string, error) {
	f.recordCall("ConfigGet", repoDir, key)

	if f.ConfigError != nil {
		return "", f.ConfigError
	}

	if value, ok := f.ConfigValues[key]; ok {
		return value, nil
	}
	return "", fmt.Errorf("config not found")
}

// ResolveRepoRoot returns the configured repository root.
func (f *FakeGitRunner) ResolveRepoRoot(_ context.Context) string {
	f.recordCall("ResolveRepoRoot")

	if !f.IsRepository {
		return ""
	}
	return f.RepoRoot
}

// FetchRemote simulates fetching from a remote.
func (f *FakeGitRunner) FetchRemote(ctx context.Context, repoDir string, remote string) error {
	f.recordCall("FetchRemote", repoDir, remote)

	if f.FetchError != nil {
		return f.FetchError
	}

	f.FetchCount++
	return nil
}

// GetRemoteURL returns the configured remote URL.
func (f *FakeGitRunner) GetRemoteURL(ctx context.Context, repoDir string, remote string) (string, error) {
	f.recordCall("GetRemoteURL", repoDir, remote)

	if f.RemoteURLError != nil {
		return "", f.RemoteURLError
	}

	if f.RemoteURL == "" {
		return "", fmt.Errorf("fatal: No such remote '%s'", remote)
	}
	return f.RemoteURL, nil
}

// Discard simulates discarding changes.
func (f *FakeGitRunner) Discard(ctx context.Context, repoDir string, paths []string, untracked bool) error {
	f.recordCall("Discard", append([]string{repoDir, fmt.Sprintf("untracked=%v", untracked)}, paths...)...)

	if f.DiscardError != nil {
		return f.DiscardError
	}

	if untracked {
		// Remove from untracked list
		for _, path := range paths {
			for i, u := range f.Untracked {
				if u == path {
					f.Untracked = append(f.Untracked[:i], f.Untracked[i+1:]...)
					break
				}
			}
		}
	} else {
		// Remove from unstaged (tracked files reverted)
		for _, path := range paths {
			delete(f.Unstaged, path)
		}
	}

	return nil
}

// Push simulates pushing to a remote.
func (f *FakeGitRunner) Push(ctx context.Context, repoDir string, remote string, branch string, setUpstream bool) error {
	f.recordCall("Push", repoDir, remote, branch, fmt.Sprintf("setUpstream=%v", setUpstream))

	if f.PushError != nil {
		return f.PushError
	}

	f.PushCount++

	// Simulate push by updating ahead count
	if f.Branch.Ahead > 0 {
		f.Branch.Ahead = 0
	}

	if f.PushUpdatesRemote {
		if remote == "" {
			remote = "origin"
		}
		if branch != "" && f.Branch.OID != "" {
			key := fmt.Sprintf("%s/%s", remote, branch)
			ref := f.RemoteBranches[key]
			ref.Name = key
			ref.OID = f.Branch.OID
			f.RemoteBranches[key] = ref
		}
	}

	return nil
}

// Pull simulates pulling from a remote.
func (f *FakeGitRunner) Pull(ctx context.Context, repoDir string, remote string, branch string) error {
	f.recordCall("Pull", repoDir, remote, branch)

	if f.PullError != nil {
		return f.PullError
	}

	f.PullCount++

	// Simulate pull by updating behind count
	if f.Branch.Behind > 0 {
		f.Branch.Behind = 0
	}

	return nil
}

func (f *FakeGitRunner) LogGraph(ctx context.Context, repoDir string, limit int) ([]byte, error) {
	f.recordCall("LogGraph", repoDir, fmt.Sprintf("limit=%d", limit))

	if f.LogError != nil {
		return nil, f.LogError
	}

	lines := f.HistoryLines
	if limit > 0 && len(lines) > limit {
		lines = lines[:limit]
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func (f *FakeGitRunner) LogDetails(ctx context.Context, repoDir string, limit int) ([]byte, error) {
	f.recordCall("LogDetails", repoDir, fmt.Sprintf("limit=%d", limit))

	if f.LogError != nil {
		return nil, f.LogError
	}

	entries := f.HistoryDetails
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	var out strings.Builder
	for index, entry := range entries {
		if index > 0 {
			out.WriteString("\n\n")
		}
		out.WriteString(fmt.Sprintf("%s\x00%s\x00%s\x00%s", entry.Hash, entry.Author, entry.Date, entry.Subject))
		for _, file := range entry.Files {
			out.WriteString("\n")
			out.WriteString(file)
		}
	}

	return []byte(out.String()), nil
}

func (f *FakeGitRunner) DiffNumstat(ctx context.Context, repoDir string, staged bool) ([]byte, error) {
	f.recordCall("DiffNumstat", repoDir, fmt.Sprintf("staged=%v", staged))
	if f.DiffError != nil {
		return nil, f.DiffError
	}
	return []byte(strings.Join(f.NumstatLines, "\n")), nil
}

func (f *FakeGitRunner) RemoveFromIndex(ctx context.Context, repoDir string, paths []string) error {
	f.recordCall("RemoveFromIndex", append([]string{repoDir}, paths...)...)

	if f.RemoveFromIndexError != nil {
		return f.RemoveFromIndexError
	}

	for _, path := range paths {
		delete(f.Staged, path)
		delete(f.Unstaged, path)
	}
	return nil
}

// ShowCommitDiff simulates showing a diff for a specific commit.
func (f *FakeGitRunner) ShowCommitDiff(ctx context.Context, repoDir string, commit string, path string) ([]byte, error) {
	f.recordCall("ShowCommitDiff", repoDir, commit, path)

	if f.DiffError != nil {
		return nil, f.DiffError
	}

	// Return a simulated commit diff
	if path != "" {
		return []byte(f.generateDiff(path, "+committed change")), nil
	}

	// If no path, return all staged files as a simulated commit
	var diffs []string
	for p := range f.Staged {
		diffs = append(diffs, f.generateDiff(p, "+committed change"))
	}
	return []byte(strings.Join(diffs, "\n")), nil
}

// ShowFileAtCommit simulates getting file content at a specific commit.
func (f *FakeGitRunner) ShowFileAtCommit(ctx context.Context, repoDir string, commit string, path string) ([]byte, error) {
	f.recordCall("ShowFileAtCommit", repoDir, commit, path)

	if f.DiffError != nil {
		return nil, f.DiffError
	}

	// Return simulated file content
	return []byte("line 1\nline 2\nline 3\nmodified line\nline 5\n"), nil
}

func (f *FakeGitRunner) Branches(ctx context.Context, repoDir string) ([]byte, error) {
	f.recordCall("Branches", repoDir)
	if f.BranchesError != nil {
		return nil, f.BranchesError
	}

	var lines []string
	localNames := make([]string, 0, len(f.LocalBranches))
	for name := range f.LocalBranches {
		localNames = append(localNames, name)
	}
	sort.Strings(localNames)
	for _, name := range localNames {
		ref := f.LocalBranches[name]
		line := fmt.Sprintf("refs/heads/%s|%s|%s|%s|%s", name, name, ref.Upstream, ref.OID, ref.LastCommitAt)
		lines = append(lines, line)
	}

	remoteNames := make([]string, 0, len(f.RemoteBranches))
	for name := range f.RemoteBranches {
		remoteNames = append(remoteNames, name)
	}
	sort.Strings(remoteNames)
	for _, name := range remoteNames {
		ref := f.RemoteBranches[name]
		line := fmt.Sprintf("refs/remotes/%s|%s|%s|%s|%s", name, name, ref.Upstream, ref.OID, ref.LastCommitAt)
		lines = append(lines, line)
	}

	return []byte(strings.Join(lines, "\n")), nil
}

func (f *FakeGitRunner) CreateBranch(ctx context.Context, repoDir string, name string, from string) error {
	f.recordCall("CreateBranch", repoDir, name, from)
	if f.CreateBranchError != nil {
		return f.CreateBranchError
	}
	if _, exists := f.LocalBranches[name]; exists {
		return fmt.Errorf("branch already exists")
	}
	oid := f.Branch.OID
	if strings.TrimSpace(from) != "" {
		oid = "from-" + from
	}
	f.LocalBranches[name] = FakeBranchRef{
		Name:         name,
		Upstream:     "",
		OID:          oid,
		LastCommitAt: "2025-01-01 00:00:00 +0000",
	}
	return nil
}

func (f *FakeGitRunner) CheckoutBranch(ctx context.Context, repoDir string, name string) error {
	f.recordCall("CheckoutBranch", repoDir, name)
	if f.CheckoutBranchError != nil {
		return f.CheckoutBranchError
	}
	ref, exists := f.LocalBranches[name]
	if !exists {
		return fmt.Errorf("branch not found")
	}
	f.Branch.Head = name
	f.Branch.Upstream = ref.Upstream
	return nil
}

func (f *FakeGitRunner) TrackRemoteBranch(ctx context.Context, repoDir string, remote string, name string) error {
	f.recordCall("TrackRemoteBranch", repoDir, remote, name)
	if f.TrackRemoteError != nil {
		return f.TrackRemoteError
	}
	remoteName := fmt.Sprintf("%s/%s", remote, name)
	ref, exists := f.RemoteBranches[remoteName]
	if !exists {
		return fmt.Errorf("remote branch not found")
	}
	f.LocalBranches[name] = FakeBranchRef{
		Name:         name,
		Upstream:     remoteName,
		OID:          ref.OID,
		LastCommitAt: ref.LastCommitAt,
	}
	f.Branch.Head = name
	f.Branch.Upstream = remoteName
	return nil
}

func (f *FakeGitRunner) CheckRefFormat(ctx context.Context, repoDir string, name string) error {
	f.recordCall("CheckRefFormat", repoDir, name)
	if f.CheckRefFormatError != nil {
		return f.CheckRefFormatError
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || strings.Contains(trimmed, " ") {
		return fmt.Errorf("invalid branch name")
	}
	return nil
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
	if _, exists := f.LocalBranches[head]; !exists {
		f.LocalBranches[head] = FakeBranchRef{
			Name:         head,
			Upstream:     upstream,
			OID:          f.Branch.OID,
			LastCommitAt: "2025-01-01 00:00:00 +0000",
		}
	}
	return f
}

func (f *FakeGitRunner) WithLocalBranch(name, upstream string, oid string) *FakeGitRunner {
	if oid == "" {
		oid = "abc123def456"
	}
	f.LocalBranches[name] = FakeBranchRef{
		Name:         name,
		Upstream:     upstream,
		OID:          oid,
		LastCommitAt: "2025-01-01 00:00:00 +0000",
	}
	return f
}

func (f *FakeGitRunner) WithRemoteBranch(name string, oid string) *FakeGitRunner {
	if oid == "" {
		oid = "abc123def456"
	}
	f.RemoteBranches[name] = FakeBranchRef{
		Name:         name,
		OID:          oid,
		LastCommitAt: "2025-01-01 00:00:00 +0000",
	}
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

// WithRemoteURL sets the remote URL.
func (f *FakeGitRunner) WithRemoteURL(url string) *FakeGitRunner {
	f.RemoteURL = url
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
