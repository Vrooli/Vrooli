package worktree

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRunner is the minimal subprocess seam this package's production
// impls need. It exists so callers in main.go can inject whatever
// already-wrapped git runner the scenario carries (GCT's existing
// ExecGitRunner satisfies this shape by accident through its Run
// method; we don't depend on the type, only the interface).
//
// Note: GitRunner is NEVER exercised in tests of this package. Tests
// substitute FakeInspector / FakeMutator directly.
type GitRunner interface {
	Run(ctx context.Context, repoDir string, args ...string) ([]byte, error)
}

// gitInspector / gitMutator implement Inspector / Mutator against a
// real git binary via GitRunner. Compile-time checks anchor the
// interface satisfaction:
type (
	gitInspector struct{ runner GitRunner }
	gitMutator   struct{ runner GitRunner }
)

var (
	_ Inspector = (*gitInspector)(nil)
	_ Mutator   = (*gitMutator)(nil)
)

// NewGitInspector returns a production Inspector backed by runner.
func NewGitInspector(runner GitRunner) Inspector { return &gitInspector{runner: runner} }

// NewGitMutator returns a production Mutator backed by runner.
func NewGitMutator(runner GitRunner) Mutator { return &gitMutator{runner: runner} }

// ExecRunner is a tiny default GitRunner that shells out via os/exec.
// Provided for callers that don't already have a runner. Not used in
// tests of this package.
type ExecRunner struct {
	GitPath string
}

// Run executes git with -C repoDir and returns combined output.
func (e ExecRunner) Run(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	gitPath := e.GitPath
	if gitPath == "" {
		gitPath = "git"
	}
	full := append([]string{"-C", repoDir}, args...)
	cmd := exec.CommandContext(ctx, gitPath, full...)
	return cmd.CombinedOutput()
}

// --- Inspector ---

func (g *gitInspector) List(ctx context.Context, repoPath string) ([]Worktree, error) {
	out, err := g.runner.Run(ctx, repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("worktree list: %w: %s", err, string(out))
	}
	return parsePorcelain(string(out)), nil
}

func (g *gitInspector) IdentifyPath(ctx context.Context, repoPath string) (Identity, error) {
	id := Identity{}
	commonDirOut, err := g.runner.Run(ctx, repoPath, "rev-parse", "--git-common-dir")
	if err != nil {
		return id, fmt.Errorf("rev-parse --git-common-dir: %w: %s", err, string(commonDirOut))
	}
	gitDirOut, err := g.runner.Run(ctx, repoPath, "rev-parse", "--git-dir")
	if err != nil {
		return id, fmt.Errorf("rev-parse --git-dir: %w: %s", err, string(gitDirOut))
	}
	topOut, err := g.runner.Run(ctx, repoPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return id, fmt.Errorf("rev-parse --show-toplevel: %w: %s", err, string(topOut))
	}
	headOut, _ := g.runner.Run(ctx, repoPath, "rev-parse", "HEAD")
	branchOut, _ := g.runner.Run(ctx, repoPath, "symbolic-ref", "--quiet", "--short", "HEAD")

	commonDir := strings.TrimSpace(string(commonDirOut))
	gitDir := strings.TrimSpace(string(gitDirOut))
	worktreeRoot := strings.TrimSpace(string(topOut))

	// commonDir != gitDir indicates a linked worktree.
	id.IsLinkedWorktree = filepath.Clean(commonDir) != filepath.Clean(gitDir)
	id.CommonRepoRoot = filepath.Dir(filepath.Clean(commonDir))
	if id.CommonRepoRoot == "." {
		// Relative paths are awkward; fall back to the worktree root for the main worktree.
		id.CommonRepoRoot = worktreeRoot
	}
	id.WorktreeName = filepath.Base(worktreeRoot)
	id.WorktreeHead = strings.TrimSpace(string(headOut))
	id.Branch = strings.TrimSpace(string(branchOut))
	id.Detached = id.Branch == ""

	wts, err := g.List(ctx, repoPath)
	if err == nil {
		id.LinkedWorktreeCount = len(wts)
	}
	return id, nil
}

