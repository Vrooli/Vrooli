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
