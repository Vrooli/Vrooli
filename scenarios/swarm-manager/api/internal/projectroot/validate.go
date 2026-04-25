package projectroot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrAcceptanceMismatch is returned by ValidateAcceptanceUnderRoot when one
// or more globs reference paths that do not exist under projectRoot.
var ErrAcceptanceMismatch = errors.New("acceptance_allow paths do not exist under project root")

// ValidateAcceptanceUnderRoot asserts that the directory or file implied by
// each glob's literal prefix exists on disk under projectRoot. It is a fail-
// closed safety net: if the resolver picked the wrong root, or a glob has a
// typo, this catches it before a sandbox is provisioned.
//
// For each glob the literal prefix is everything up to the first wildcard
// character (*, ?, [, {). Wholly-wildcard globs (e.g. "**/*.go") have no
// literal prefix and are skipped. Globs whose prefix would escape projectRoot
// (via "..") are rejected.
//
// projectRoot must be absolute. Returns nil if every glob validates, or an
// error wrapping ErrAcceptanceMismatch listing every problem.
func ValidateAcceptanceUnderRoot(projectRoot string, allow []string) error {
	if !filepath.IsAbs(projectRoot) {
		return fmt.Errorf("projectRoot must be absolute, got %q", projectRoot)
	}
	cleanRoot := filepath.Clean(projectRoot)

	var problems []string
	for _, glob := range allow {
		prefix := literalPrefix(glob)
		if prefix == "" {
			continue
		}
		joined := filepath.Clean(filepath.Join(cleanRoot, prefix))
		rel, err := filepath.Rel(cleanRoot, joined)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			problems = append(problems, fmt.Sprintf("glob %q resolves outside project root", glob))
			continue
		}
		if _, err := os.Stat(joined); err != nil {
			if os.IsNotExist(err) {
				problems = append(problems, fmt.Sprintf("glob %q: path %q does not exist under project root", glob, filepath.ToSlash(rel)))
			} else {
				problems = append(problems, fmt.Sprintf("glob %q: stat: %v", glob, err))
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrAcceptanceMismatch, strings.Join(problems, "; "))
	}
	return nil
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
