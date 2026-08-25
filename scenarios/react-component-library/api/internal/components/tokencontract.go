package components

import (
	"regexp"
	"sort"
)

var (
	cssVariableReferenceRE   = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)`)
	cssVariableDeclarationRE = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
	dynamicTokenPatternRE    = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)-\$\{`)
)

// ExtractRequiredTokens derives the external CSS custom-property contract for
// one version. A property is required when the version references it through
// var(--name) but does not declare --name anywhere in its own source files.
// TS/TSX template literals are scanned as source text because the library
// stores several stylesheets in those literals.
func ExtractRequiredTokens(files []ComponentVersionFile) []string {
	referenced := make(map[string]struct{})
	declared := make(map[string]struct{})
	for _, file := range files {
		for _, match := range cssVariableReferenceRE.FindAllStringSubmatch(file.Content, -1) {
			if len(match) > 1 {
				referenced[match[1]] = struct{}{}
			}
		}
		for _, match := range cssVariableDeclarationRE.FindAllStringSubmatch(file.Content, -1) {
			if len(match) > 1 {
				declared[match[1]] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(referenced))
	for property := range referenced {
		if _, defined := declared[property]; !defined {
			result = append(result, property)
		}
	}
	sort.Strings(result)
	return result
}

// ExtractRequiredTokenPatterns derives dynamic CSS custom-property families
// such as --space-* from a version's source. The concrete suffix is selected
// at runtime, so it cannot be represented by RequiredTokens alone.
func ExtractRequiredTokenPatterns(files []ComponentVersionFile) []string {
	patterns := make(map[string]struct{})
	for _, file := range files {
		for _, match := range dynamicTokenPatternRE.FindAllStringSubmatch(file.Content, -1) {
			if len(match) == 2 {
				patterns[match[1]+"-*"] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(patterns))
	for pattern := range patterns {
		result = append(result, pattern)
	}
	sort.Strings(result)
	return result
}
