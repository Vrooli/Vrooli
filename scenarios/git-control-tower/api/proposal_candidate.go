package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git-control-tower/internal/proposals"
)

func (s *Server) proposalCandidateCheck(repoPath string, skip bool) proposals.CandidateCheck {
	if s.precommit == nil || skip {
		return nil
	}
	return func(ctx context.Context, candidate string) (*proposals.PrecommitOutcome, error) {
		cfg, err := s.precommit.Get(ctx, repoPath)
		if err != nil {
			return nil, err
		}
		if !cfg.Enabled || !cfg.RunBeforeCommit {
			return nil, nil
		}
		rel, err := filepath.Rel(repoPath, cfg.WorkingDirectory)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("proposal checks must use a directory inside the repository")
		}
		if strings.Contains(cfg.Command, repoPath) {
			return nil, fmt.Errorf("proposal checks must use portable paths, not the original repository's absolute path")
		}
		cfg.WorkingDirectory = filepath.Join(candidate, rel)
		result, err := s.precommit.runConfig(withCandidateEnvironment(ctx, repoPath, candidate), repoPath, cfg)
		return precommitOutcome(&result), err
	}
}

// PrepareCommit builds and checks a private candidate. Only Publish may write
// the real repository, and it advances an explicit ref with compare-and-swap
// while holding Git's HEAD/index locks. The operator's files are never checked
// out or reset, and hooks cannot rewrite a published candidate.
func (g *proposalGit) PrepareCommit(ctx context.Context, p proposals.Proposal, blob func(context.Context, string) ([]byte, error), check proposals.CandidateCheck, skipHooks bool) (proposals.CommitOutcome, error) {
	head, err := g.Head(ctx)
	if err != nil {
		return proposals.CommitOutcome{}, err
	}
	branch, err := g.Branch(ctx)
	if err != nil {
		return proposals.CommitOutcome{}, err
	}
	if branch == "" || branch != p.Branch {
		return proposals.CommitOutcome{}, fmt.Errorf("review a proposal on the current named branch before approval")
	}
	dir, err := os.MkdirTemp("", "gct-commit-candidate-")
	if err != nil {
		return proposals.CommitOutcome{}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	completed := false
	defer func() {
		if !completed {
			cleanup()
		}
	}()
	// No hooks from templates or configured global paths are used accidentally.
	run := func(input []byte, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, g.gitPath, args...)
		cmd.Stdin = bytes.NewReader(input)
		cmd.Env = candidateEnvironment(g.repoDir, dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("candidate %s: %w (%s)", args[0], err, capOutput(string(out)))
		}
		return out, nil
	}
	if _, err = run(nil, "clone", "--template=", "--shared", "--no-checkout", "--no-hardlinks", "--", g.repoDir, dir); err != nil {
		return proposals.CommitOutcome{}, err
	}
	candidate := &proposalGit{gitPath: g.gitPath, repoDir: dir}
	git := func(input []byte, args ...string) ([]byte, error) {
		return run(input, append([]string{"-C", dir}, args...)...)
	}
	ident, err := g.read(ctx, nil, "var", "GIT_AUTHOR_IDENT")
	if err != nil {
		return proposals.CommitOutcome{}, err
	}
	name, rest, ok := strings.Cut(string(ident), " <")
	email, _, emailOK := strings.Cut(rest, ">")
	if !ok || !emailOK {
		return proposals.CommitOutcome{}, fmt.Errorf("repository author identity is unavailable")
	}
	for key, value := range map[string]string{"user.name": name, "user.email": email, "core.hooksPath": filepath.Join(dir, ".git", "hooks")} {
		if _, err = git(nil, "config", "--local", key, value); err != nil {
			return proposals.CommitOutcome{}, err
		}
	}
	if head != "" {
		if _, err = git(nil, "checkout", "--detach", head); err != nil {
			return proposals.CommitOutcome{}, err
		}
	}
	for _, f := range p.Files {
		if f.Deleted {
			if _, err = git(nil, "update-index", "--force-remove", "--", f.Path); err != nil {
				return proposals.CommitOutcome{}, err
			}
			if err = os.Remove(filepath.Join(dir, filepath.FromSlash(f.Path))); err != nil && !os.IsNotExist(err) {
				return proposals.CommitOutcome{}, err
			}
			continue
		}
		data, err := blob(ctx, f.BlobID)
		if err != nil {
			return proposals.CommitOutcome{}, err
		}
		id, err := git(data, "hash-object", "-w", "--no-filters", "--stdin")
		if err != nil {
			return proposals.CommitOutcome{}, err
		}
		if strings.TrimSpace(string(id)) != f.BlobID {
			return proposals.CommitOutcome{}, fmt.Errorf("retained blob identity mismatch for %s", f.Path)
		}
		if _, err = git(nil, "update-index", "--add", "--cacheinfo", f.Mode, f.BlobID, f.Path); err != nil {
			return proposals.CommitOutcome{}, err
		}
	}
	if _, err = git(nil, "checkout-index", "--all", "--force"); err != nil {
		return proposals.CommitOutcome{}, err
	}
	var precommit *proposals.PrecommitOutcome
	if check != nil {
		precommit, err = check(ctx, dir)
		if err != nil {
			return proposals.CommitOutcome{}, err
		}
		if precommit != nil && precommit.Status != "passed" {
			return proposals.CommitOutcome{Failure: "precommit failed", Precommit: precommit}, nil
		}
	}
	if !skipHooks {
		hooks, _, err := resolveHooksDir(ctx, g.repoDir)
		if err != nil {
			return proposals.CommitOutcome{}, err
		}
		for _, hook := range []string{"pre-commit", "prepare-commit-msg", "commit-msg"} {
			path := filepath.Join(hooks, hook)
			info, err := os.Stat(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return proposals.CommitOutcome{}, err
			}
			if !info.Mode().IsRegular() || info.Size() > 256<<10 {
				return proposals.CommitOutcome{}, fmt.Errorf("hook %s exceeds the portable candidate hook contract", hook)
			}
			if info.Mode()&0111 == 0 {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return proposals.CommitOutcome{}, err
			}
			if hasGCTSentinel(data) {
				continue
			} // configured checks already ran once
			if bytes.Contains(data, []byte(g.repoDir)) {
				return proposals.CommitOutcome{}, fmt.Errorf("hook %s references the original repository; use portable paths for candidate checks", hook)
			}
			if err = os.WriteFile(filepath.Join(dir, ".git", "hooks", hook), data, 0700); err != nil {
				return proposals.CommitOutcome{}, err
			}
		}
	}
	args := []string{"commit", "--cleanup=verbatim", "-m", p.Message.Rendered}
	if skipHooks {
		args = append(args, "--no-verify")
	}
	if _, err = git(nil, args...); err != nil {
		return proposals.CommitOutcome{Failure: err.Error(), Precommit: precommit}, nil
	}
	oid, err := candidate.Head(ctx)
	if err != nil {
		return proposals.CommitOutcome{}, err
	}
	facts, err := candidate.CommitFacts(ctx, oid)
	if err != nil {
		return proposals.CommitOutcome{}, err
	}
	// Unstaged hook edits are also refusals: a passing check must not silently
	// leave a different work product than the captured candidate.
	dirty, err := candidate.read(ctx, nil, "diff", "--name-only")
	if err != nil {
		return proposals.CommitOutcome{}, err
	}
	if len(bytes.TrimSpace(dirty)) != 0 {
		return proposals.CommitOutcome{Failure: "checks changed candidate files; review a new proposal revision", Precommit: precommit}, nil
	}
	completed = true
	return proposals.CommitOutcome{OID: oid, Facts: &facts, Precommit: precommit, Cleanup: cleanup, Publish: func(publishCtx context.Context) (string, error) {
		return g.publishCandidate(publishCtx, dir, oid, head, branch, p)
	}}, nil
}

