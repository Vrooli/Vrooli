package typescript

import "regexp"

// Patterns for TypeScript/JavaScript import statements
var (
	// Static imports: import X from 'source', import { X } from 'source', import * as X from 'source'
	staticImportPattern = regexp.MustCompile(`(?m)^\s*import\s+(?:(?:[\w*{}\s,]+)\s+from\s+)?['"]([^'"]+)['"]`)

	// Dynamic imports: import('source'), import("source")
	dynamicImportPattern = regexp.MustCompile(`import\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	// Re-exports: export * from 'source', export { X } from 'source'
	reExportPattern = regexp.MustCompile(`(?m)^\s*export\s+(?:\*|{[^}]*})\s+from\s+['"]([^'"]+)['"]`)

	// Side-effect imports: import 'source' (no bindings)
	sideEffectImportPattern = regexp.MustCompile(`(?m)^\s*import\s+['"]([^'"]+)['"]\s*;?\s*$`)
)

// MatchStaticImports finds all static import statements and returns (source, line) pairs
func MatchStaticImports(content string) []struct {
	Source string
	Line   int
} {
	return matchWithLines(content, staticImportPattern)
}

// MatchDynamicImports finds all dynamic import() expressions
func MatchDynamicImports(content string) []struct {
	Source string
	Line   int
} {
	return matchWithLines(content, dynamicImportPattern)
}

// MatchReExports finds all re-export statements
func MatchReExports(content string) []struct {
	Source string
	Line   int
} {
	return matchWithLines(content, reExportPattern)
}

// MatchSideEffectImports finds all side-effect imports
func MatchSideEffectImports(content string) []struct {
	Source string
	Line   int
} {
	return matchWithLines(content, sideEffectImportPattern)
}

// matchWithLines finds all matches and returns source with line numbers
func matchWithLines(content string, pattern *regexp.Regexp) []struct {
	Source string
	Line   int
} {
	var results []struct {
		Source string
		Line   int
	}

	// Build line index for efficient line number lookup
	lineStarts := buildLineIndex(content)

	matches := pattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			source := content[match[2]:match[3]]
			line := findLineNumber(lineStarts, match[0])
			results = append(results, struct {
				Source string
				Line   int
			}{Source: source, Line: line})
		}
	}

	return results
}

// buildLineIndex returns the starting index of each line
func buildLineIndex(content string) []int {
	lineStarts := []int{0}
	for i, ch := range content {
		if ch == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	return lineStarts
}

// findLineNumber returns the 1-indexed line number for the given position
func findLineNumber(lineStarts []int, pos int) int {
	for i := len(lineStarts) - 1; i >= 0; i-- {
		if lineStarts[i] <= pos {
			return i + 1
		}
	}
	return 1
}

// IsRelativeImport returns true if the import source is a relative path
func IsRelativeImport(source string) bool {
	return len(source) > 0 && (source[0] == '.' || source[0] == '/')
}
