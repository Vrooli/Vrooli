package bundle

import (
	"path/filepath"
	"strings"
)

// IsExcluded checks if a path matches any exclusion pattern.
// Supports five glob pattern styles (see individual matchers for details).
func IsExcluded(path string, patterns []string) bool {
	p := filepath.ToSlash(filepath.Clean(path))
	for _, pat := range patterns {
		pat = filepath.ToSlash(filepath.Clean(pat))
		if matchesExclusionPattern(p, pat) {
			return true
		}
	}
	return false
}

// matchesExclusionPattern checks if path matches a single exclusion pattern.
// Pattern types (in order of precedence):
//   - **/segment/**  : segment appears anywhere as path component
//   - **/filename    : filename at any depth
//   - prefix/**      : prefix directory and all contents
//   - prefix/**/seg  : prefix + segment anywhere after
//   - simple glob    : standard glob (no **)
func matchesExclusionPattern(path, pattern string) bool {
	switch {
	case matchesAnywhereSegment(path, pattern):
		return true
	case matchesAnywhereFile(path, pattern):
		return true
	case matchesPrefixGlob(path, pattern):
		return true
	case matchesNestedSegmentGlob(path, pattern):
		return true
	default:
		return matchesSimpleGlob(path, pattern)
	}
}

// matchesAnywhereSegment handles **/segment/** patterns.
// Returns true if segment appears anywhere as a complete path component.
// Example: **/node_modules/** matches "foo/node_modules/bar"
func matchesAnywhereSegment(path, pattern string) bool {
	if !strings.HasPrefix(pattern, "**/") || !strings.HasSuffix(pattern, "/**") {
		return false
	}
	segment := strings.TrimSuffix(strings.TrimPrefix(pattern, "**/"), "/**")
	return segment != "" && containsPathSegment(path, segment)
}

// matchesAnywhereFile handles **/filename patterns (no trailing /**).
// Returns true if filename appears at any depth.
// Example: **/.DS_Store matches "any/path/.DS_Store"
func matchesAnywhereFile(path, pattern string) bool {
	if !strings.HasPrefix(pattern, "**/") || strings.HasSuffix(pattern, "/**") {
		return false
	}
	suffix := strings.TrimPrefix(pattern, "**/")
	return path == suffix || strings.HasSuffix(path, "/"+suffix)
}

// matchesPrefixGlob handles prefix/** patterns (no ** in middle).
// Returns true if path starts with prefix or is exactly prefix.
// Example: coverage/** matches "coverage/reports/test.xml"
func matchesPrefixGlob(path, pattern string) bool {
	if !strings.HasSuffix(pattern, "/**") {
		return false
	}
	prefixPart := pattern[:len(pattern)-3]
	if strings.Contains(prefixPart, "**") {
		return false // Has ** in middle - not a simple prefix glob
	}
	prefix := strings.TrimSuffix(pattern, "/**")
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// matchesNestedSegmentGlob handles prefix/**/segment/** patterns.
// Returns true if path starts with prefix and contains segment(s) anywhere after.
// Example: scenarios/**/dist/** matches "scenarios/foo/bar/dist/file.js"
func matchesNestedSegmentGlob(path, pattern string) bool {
	if !strings.Contains(pattern, "/**/") {
		return false
	}

	parts := strings.Split(pattern, "/**/")
	if len(parts) < 2 {
		return false
	}

	prefix := parts[0]
	if !strings.HasPrefix(path, prefix+"/") && path != prefix {
		return false
	}

	remainder := strings.TrimPrefix(path, prefix+"/")
	for i := 1; i < len(parts); i++ {
		segment := strings.TrimSuffix(parts[i], "/**")
		if segment == "" {
			continue
		}
		if !containsPathSegment(remainder, segment) {
			return false
		}
		// Advance remainder past this segment
		remainder = advancePastSegment(remainder, segment)
	}
	return true
}

// advancePastSegment returns the portion of path after the first occurrence of segment.
func advancePastSegment(path, segment string) string {
	idx := strings.Index("/"+path+"/", "/"+segment+"/")
	if idx < 0 {
		return path
	}
	afterSegment := idx + len(segment) + 1
	if afterSegment < len(path)+1 {
		return path[afterSegment:]
	}
	return ""
}

// matchesSimpleGlob handles patterns without ** using filepath.Match.
// Example: *.log matches "error.log"
func matchesSimpleGlob(path, pattern string) bool {
	ok, err := filepath.Match(pattern, path)
	return err == nil && ok
}

func containsPathSegment(path, segment string) bool {
	path = "/" + strings.Trim(path, "/") + "/"
	segment = "/" + strings.Trim(segment, "/") + "/"
	return strings.Contains(path, segment)
}