func (g *gitInspector) ClaimedBranches(ctx context.Context, repoPath string) (map[string]string, error) {
	wts, err := g.List(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, w := range wts {
		if w.IsMain || w.Branch == "" {
			continue
		}
		out[w.Branch] = w.Path
	}
	return out, nil
}

// --- Mutator ---

func (g *gitMutator) Add(ctx context.Context, in CreateInput) (Worktree, error) {
	args := []string{"worktree", "add"}
	if in.Force {
		args = append(args, "--force")
	}
	switch in.Mode() {
	case CreateModeExistingBranch:
		args = append(args, in.NewWorktreePath, in.ExistingBranch)
	case CreateModeNewBranch:
		args = append(args, "-b", in.NewBranchName)
		if in.Track {
			args = append(args, "--track")
		}
		args = append(args, in.NewWorktreePath)
		if in.NewBranchStart != "" {
			args = append(args, in.NewBranchStart)
		}
	case CreateModeDetachedCommit:
		args = append(args, "--detach", in.NewWorktreePath, in.Commit)
	default:
		return Worktree{}, fmt.Errorf("%w: unspecified source", ErrInvalid)
	}
	if out, err := g.runner.Run(ctx, in.RepoPath, args...); err != nil {
		return Worktree{}, classifyAddErr(err, string(out))
	}
	// Re-read so the caller sees current metadata.
	wts, err := (&gitInspector{runner: g.runner}).List(ctx, in.RepoPath)
	if err != nil {
		return Worktree{}, err
	}
	for _, w := range wts {
		if filepath.Clean(w.Path) == filepath.Clean(in.NewWorktreePath) {
			return w, nil
		}
	}
	return Worktree{Path: in.NewWorktreePath, Name: filepath.Base(in.NewWorktreePath)}, nil
}

func (g *gitMutator) Remove(ctx context.Context, repoPath, worktreePath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)
	if out, err := g.runner.Run(ctx, repoPath, args...); err != nil {
		return classifyRemoveErr(err, string(out))
	}
	return nil
}

func (g *gitMutator) Lock(ctx context.Context, in LockInput) (Worktree, error) {
	args := []string{"worktree", "lock"}
	if in.Reason != "" {
		args = append(args, "--reason", in.Reason)
	}
	args = append(args, in.WorktreePath)
	if out, err := g.runner.Run(ctx, in.RepoPath, args...); err != nil {
		return Worktree{}, fmt.Errorf("worktree lock: %w: %s", err, string(out))
	}
	return findOne(ctx, g.runner, in.RepoPath, in.WorktreePath)
}

func (g *gitMutator) Unlock(ctx context.Context, repoPath, worktreePath string) (Worktree, error) {
	if out, err := g.runner.Run(ctx, repoPath, "worktree", "unlock", worktreePath); err != nil {
		return Worktree{}, fmt.Errorf("worktree unlock: %w: %s", err, string(out))
	}
	return findOne(ctx, g.runner, repoPath, worktreePath)
}

func (g *gitMutator) Move(ctx context.Context, in MoveInput) (Worktree, error) {
	if out, err := g.runner.Run(ctx, in.RepoPath, "worktree", "move", in.WorktreePath, in.NewWorktreePath); err != nil {
		return Worktree{}, fmt.Errorf("worktree move: %w: %s", err, string(out))
	}
	return findOne(ctx, g.runner, in.RepoPath, in.NewWorktreePath)
}

