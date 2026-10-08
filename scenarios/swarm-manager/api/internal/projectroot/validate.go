package projectroot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrAcceptanceMismatch is returned by ValidateAcceptance when one or more
// globs reference paths that do not exist under projectRoot and are not
// declared by `creates`.
var ErrAcceptanceMismatch = errors.New("acceptance_allow paths do not exist under project root")

// GlobProblem describes a single failed acceptance-glob check.
type GlobProblem struct {
	// Glob is the original glob pattern from acceptance_allow.
	Glob string `json:"glob"`
	// ResolvedRel is the literal-prefix path the glob expanded to, relative
	// to the project root. Empty if the glob escaped the project root.
	ResolvedRel string `json:"resolved"`
	// Reason is a short human-readable explanation ("path does not exist",
	// "resolves outside project root", "stat: <err>").
	Reason string `json:"reason"`
	// Existed is true if the resolved path exists on disk; false otherwise.
	Existed bool `json:"existed"`
	// AllowedByCreates is true if the glob's prefix would have failed the
	// existence check but is covered by a creates entry. Such globs are
	// not problems and are not included in AcceptanceReport.Problems.
	AllowedByCreates bool `json:"allowed_by_creates"`
}

// AcceptanceReport is the structured result of ValidateAcceptance.
// Problems is empty when the report is clean.
type AcceptanceReport struct {
	Problems []GlobProblem `json:"problems"`
}

// Clean reports whether the report has no problems.
func (r *AcceptanceReport) Clean() bool {
	return r == nil || len(r.Problems) == 0
}

// ValidateAcceptance asserts that the directory or file implied by each
// glob's literal prefix either exists under projectRoot or is covered by a
// glob in creates (paths the work plans to create).
//
// For each glob the literal prefix is everything up to the first wildcard
// character (*, ?, [, {). Wholly-wildcard globs (e.g. "**/*.go") have no
// literal prefix and are skipped. Globs whose prefix would escape projectRoot
// (via "..") are rejected unconditionally; the same traversal check applies
// to creates entries — a creates entry that escapes the root is itself a
// problem.
//
// Returns a structured report listing every problem and, when problems
// remain, an error wrapping ErrAcceptanceMismatch so callers can do
// errors.Is(err, ErrAcceptanceMismatch). The report is always non-nil.
//
// projectRoot must be absolute.
func ValidateAcceptance(projectRoot string, allow, creates []string) (*AcceptanceReport, error) {
	if !filepath.IsAbs(projectRoot) {
		return &AcceptanceReport{}, fmt.Errorf("projectRoot must be absolute, got %q", projectRoot)
	}
	cleanRoot := filepath.Clean(projectRoot)

	// Pre-compute creates prefixes for matching; reject any creates entry
	// that escapes the project root.
	var createsPrefixes []string
	report := &AcceptanceReport{}
	for _, c := range creates {
		prefix := literalPrefix(c)
		if prefix == "" {
			// Wholly-wildcard creates entry covers nothing literal; skip.
			continue
		}
		joined := filepath.Clean(filepath.Join(cleanRoot, prefix))
		rel, err := filepath.Rel(cleanRoot, joined)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			report.Problems = append(report.Problems, GlobProblem{
				Glob:        c,
				ResolvedRel: "",
				Reason:      "creates entry resolves outside project root",
			})
			continue
		}
		createsPrefixes = append(createsPrefixes, filepath.ToSlash(rel))
	}

	for _, glob := range allow {
		prefix := literalPrefix(glob)
		if prefix == "" {
			continue
		}
		joined := filepath.Clean(filepath.Join(cleanRoot, prefix))
		rel, err := filepath.Rel(cleanRoot, joined)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			report.Problems = append(report.Problems, GlobProblem{
				Glob:        glob,
				ResolvedRel: "",
				Reason:      "resolves outside project root",
			})
			continue
		}
		relSlash := filepath.ToSlash(rel)
		if _, statErr := os.Stat(joined); statErr == nil {
			// path exists — clean
			continue
		} else if !os.IsNotExist(statErr) {
			report.Problems = append(report.Problems, GlobProblem{
				Glob:        glob,
				ResolvedRel: relSlash,
				Reason:      fmt.Sprintf("stat: %v", statErr),
			})
			continue
		}
		// path does not exist — check creates coverage
		if coveredByCreates(relSlash, createsPrefixes) {
			// declared as a forward-looking path; not a problem.
			continue
		}
		report.Problems = append(report.Problems, GlobProblem{
			Glob:        glob,
			ResolvedRel: relSlash,
			Reason:      "path does not exist under project root",
		})
	}

	if len(report.Problems) > 0 {
		var summaries []string
		for _, p := range report.Problems {
			if p.ResolvedRel == "" {
				summaries = append(summaries, fmt.Sprintf("glob %q: %s", p.Glob, p.Reason))
			} else {
				summaries = append(summaries, fmt.Sprintf("glob %q: path %q: %s", p.Glob, p.ResolvedRel, p.Reason))
			}
		}
		return report, fmt.Errorf("%w: %s", ErrAcceptanceMismatch, strings.Join(summaries, "; "))
	}
	return report, nil
}

// coveredByCreates reports whether a glob's resolved relative path is a
// descendant of (or equal to) any creates prefix. The plan's `creates` entry
// covers a path "scenarios/foo" if the entry's literal prefix is also
// "scenarios/foo" or any ancestor.
func coveredByCreates(relSlash string, createsPrefixes []string) bool {
	for _, c := range createsPrefixes {
		if c == "" {
			continue
		}
		if relSlash == c || strings.HasPrefix(relSlash, c+"/") {
			return true
		}
		// Symmetrically: a creates entry that is more specific than the
		// acceptance prefix (e.g. acceptance "docs", creates "docs/internal/SANDBOX-CONTRACT.md")
		// also indicates the acceptance prefix's nonexistence is intentional —
		// the work creates content underneath it.
		if strings.HasPrefix(c, relSlash+"/") {
			return true
		}
	}
	return false
}

// literalPrefix returns the directory portion of a glob's literal prefix —
// everything up to (but not including) the segment that contains the first
// wildcard character. A glob with no wildcard returns the whole path; a glob
// that begins with a wildcard returns "".
func literalPrefix(glob string) string {
	g := filepath.ToSlash(strings.TrimSpace(glob))
	if g == "" {
		return ""
	}
	idx := strings.IndexAny(g, "*?[{")
	if idx < 0 {
		return strings.TrimSuffix(g, "/")
	}
	prefix := g[:idx]
	last := strings.LastIndex(prefix, "/")
	if last < 0 {
		return ""
	}
	return prefix[:last]
}
