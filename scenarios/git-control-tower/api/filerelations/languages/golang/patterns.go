package golang

import "regexp"

// Patterns for Go import statements
var (
	// Single import: import "fmt"
	singleImportPattern = regexp.MustCompile(`(?m)^\s*import\s+"([^"]+)"`)

	// Import block: import ( ... )
	importBlockPattern = regexp.MustCompile(`(?s)import\s*\(\s*(.*?)\s*\)`)

	// Import line within a block: "path" or alias "path"
	importLinePattern = regexp.MustCompile(`(?m)^\s*(?:[\w.]+\s+)?"([^"]+)"`)
)

// ImportMatch represents a matched import with its line number
type ImportMatch struct {
	Path string
	Line int
}

// MatchSingleImports finds single-line import statements
func MatchSingleImports(content string) []ImportMatch {
	var results []ImportMatch
	lineStarts := buildLineIndex(content)

	matches := singleImportPattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			path := content[match[2]:match[3]]
			line := findLineNumber(lineStarts, match[0])
			results = append(results, ImportMatch{Path: path, Line: line})
		}
	}

	return results
}

// MatchImportBlocks finds all imports within import blocks
func MatchImportBlocks(content string) []ImportMatch {
	var results []ImportMatch
	lineStarts := buildLineIndex(content)

	// Find all import blocks
	blockMatches := importBlockPattern.FindAllStringSubmatchIndex(content, -1)
	for _, blockMatch := range blockMatches {
		if len(blockMatch) < 4 {
			continue
		}

		blockContent := content[blockMatch[2]:blockMatch[3]]
		blockStart := blockMatch[2]

		// Find imports within the block
		importMatches := importLinePattern.FindAllStringSubmatchIndex(blockContent, -1)
		for _, importMatch := range importMatches {
			if len(importMatch) >= 4 {
				path := blockContent[importMatch[2]:importMatch[3]]
				absolutePos := blockStart + importMatch[0]
				line := findLineNumber(lineStarts, absolutePos)
				results = append(results, ImportMatch{Path: path, Line: line})
			}
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

// IsRelativeImport returns true if the import is a relative path (starts with ./)
func IsRelativeImport(path string) bool {
	return len(path) > 0 && path[0] == '.'
}

// IsStandardLibrary returns true if the import appears to be a Go standard library
func IsStandardLibrary(path string) bool {
	// Standard library imports don't contain dots in the first segment
	for i, ch := range path {
		if ch == '/' {
			// Check first segment
			return true
		}
		if ch == '.' {
			return false
		}
		if i > 20 {
			// Long enough without slash - probably standard lib
			break
		}
	}
	return true
}