func (g *gitMutator) Prune(ctx context.Context, in PruneInput) (PruneResult, error) {
	args := []string{"worktree", "prune", "--verbose"}
	if in.ReportOnly {
		args = append(args, "--dry-run")
	}
	if in.Reason != "" {
		args = append(args, "--expire=now")
	}
	out, err := g.runner.Run(ctx, in.RepoPath, args...)
	if err != nil {
		return PruneResult{}, fmt.Errorf("worktree prune: %w: %s", err, string(out))
	}
	var pruned []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// git emits "Removing worktrees/<name>: <reason>" for each.
		if strings.HasPrefix(line, "Removing worktrees/") {
			rest := strings.TrimPrefix(line, "Removing worktrees/")
			if i := strings.Index(rest, ":"); i >= 0 {
				rest = rest[:i]
			}
			pruned = append(pruned, rest)
		}
	}
	return PruneResult{PrunedPaths: pruned}, nil
}

func findOne(ctx context.Context, runner GitRunner, repoPath, worktreePath string) (Worktree, error) {
	wts, err := (&gitInspector{runner: runner}).List(ctx, repoPath)
	if err != nil {
		return Worktree{}, err
	}
	for _, w := range wts {
		if filepath.Clean(w.Path) == filepath.Clean(worktreePath) {
			return w, nil
		}
	}
	return Worktree{}, fmt.Errorf("%w: %s", ErrNotFound, worktreePath)
}

func classifyAddErr(err error, output string) error {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "already exists"):
		return fmt.Errorf("%w: %s", ErrInvalid, output)
	case strings.Contains(lower, "is already checked out"):
		return fmt.Errorf("%w: %s", ErrInvalid, output)
	}
	return fmt.Errorf("worktree add: %w: %s", err, output)
}

func classifyRemoveErr(err error, output string) error {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "is locked"):
		return fmt.Errorf("%w: %s", ErrLocked, output)
	case strings.Contains(lower, "contains modified") || strings.Contains(lower, "untracked"):
		return fmt.Errorf("%w: %s", ErrDirty, output)
	case strings.Contains(lower, "not a working tree") || strings.Contains(lower, "not a worktree"):
		return fmt.Errorf("%w: %s", ErrNotFound, output)
	}
	return fmt.Errorf("worktree remove: %w: %s", err, output)
}

// parsePorcelain parses `git worktree list --porcelain` output.
//
// Each block has lines like:
//
//	worktree /abs/path
//	HEAD <sha>
//	branch refs/heads/<name>      (or 'detached')
//	locked [<reason>]              (optional)
//	prunable <reason>              (optional)
//
// Blocks are separated by blank lines. parsePorcelain is exported via
// unexported name; tests in this package may exercise it directly
// without invoking real git.
func parsePorcelain(s string) []Worktree {
	var out []Worktree
	var cur *Worktree
	flush := func() {
		if cur == nil {
			return
		}
		if cur.Branch != "" {
			cur.Name = filepath.Base(cur.Path)
		}
		if cur.Name == "" {
			cur.Name = filepath.Base(cur.Path)
		}
		out = append(out, *cur)
		cur = nil
	}
	sc := bufio.NewScanner(strings.NewReader(s))
	first := true
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		key := parts[0]
		val := ""
		if len(parts) == 2 {
			val = parts[1]
		}
		switch key {
		case "worktree":
			flush()
			cur = &Worktree{Path: val, IsMain: first}
			first = false
		case "HEAD":
			if cur != nil {
				cur.HeadCommit = val
			}
		case "branch":
			if cur != nil {
				cur.Branch = strings.TrimPrefix(val, "refs/heads/")
				cur.Detached = false
			}
		case "detached":
			if cur != nil {
				cur.Detached = true
				cur.Branch = ""
			}
		case "locked":
			if cur != nil {
				cur.Locked = true
				cur.LockReason = val
			}
		case "prunable":
			if cur != nil {
				cur.Prunable = true
				cur.PrunableReason = val
			}
		}
	}
	flush()
	if errors.Is(sc.Err(), bufio.ErrTooLong) {
		return out
	}
	return out
}
