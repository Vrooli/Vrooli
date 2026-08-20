package components

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	cssVariableReferenceRE   = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)`)
	cssVariableDeclarationRE = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
	cssVariableNameRE        = regexp.MustCompile(`^--[A-Za-z0-9_-]+$`)
	dynamicTokenPatternRE    = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)-\$\{`)
)

func normalizeManifestRequiredTokens(path string, properties []string) ([]string, error) {
	result := make(map[string]struct{}, len(properties))
	for _, property := range properties {
		property = strings.TrimSpace(property)
		if !cssVariableNameRE.MatchString(property) {
			return nil, ErrInvalidHeader{SourcePath: path, Field: "requiredTokens", Reason: fmt.Sprintf("%q is not a CSS custom property", property)}
		}
		result[property] = struct{}{}
	}
	return sortedTokenSet(result), nil
}

func mergeRequiredTokens(sets ...[]string) []string {
	merged := map[string]struct{}{}
	for _, set := range sets {
		for _, property := range set {
			merged[property] = struct{}{}
		}
	}
	return sortedTokenSet(merged)
}

func sortedTokenSet(properties map[string]struct{}) []string {
	result := make([]string, 0, len(properties))
	for property := range properties {
		result = append(result, property)
	}
	sort.Strings(result)
	return result
}

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
