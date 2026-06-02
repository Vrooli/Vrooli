// Package gameguard is the anti-gaming classifier for the ecosystem-manager
// closed-loop controller. It inspects the code-level diff an agent run produced
// and detects the gaming-shaped iterations the controller must refuse to
// reward: loosened/deleted [REQ:]-tagged tests, deletion of known-issue ledgers
// (PROBLEMS.md / PROGRESS.md), and added suppression directives or rule-scoping.
//
// The classifier is pure and deterministic: same diff in → same verdict out, no
// I/O, no clocks, no randomness. The orchestrator wires a RunDiffProvider to
// fetch the diff; this package only judges it. Clear cases are penalized (credit
// zeroed); ambiguous cases are flagged for human/GCT review with no auto-penalty
// (decision #5).
package gameguard

import (
	"path"
	"sort"
	"strings"
)

// Cause is a specific gaming pattern the classifier recognizes.
type Cause string

const (
	// CauseTestWeakening: assertions removed from a [REQ:]-tagged test file.
	CauseTestWeakening Cause = "test-weakening"
	// CauseLedgerDeletion: a known-issue ledger (PROBLEMS.md, PROGRESS.md, …)
	// was deleted rather than migrated.
	CauseLedgerDeletion Cause = "ledger-deletion"
	// CauseSuppression: a linter/type-check suppression directive or a rule
	// disable was added.
	CauseSuppression Cause = "suppression"
)

// FileChange is the per-file record from the run diff. ChangeType mirrors
// agent-manager's FileDiff.ChangeType ("added"/"modified"/"deleted"/"renamed").
type FileChange struct {
	Path       string
	ChangeType string
	Additions  int
	Deletions  int
}

// Diff is the input to Classify: the unified diff text plus the per-file change
// records. Content is authoritative for line-level detectors (test-weakening,
// suppression); Files backstops change-type detection (ledger-deletion) when the
// unified content lacks explicit delete markers.
type Diff struct {
	Content string
	Files   []FileChange
}

// Result is the classifier verdict.
type Result struct {
	// Gamed is true when at least one hard detector fired; the controller zeros
	// the iteration's credit when this is set.
	Gamed bool
	// Causes are the distinct gaming patterns detected (sorted, deduped).
	Causes []Cause
	// FlaggedForReview marks an ambiguous iteration (e.g. test assertions removed
	// from a file with no [REQ:] tag) — surfaced for review, not auto-penalized.
	FlaggedForReview bool
	// Details are human-readable notes (one per detection) for the decision trace.
	Details []string
}

// CauseString renders the detected causes as a single trace-friendly token,
// e.g. "gamed:test-weakening,suppression". Empty when not gamed.
func (r Result) CauseString() string {
	if len(r.Causes) == 0 {
		return ""
	}
	parts := make([]string, len(r.Causes))
	for i, c := range r.Causes {
		parts[i] = string(c)
	}
	return "gamed:" + strings.Join(parts, ",")
}

// fileDiff is the parsed per-file view of the unified diff.
type fileDiff struct {
	path    string
	deleted bool
	added   []string
	removed []string
	context []string
}

// Classify runs every detector over the diff and returns the combined verdict.
func Classify(d Diff) Result {
	files := parseUnifiedDiff(d.Content)

	causeSet := map[Cause]bool{}
	var details []string
	flagged := false

	// --- ledger-deletion: a known-issue ledger was deleted. ---
	for _, f := range files {
		if f.deleted && isLedger(f.path) {
			causeSet[CauseLedgerDeletion] = true
			details = append(details, "deleted known-issue ledger "+f.path)
		}
	}
	for _, fc := range d.Files {
		if strings.EqualFold(fc.ChangeType, "deleted") && isLedger(fc.Path) {
			if !ledgerAlreadyNoted(details, fc.Path) {
				causeSet[CauseLedgerDeletion] = true
				details = append(details, "deleted known-issue ledger "+fc.Path)
			}
		}
	}

	// --- test-weakening & suppression: line-level, per file. ---
	for _, f := range files {
		// suppression: an added suppression directive anywhere.
		for _, line := range f.added {
			if tok := suppressionToken(line, f.path); tok != "" {
				causeSet[CauseSuppression] = true
				details = append(details, "added suppression ("+tok+") in "+f.path)
				break
			}
		}

		if !isTestFile(f.path) {
			continue
		}
		removedAssertions := countAssertions(f.removed)
		addedAssertions := countAssertions(f.added)
		if removedAssertions <= addedAssertions {
			continue // net assertions did not drop — not a weakening
		}
		if fileMentionsRequirement(f) {
			causeSet[CauseTestWeakening] = true
			details = append(details, "removed assertion(s) from [REQ:]-tagged test "+f.path)
		} else {
			// Assertions dropped but no requirement tag — suspicious, not certain.
			flagged = true
			details = append(details, "assertions removed from test "+f.path+" (no [REQ:] tag — flagged for review)")
		}
	}

	causes := make([]Cause, 0, len(causeSet))
	for c := range causeSet {
		causes = append(causes, c)
	}
	sort.Slice(causes, func(i, j int) bool { return causes[i] < causes[j] })

	return Result{
		Gamed:            len(causes) > 0,
		Causes:           causes,
		FlaggedForReview: flagged,
		Details:          details,
	}
}

