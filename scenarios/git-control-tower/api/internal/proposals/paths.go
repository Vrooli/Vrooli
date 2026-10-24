package proposals

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// CleanPath returns a safe repository-relative path or an error. It refuses
// absolute paths, parent traversal, option-looking paths, control characters
// and paths inside .git, so a path can never become a git option or leave the
// repository.
func CleanPath(raw string) (string, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if value == "" {
		return "", fmt.Errorf("%w: empty path", ErrInvalid)
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return "", fmt.Errorf("%w: path %q contains control characters", ErrInvalid, raw)
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "-") || strings.HasPrefix(value, ":") {
		return "", fmt.Errorf("%w: path %q must be repository-relative", ErrInvalid, raw)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: path %q leaves the repository", ErrInvalid, raw)
	}
	if cleaned == ".git" || strings.HasPrefix(cleaned, ".git/") {
		return "", fmt.Errorf("%w: path %q is inside .git", ErrInvalid, raw)
	}
	return cleaned, nil
}

// cleanPaths validates, de-duplicates and sorts paths.
func cleanPaths(raw []string) ([]string, error) {
	seen := map[string]struct{}{}
	var result []string
	for _, item := range raw {
		cleaned, err := CleanPath(item)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}
	sort.Strings(result)
	return result, nil
}

// scopeMatcher matches repository paths against git-pathspec-like scopes:
// a plain path matches itself and everything below it; "*" matches within
// one segment, "**" across segments and "?" one character.
type scopeMatcher struct{ patterns []*regexp.Regexp }

func newScopeMatcher(scopes []string) (scopeMatcher, error) {
	var matcher scopeMatcher
	for _, scope := range scopes {
		trimmed := strings.TrimSuffix(strings.TrimSpace(strings.ReplaceAll(scope, "\\", "/")), "/")
		if trimmed == "" || trimmed == "." || trimmed == "**" {
			matcher.patterns = append(matcher.patterns, regexp.MustCompile(`^.+$`))
			continue
		}
		if _, err := CleanPath(strings.NewReplacer("*", "x", "?", "x").Replace(trimmed)); err != nil {
			return scopeMatcher{}, fmt.Errorf("%w: scope %q", ErrInvalid, scope)
		}
		var builder strings.Builder
		builder.WriteString("^")
		for i := 0; i < len(trimmed); i++ {
			switch {
			case strings.HasPrefix(trimmed[i:], "**/"):
				builder.WriteString("(?:.*/)?")
				i += 2
			case strings.HasPrefix(trimmed[i:], "**"):
				builder.WriteString(".*")
				i++
			case trimmed[i] == '*':
				builder.WriteString("[^/]*")
			case trimmed[i] == '?':
				builder.WriteString("[^/]")
			default:
				builder.WriteString(regexp.QuoteMeta(trimmed[i : i+1]))
			}
		}
		builder.WriteString("(?:/.*)?$")
		pattern, err := regexp.Compile(builder.String())
		if err != nil {
			return scopeMatcher{}, fmt.Errorf("%w: scope %q", ErrInvalid, scope)
		}
		matcher.patterns = append(matcher.patterns, pattern)
	}
	return matcher, nil
}

func (m scopeMatcher) Match(p string) bool {
	for _, pattern := range m.patterns {
		if pattern.MatchString(p) {
			return true
		}
	}
	return false
}
