package repocontract

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// ValidateRepoGlob ensures a pattern obeys repo-contract glob rules.
func ValidateRepoGlob(pattern string) error {
	pattern, err := normalizeRepoRelative(pattern, true)
	if err != nil {
		return err
	}
	if _, err := doublestar.Match(pattern, ""); err != nil {
		return &Error{Kind: ErrInvalidInput, Message: "invalid glob syntax", Details: pattern, Err: err}
	}
	return nil
}

// MatchRepoGlob matches a root-relative path against a repo glob using the
// contract's slash-normalized doublestar semantics.
func MatchRepoGlob(pattern, relPath string) (bool, error) {
	pattern, err := normalizeRepoRelative(pattern, true)
	if err != nil {
		return false, err
	}
	relPath, err = normalizeRepoRelative(relPath, false)
	if err != nil {
		return false, err
	}
	ok, err := doublestar.Match(pattern, relPath)
	if err != nil {
		return false, &Error{Kind: ErrInvalidInput, Message: "invalid glob syntax", Details: pattern, Err: err}
	}
	return ok, nil
}

func (c *Contract) MatchRepoGlob(pattern, relPath string) (bool, error) {
	return MatchRepoGlob(pattern, relPath)
}

func (c *Contract) AffectedScenarios(patterns []string) []string {
	scenarioDir := c.doc.Layout.ScenarioDir
	seen := map[string]struct{}{}

	for _, raw := range patterns {
		pattern, err := normalizeRepoRelative(raw, true)
		if err != nil || pattern == "" {
			continue
		}
		parts := strings.Split(pattern, "/")
		if len(parts) < 2 || parts[0] != scenarioDir {
			continue
		}
		candidate := parts[1]
		if candidate == "" || containsGlobMeta(candidate) {
			continue
		}
		seen[candidate] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for scenario := range seen {
		out = append(out, scenario)
	}
	sortStrings(out)
	return out
}

func normalizeRepoRelative(value string, forbidEmpty bool) (string, error) {
	value = filepathToSlashTrimmed(value)
	if value == "" {
		if forbidEmpty {
			return "", &Error{Kind: ErrInvalidInput, Message: "path must not be empty"}
		}
		return "", nil
	}
	if isAbsolutePathLike(value) {
		return "", &Error{Kind: ErrInvalidInput, Message: "absolute paths are not allowed", Details: value}
	}
	value = strings.TrimPrefix(value, "./")
	value = cleanSlashPath(value)
	if value == "." || value == "" {
		if forbidEmpty {
			return "", &Error{Kind: ErrInvalidInput, Message: "path must not be empty"}
		}
		return "", nil
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return "", &Error{Kind: ErrInvalidInput, Message: "path must not contain parent traversal", Details: value}
		}
	}
	return value, nil
}

func containsGlobMeta(value string) bool {
	return strings.ContainsAny(value, "*?[{")
}