// parseUnifiedDiff splits a git unified diff into per-file views. It is tolerant
// of missing hunk headers; any +/- line is attributed to the current file. A
// `+++ /dev/null` new-side or a `deleted file mode` header marks a deletion.
func parseUnifiedDiff(content string) []*fileDiff {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	var files []*fileDiff
	var cur *fileDiff
	ensure := func() *fileDiff {
		if cur == nil {
			cur = &fileDiff{}
			files = append(files, cur)
		}
		return cur
	}
	for _, line := range strings.Split(content, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			cur = &fileDiff{path: pathFromGitHeader(line)}
			files = append(files, cur)
		case strings.HasPrefix(line, "deleted file mode"):
			ensure().deleted = true
		case line == "+++ /dev/null":
			ensure().deleted = true
		case strings.HasPrefix(line, "+++ "):
			if p := pathFromMarker(line); p != "" && p != "/dev/null" {
				ensure().path = p
			}
		case strings.HasPrefix(line, "--- "):
			// old-side header; ignore for content
		case strings.HasPrefix(line, "@@"):
			// hunk header; ignore
		case strings.HasPrefix(line, "+"):
			c := ensure()
			c.added = append(c.added, strings.TrimPrefix(line, "+"))
		case strings.HasPrefix(line, "-"):
			c := ensure()
			c.removed = append(c.removed, strings.TrimPrefix(line, "-"))
		default:
			ensure().context = append(ensure().context, line)
		}
	}
	return files
}

func pathFromGitHeader(line string) string {
	// "diff --git a/foo b/foo"
	fields := strings.Fields(strings.TrimPrefix(line, "diff --git "))
	if len(fields) >= 2 {
		return strings.TrimPrefix(fields[len(fields)-1], "b/")
	}
	return ""
}

func pathFromMarker(line string) string {
	// "+++ b/foo" or "+++ foo"
	p := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
	p = strings.TrimPrefix(p, "b/")
	return p
}

func ledgerAlreadyNoted(details []string, p string) bool {
	for _, d := range details {
		if strings.Contains(d, p) {
			return true
		}
	}
	return false
}

// isLedger reports whether a path is a known-issue ledger that must be migrated,
// not deleted.
func isLedger(p string) bool {
	base := strings.ToUpper(path.Base(p))
	switch base {
	case "PROBLEMS.MD", "PROGRESS.MD", "KNOWN_ISSUES.MD", "KNOWN-ISSUES.MD", "DEFERRED.MD":
		return true
	}
	return false
}

// isTestFile reports whether a path is a test file across the stacks EM targets.
func isTestFile(p string) bool {
	b := strings.ToLower(path.Base(p))
	switch {
	case strings.HasSuffix(b, "_test.go"):
		return true
	case strings.HasSuffix(b, ".test.ts"), strings.HasSuffix(b, ".test.tsx"),
		strings.HasSuffix(b, ".test.js"), strings.HasSuffix(b, ".test.jsx"):
		return true
	case strings.HasSuffix(b, ".spec.ts"), strings.HasSuffix(b, ".spec.tsx"),
		strings.HasSuffix(b, ".spec.js"), strings.HasSuffix(b, ".spec.jsx"):
		return true
	case strings.HasSuffix(b, "_test.py"), strings.HasPrefix(b, "test_"):
		return true
	}
	lp := strings.ToLower(p)
	return strings.Contains(lp, "/__tests__/") || strings.Contains(lp, "/tests/")
}

// assertionTokens are the substrings that mark a test assertion across stacks.
var assertionTokens = []string{
	"t.error", "t.fatal", "t.fail", "require.", "assert.", "assert(",
	"expect(", ".should", ".toequal", ".tobe", ".tohave", ".tothrow",
	"assertequal", "asserttrue", "assertfalse",
}

func countAssertions(lines []string) int {
	n := 0
	for _, line := range lines {
		l := strings.ToLower(line)
		for _, tok := range assertionTokens {
			if strings.Contains(l, tok) {
				n++
				break
			}
		}
	}
	return n
}

// fileMentionsRequirement reports whether the file's diff (removed, added, or
// surrounding context) carries a [REQ:...] requirement tag — i.e. the weakened
// test validated a tracked requirement.
func fileMentionsRequirement(f *fileDiff) bool {
	for _, set := range [][]string{f.removed, f.added, f.context} {
		for _, line := range set {
			if strings.Contains(line, "[REQ:") {
				return true
			}
		}
	}
	return false
}

// suppressionToken returns the matched suppression directive for an added line,
// or "" if none. Config-file rule disables (enabled:false) are treated as
// suppression only inside config files to avoid tripping on ordinary code.
func suppressionToken(line, filePath string) string {
	l := strings.ToLower(line)
	for _, tok := range []string{
		"nolint", "+build ignore", "go:build ignore", "eslint-disable",
		"@ts-ignore", "@ts-nocheck", "# type: ignore", "# noqa",
		"# pylint: disable", "istanbul ignore", "stylelint-disable",
	} {
		if strings.Contains(l, tok) {
			return strings.TrimSpace(tok)
		}
	}
	if isConfigFile(filePath) {
		compact := strings.ReplaceAll(l, " ", "")
		if strings.Contains(compact, "enabled:false") || strings.Contains(compact, "\"enabled\":false") ||
			strings.Contains(compact, "disabled:true") || strings.Contains(compact, "\"disabled\":true") {
			return "rule-disable"
		}
	}
	return ""
}

func isConfigFile(p string) bool {
	b := strings.ToLower(path.Base(p))
	switch {
	case strings.HasSuffix(b, ".json"), strings.HasSuffix(b, ".yaml"), strings.HasSuffix(b, ".yml"):
		return true
	}
	return false
}
