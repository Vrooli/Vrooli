// Package treedigest computes a deterministic content digest of a scenario's
// working tree. The digest is the freshness identity for test runs: a run
// stamped with digest D proves "these phases executed against exactly this
// byte-state of the scenario". CheckFreshness compares the scenario's CURRENT
// digest against stamped runs, so the comparison must be byte-exact and
// reproducible — no timestamps, no ordering nondeterminism.
//
// Spec (frozen by the requirements-traceability plan, §8 Contract Decisions):
// sha256 over the sorted list of `relpath \x00 sha256(file bytes) \x0a` for
// every file under the scenario directory that is git-tracked OR
// untracked-and-not-ignored (`git ls-files --cached --others
// --exclude-standard`), excluding generated/state-rooted directories
// (coverage/, data/, dist/, node_modules/, …). Working-tree bytes are hashed
// — never the index — because tests run against the working tree.
//
// Documented v1 limitation: the digest scopes to the scenario directory only.
// Edits to shared packages (packages/proto, packages/*-go) do NOT change a
// scenario's digest, so freshness can read "fresh" after a shared-package
// change. Extending scope to declared dependencies is future work.
package treedigest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// excludedDirs are scenario-relative top-level directory names whose contents
// never participate in the digest: generated artifacts and runtime state that
// change as a *consequence* of running tests (hashing them would make every
// run stale-ify itself) or that no test reads as source.
var excludedDirs = map[string]struct{}{
	".git":         {},
	"coverage":     {},
	"data":         {},
	"dist":         {},
	"build":        {},
	"node_modules": {},
	"logs":         {},
	"tmp":          {},
	".cache":       {},
}

// Runner abstracts command execution (the seam for tests).
type Runner func(dir string, name string, args ...string) ([]byte, error)

func defaultRunner(dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %s: %w (%s)", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// Compute returns the digest of the scenario directory's current working
// tree, as a "td:"-prefixed hex string.
func Compute(scenarioDir string) (string, error) {
	return ComputeWithRunner(scenarioDir, defaultRunner)
}

// ComputeWithRunner is Compute with an injectable command runner.
func ComputeWithRunner(scenarioDir string, run Runner) (string, error) {
	scenarioDir = strings.TrimSpace(scenarioDir)
	if scenarioDir == "" {
		return "", fmt.Errorf("scenario directory is required")
	}
	info, err := os.Stat(scenarioDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("scenario directory %q is not a readable directory", scenarioDir)
	}

	files, err := listFiles(scenarioDir, run)
	if err != nil {
		return "", err
	}

	entries := make([]string, 0, len(files))
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(scenarioDir, rel))
		if err != nil {
			// Tracked-but-deleted (or unreadable mid-edit) files simply do
			// not contribute an entry; their absence changes the digest
			// relative to when they existed, which is the correct signal.
			continue
		}
		sum := sha256.Sum256(data)
		entries = append(entries, rel+"\x00"+hex.EncodeToString(sum[:])+"\n")
	}
	sort.Strings(entries)

	h := sha256.New()
	for _, e := range entries {
		h.Write([]byte(e))
	}
	return "td:" + hex.EncodeToString(h.Sum(nil)), nil
}

// listFiles enumerates the digest-relevant files, scenario-relative with
// forward slashes. Inside a git work tree it uses `git ls-files --cached
// --others --exclude-standard` (tracked + untracked-not-ignored, the spec's
// authority); outside one (e.g. --scenario-path into /tmp) it falls back to a
// filesystem walk with the same directory exclusions.
func listFiles(scenarioDir string, run Runner) ([]string, error) {
	if out, err := run(scenarioDir, "git", "ls-files", "--cached", "--others", "--exclude-standard"); err == nil {
		seen := make(map[string]struct{})
		var files []string
		for _, line := range strings.Split(string(out), "\n") {
			rel := strings.TrimSpace(line)
			if rel == "" || isExcluded(rel) {
				continue
			}
			if _, dup := seen[rel]; dup {
				continue
			}
			seen[rel] = struct{}{}
			files = append(files, rel)
		}
		return files, nil
	}

	var files []string
	err := filepath.WalkDir(scenarioDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(scenarioDir, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && isExcluded(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isExcluded(rel) {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk scenario dir: %w", err)
	}
	return files, nil
}

// isExcluded reports whether a scenario-relative path lives under an excluded
// top-level directory.
func isExcluded(rel string) bool {
	top := rel
	if i := strings.IndexByte(rel, '/'); i >= 0 {
		top = rel[:i]
	}
	_, ok := excludedDirs[top]
	return ok
}

// GitContext describes the repository state a run executed against. All
// fields are best-effort: outside a git work tree every field is zero.
type GitContext struct {
	Sha          string
	Branch       string
	Dirty        bool
	DirtySummary string
}

// CollectGitContext gathers the git fields stamped onto run records. Errors
// are deliberately swallowed into a zero value — freshness degrades to
// "unknown" rather than failing a test run over git trouble.
func CollectGitContext(scenarioDir string) GitContext {
	return CollectGitContextWithRunner(scenarioDir, defaultRunner)
}

// CollectGitContextWithRunner is CollectGitContext with an injectable runner.
func CollectGitContextWithRunner(scenarioDir string, run Runner) GitContext {
	var ctx GitContext
	if out, err := run(scenarioDir, "git", "rev-parse", "HEAD"); err == nil {
		ctx.Sha = strings.TrimSpace(string(out))
	}
	if out, err := run(scenarioDir, "git", "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		ctx.Branch = strings.TrimSpace(string(out))
	}
	if out, err := run(scenarioDir, "git", "status", "--porcelain", "--", "."); err == nil {
		lines := 0
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) != "" {
				lines++
			}
		}
		if lines > 0 {
			ctx.Dirty = true
			ctx.DirtySummary = fmt.Sprintf("%d path(s) modified or untracked under the scenario", lines)
		}
	}
	return ctx
}
