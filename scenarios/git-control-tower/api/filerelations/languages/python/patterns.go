package python

import (
	"regexp"

	"git-control-tower/filerelations/languages"
)

// Patterns for Python import statements
var (
	// import module, import module.submodule
	importPattern = regexp.MustCompile(`(?m)^\s*import\s+([\w.]+(?:\s*,\s*[\w.]+)*)`)

	// from module import X, from .module import X
	fromImportPattern = regexp.MustCompile(`(?m)^\s*from\s+(\.{0,2}[\w.]*)\s+import`)
)

// ImportMatch represents a matched import with its line number
type ImportMatch struct {
	Module string
	Line   int
}

// MatchImports finds all 'import module' statements
func MatchImports(content string) []ImportMatch {
	var results []ImportMatch
	lineStarts := languages.BuildLineIndex(content)

	matches := importPattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			modules := content[match[2]:match[3]]
			line := languages.FindLineNumber(lineStarts, match[0])
			// Handle multiple imports on one line: import a, b, c
			for _, mod := range splitModules(modules) {
				results = append(results, ImportMatch{Module: mod, Line: line})
			}
		}
	}

	return results
}

// MatchFromImports finds all 'from module import' statements
func MatchFromImports(content string) []ImportMatch {
	var results []ImportMatch
	lineStarts := languages.BuildLineIndex(content)

	matches := fromImportPattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			module := content[match[2]:match[3]]
			line := languages.FindLineNumber(lineStarts, match[0])
			results = append(results, ImportMatch{Module: module, Line: line})
		}
	}

	return results
}

// splitModules splits a comma-separated list of module names
func splitModules(modules string) []string {
	var result []string
	current := ""
	for _, ch := range modules {
		switch ch {
		case ',':
			if trimmed := trimSpaces(current); trimmed != "" {
				result = append(result, trimmed)
			}
			current = ""
		case ' ', '\t':
			// Skip whitespace
		default:
			current += string(ch)
		}
	}
	if trimmed := trimSpaces(current); trimmed != "" {
		result = append(result, trimmed)
	}
	return result
}

func trimSpaces(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// IsRelativeImport returns true if the import is a relative import (starts with .)
func IsRelativeImport(module string) bool {
	return len(module) > 0 && module[0] == '.'
}
