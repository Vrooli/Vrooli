package domains

import "strings"

// PathMatches reports whether a repo-relative path is covered by a path
// pattern. It accepts the three shapes domain sources use:
//
//   - manifest-style globs: "api/internal/graph/**" (recursive),
//     "api/internal/graph/*" (one level)
//   - DOMAINS.md directory prefixes: "api/internal/graph/" (recursive)
//   - exact files: "api/main.go"
//
// Matching is purely lexical over forward-slash paths; it does not touch
// the filesystem. A "**" pattern matches everything.
func PathMatches(path, pattern string) bool {
	path = strings.TrimSpace(path)
	pattern = strings.TrimSpace(pattern)
	if path == "" || pattern == "" {
		return false
	}
	switch {
	case pattern == "**":
		return true
	case strings.HasSuffix(pattern, "/**"):
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	case strings.HasSuffix(pattern, "/*"):
		prefix := strings.TrimSuffix(pattern, "/*")
		if !strings.HasPrefix(path, prefix+"/") {
			return false
		}
		// One level only: no further slash after the prefix.
		return !strings.Contains(path[len(prefix)+1:], "/")
	case strings.HasSuffix(pattern, "/"):
		prefix := strings.TrimSuffix(pattern, "/")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	default:
		return path == pattern
	}
}

// NormalizePath canonicalizes a path token: trims whitespace and
// surrounding backticks (DOMAINS.md wraps paths in backticks) and strips a
// leading "./".
func NormalizePath(token string) string {
	token = strings.TrimSpace(token)
	token = strings.Trim(token, "`")
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "./")
	return token
}