func (g *proposalGit) publishCandidate(ctx context.Context, candidate, oid, head, branch string, p proposals.Proposal) (string, error) {
	var locks []string
	defer func() {
		for _, path := range locks {
			_ = os.Remove(path)
		}
	}()
	for _, name := range []string{"index"} {
		out, err := g.read(ctx, nil, "rev-parse", "--git-path", name)
		if err != nil {
			return "", err
		}
		path := strings.TrimSpace(string(out))
		if !filepath.IsAbs(path) {
			path = filepath.Join(g.repoDir, path)
		}
		lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return "", fmt.Errorf("repository is busy: %w", err)
		}
		_ = lock.Close()
		locks = append(locks, path+".lock")
	}
	currentHead, err := g.Head(ctx)
	if err != nil {
		return "", err
	}
	currentBranch, err := g.Branch(ctx)
	if err != nil {
		return "", err
	}
	if currentHead != head || currentBranch != branch {
		return "", fmt.Errorf("repository destination changed before publication")
	}
	staged, err := g.StagedPaths(ctx)
	if err != nil {
		return "", err
	}
	allowed := map[string]proposals.File{}
	for _, file := range p.Files {
		allowed[file.Path] = file
	}
	for _, path := range staged {
		if _, ok := allowed[path]; !ok {
			return "", fmt.Errorf("foreign staged path before publication: %s", path)
		}
	}
	index, err := g.IndexEntries(ctx, p.Paths())
	if err != nil {
		return "", err
	}
	for _, file := range p.Files {
		entry, exists := index[file.Path]
		if file.Deleted {
			if exists {
				return "", fmt.Errorf("index changed before publication: %s", file.Path)
			}
		} else if !exists || entry.BlobID != file.BlobID || entry.Mode != file.Mode {
			return "", fmt.Errorf("index changed before publication: %s", file.Path)
		}
		if !g.keepLaterChanges {
			contents, err := g.Hash(ctx, []string{file.Path})
			if err != nil {
				return "", err
			}
			content := contents[file.Path]
			if content.SHA256 != file.SHA256 || content.Mode != file.Mode || content.Deleted != file.Deleted {
				return "", fmt.Errorf("file changed before publication: %s", file.Path)
			}
		}

	}
	if err := g.write(ctx, nil, "fetch", "--no-write-fetch-head", "--no-tags", "--no-recurse-submodules", "--no-auto-maintenance", candidate, oid); err != nil {
		return "", err
	}
	expected := head
	if expected == "" {
		expected = strings.Repeat("0", len(oid))
	}
	// Git owns the branch/HEAD locks through a prepared ref transaction. Manual
	// HEAD.lock files prevent update-ref from taking its required reflog lock.
	cmd := exec.CommandContext(ctx, g.gitPath, "-C", g.repoDir, "update-ref", "-m", "commit: "+p.Message.Subject, "--stdin")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		return "", err
	}
	finished := false
	defer func() {
		stdin.Close()
		if !finished {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	reader := bufio.NewReader(stdout)
	step := func(command, expected string) error {
		if _, e := fmt.Fprintln(stdin, command); e != nil {
			return e
		}
		line, e := reader.ReadString('\n')
		if e != nil || strings.TrimSpace(line) != expected {
			return fmt.Errorf("ref transaction %s failed: %v", strings.Fields(command)[0], e)
		}
		return nil
	}
	if err = step("start", "start: ok"); err != nil {
		return "", err
	}
	if _, err = fmt.Fprintf(stdin, "update refs/heads/%s %s %s\n", branch, oid, expected); err != nil {
		return "", err
	}
	if err = step("prepare", "prepare: ok"); err != nil {
		return "", err
	}
	// The ref transaction now holds the destination locks. Recheck the symbolic
	// destination after acquisition rather than trusting the earlier read.
	lockedHead, err := g.Head(ctx)
	if err != nil {
		return "", err
	}
	lockedBranch, err := g.Branch(ctx)
	if err != nil {
		return "", err
	}
	if lockedHead != head || lockedBranch != branch {
		return "", fmt.Errorf("repository destination changed while acquiring publication locks")
	}
	if err = step("commit", "commit: ok"); err != nil {
		// The ref may already have advanced even when its acknowledgment is
		// lost. Finish the subprocess and inspect under a bounded recovery
		// context before allowing the caller to restore the old index.
		stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		finished = true
		recovery, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		ref, readErr := g.read(recovery, nil, "rev-parse", "--verify", "refs/heads/"+branch)
		actual := strings.TrimSpace(string(ref))
		if readErr == nil && actual == oid {
			return oid, nil
		}
		if readErr == nil && actual == head {
			return "", err
		}
		return oid, fmt.Errorf("publication acknowledgment unavailable; inspect candidate %s and destination %s before retrying: %w", oid, branch, errors.Join(err, readErr))
	}
	stdin.Close()
	err = cmd.Wait()
	finished = true
	if err != nil {
		return oid, fmt.Errorf("commit published but ref transaction completion failed: %w", err)
	}

	return oid, nil
}
