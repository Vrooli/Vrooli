package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

var errPatternOutside = errors.New("pattern outside campaign base")

type TargetResolution struct {
	Paths     []string
	Notes     map[string]string
	Unmatched []string
}

type PathMatcher interface {
	Match(pattern string) ([]string, error)
}

type campaignPathMatcher struct {
	baseDir string
	fsys    fs.FS
}

func newCampaignPathMatcher(campaign *Campaign) PathMatcher {
	baseDir := getCampaignBaseDir(campaign)
	return campaignPathMatcher{
		baseDir: baseDir,
		fsys:    os.DirFS(baseDir),
	}
}

func (m campaignPathMatcher) Match(pattern string) ([]string, error) {
	relPattern, ok := toCampaignRelativePattern(m.baseDir, pattern)
	if !ok {
		return nil, errPatternOutside
	}
	return doublestar.Glob(m.fsys, relPattern)
}

func resolveTargets(campaign *Campaign, files []string, fileNotes map[string]string) TargetResolution {
	return resolveTargetsWithMatcher(files, fileNotes, newCampaignPathMatcher(campaign))
}

func resolveTargetsWithMatcher(files []string, fileNotes map[string]string, matcher PathMatcher) TargetResolution {
	allPatterns := append([]string{}, files...)
	for key := range fileNotes {
		allPatterns = append(allPatterns, key)
	}

	expandedPaths, unmatchedPaths := expandPatterns(allPatterns, matcher)
	expandedNotes, unmatchedNotes := expandNotes(fileNotes, matcher)
	unmatched := append(unmatchedPaths, unmatchedNotes...)

	return TargetResolution{
		Paths:     expandedPaths,
		Notes:     expandedNotes,
		Unmatched: uniqueStrings(unmatched),
	}
}

func expandPatterns(patterns []string, matcher PathMatcher) ([]string, []string) {
	seen := make(map[string]struct{})
	var matches []string
	var unmatched []string

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		found, isGlob := expandPattern(pattern, matcher)
		if isGlob && len(found) == 0 {
			unmatched = append(unmatched, pattern)
			continue
		}
		for _, match := range found {
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			matches = append(matches, match)
		}
	}

	return matches, unmatched
}

func expandNotes(notes map[string]string, matcher PathMatcher) (map[string]string, []string) {
	if len(notes) == 0 {
		return map[string]string{}, nil
	}

	expanded := make(map[string]string)
	var unmatched []string
	keys := make([]string, 0, len(notes))
	for key := range notes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		note := notes[key]
		matches, isGlob := expandPattern(key, matcher)
		if isGlob && len(matches) == 0 {
			unmatched = append(unmatched, key)
			continue
		}
		for _, match := range matches {
			expanded[match] = note
		}
	}

	return expanded, unmatched
}

func expandPattern(pattern string, matcher PathMatcher) ([]string, bool) {
	if !hasGlobPattern(pattern) {
		return []string{pattern}, false
	}

	var matches []string
	for _, expanded := range expandBraces(pattern) {
		globMatches, err := matcher.Match(expanded)
		if err != nil {
			if !errors.Is(err, errPatternOutside) {
				logger.Printf("⚠️ Pattern glob failed for %s: %v", expanded, err)
			}
			continue
		}
		matches = append(matches, globMatches...)
	}

	return matches, true
}

func hasGlobPattern(path string) bool {
	return strings.ContainsAny(path, "*?[") || strings.Contains(path, "{")
}

func toCampaignRelativePattern(baseDir string, pattern string) (string, bool) {
	if !filepath.IsAbs(pattern) {
		return pattern, true
	}

	rel, err := filepath.Rel(baseDir, filepath.Clean(pattern))
	if err != nil {
		logger.Printf("⚠️ Could not resolve pattern %s relative to %s: %v", pattern, baseDir, err)
		return "", false
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		logger.Printf("⚠️ Pattern outside campaign base ignored: %s", pattern)
		return "", false
	}

	return rel, true
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
