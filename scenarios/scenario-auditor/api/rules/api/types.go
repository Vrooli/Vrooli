package api

import (
	rules "scenario-auditor/rules"
	"strings"
)

type Violation = rules.Violation

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isCommentOrBlank(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "//")
}

func isTestFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, "_test.go") {
		return true
	}
	return strings.Contains(lower, "/test/") || strings.Contains(lower, "\\test\\")
}

// isExemptPath returns true for paths that are exempt from database-substrate
// rules (test files, migrations, scripts, etc.). Shared by database_backoff,
// routed_database_drivers, and routed_database_handle_capture.
func isExemptPath(path string) bool {
	lowerPath := strings.ToLower(path)

	if strings.HasSuffix(lowerPath, "_test.go") {
		return true
	}

	base := lowerPath
	if idx := strings.LastIndex(lowerPath, "/"); idx >= 0 {
		base = lowerPath[idx+1:]
	}
	if strings.HasPrefix(base, "test_") {
		return true
	}

	exemptDirs := []string{
		"test",
		"testutil",
		"migrate",
		"migration",
		"migrations",
		"initialization",
		"init",
		"scripts",
		"tools",
	}

	for _, dir := range exemptDirs {
		if strings.Contains(lowerPath, "/"+dir+"/") {
			return true
		}
		if strings.HasPrefix(lowerPath, dir+"/") {
			return true
		}
	}

	return false
}

// isAPICorePath returns true for paths inside packages/api-core/, which is
// the substrate that owns the RoutedDB seam and therefore must be allowed to
// import drivers and call sql.OpenDB.
func isAPICorePath(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "packages/api-core/")
}

func findWithinWindow(lines []string, start, lookahead int, predicate func(string) bool) bool {
	end := min(len(lines), start+lookahead)
	for i := start; i < end; i++ {
		line := lines[i]
		if isCommentOrBlank(line) {
			continue
		}
		if predicate(line) {
			return true
		}
	}
	return false
}
