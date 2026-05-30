package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// baselineNamePrefix prefixes every pre-execution baseline so they are easy to
// recognize and bulk-clean. The working-tree-state hash suffix makes the name
// a content-addressed cache key: identical scenario state ⇒ identical name ⇒
// EnsureSnapshot reuses the existing baseline instead of recapturing.
const baselineNamePrefix = "preexec"

// baselineNameFor builds the deterministic baseline name for a scenario at a
// given working-tree-state hash. The hash is truncated to 12 hex chars — ample
// to avoid collisions across the handful of in-flight baselines per branch.
func baselineNameFor(scenario, stateHash string) string {
	short := stateHash
	if len(short) > 12 {
		short = short[:12]
	}
	return fmt.Sprintf("%s-%s-%s", baselineNamePrefix, scenario, short)
}

// gitRepoRoot resolves the enclosing git repository root for an anchor path
// inside it. Used so working-tree hashing runs from the repo top level.
func gitRepoRoot(ctx context.Context, anchor string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", anchor, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve git root from %s: %w", anchor, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// workingTreeStateHash computes a deterministic fingerprint of a scenario's
// working-tree state: the committed subtree object plus every uncommitted edit
// (tracked-modified, staged, or untracked) under scenarios/<scenario>. Two
// distinct edits to the same file produce different hashes because file
// contents — not just the porcelain status lines — feed the digest.
//
// gitRoot must be the repository top level; scenario is the bare scenario name.
func workingTreeStateHash(ctx context.Context, gitRoot, scenario string) (string, error) {
	rel := filepath.Join("scenarios", scenario)

	// Committed subtree object hash. Missing in HEAD (brand-new scenario) is
	// not an error — an empty tree hash still combines into a stable digest.
	treeHash := ""
	if out, err := exec.CommandContext(ctx, "git", "-C", gitRoot, "rev-parse", "HEAD:"+rel).Output(); err == nil {
		treeHash = strings.TrimSpace(string(out))
	}

	// Uncommitted changes scoped to the scenario subtree.
	statusOut, err := exec.CommandContext(ctx, "git", "-C", gitRoot, "status", "--porcelain", "--", rel).Output()
	if err != nil {
		return "", fmt.Errorf("git status for %s: %w", rel, err)
	}
	porcelain := strings.TrimRight(string(statusOut), "\n")

	contents := map[string][]byte{}
	for _, path := range changedPathsFromPorcelain(porcelain) {
		abs := filepath.Join(gitRoot, path)
		if data, readErr := os.ReadFile(abs); readErr == nil {
			contents[path] = data
		}
	}

	return computeStateHash(treeHash, porcelain, contents), nil
}

// changedPathsFromPorcelain extracts file paths from `git status --porcelain`
// output, resolving rename entries ("R  old -> new") to their new path.
func changedPathsFromPorcelain(porcelain string) []string {
	var paths []string
	for _, line := range strings.Split(porcelain, "\n") {
		if len(line) < 4 {
			continue
		}
		// Porcelain format: XY<space>PATH (status code in cols 0-1, path from col 3).
		entry := strings.TrimSpace(line[3:])
		if entry == "" {
			continue
		}
		if idx := strings.Index(entry, " -> "); idx >= 0 {
			entry = entry[idx+len(" -> "):]
		}
		entry = strings.Trim(entry, "\"")
		paths = append(paths, entry)
	}
	return paths
}

// computeStateHash is the pure digest function: it folds the committed subtree
// hash, the porcelain status, and the content of each changed file (in sorted
// path order for determinism) into one sha256 hex string.
func computeStateHash(treeHash, porcelain string, contents map[string][]byte) string {
	h := sha256.New()
	fmt.Fprintf(h, "tree:%s\n", treeHash)
	fmt.Fprintf(h, "status:%s\n", porcelain)

	paths := make([]string, 0, len(contents))
	for p := range contents {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		fileSum := sha256.Sum256(contents[p])
		fmt.Fprintf(h, "file:%s:%s\n", p, hex.EncodeToString(fileSum[:]))
	}
	return hex.EncodeToString(h.Sum(nil))
}
