package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HealthDeps contains dependencies for gitignore health analysis.
type HealthDeps struct {
	FS           FileIO
	RepoDir      string
	GroupingDeps GroupingDeps
}

// normalizedRule pairs a grouping rule with its normalized prefixes.
type normalizedRule struct {
	rule     GroupingRule
	prefixes []string
}

// AnalyzeGitignoreHealth inspects the root .gitignore and suggests entries
// that could be moved into group-level .gitignore files.
func AnalyzeGitignoreHealth(deps HealthDeps) (*GitignoreHealthResponse, error) {
	cfg, err := LoadGroupingRules(deps.GroupingDeps)
	if err != nil {
		return nil, fmt.Errorf("load grouping rules: %w", err)
	}
	if cfg == nil || len(cfg.Rules) == 0 {
		return &GitignoreHealthResponse{}, nil
	}

	rootGitignore := filepath.Join(deps.RepoDir, ".gitignore")
	raw, err := deps.FS.ReadFile(rootGitignore)
	if err != nil {
		if os.IsNotExist(err) {
			return &GitignoreHealthResponse{}, nil
		}
		return nil, fmt.Errorf("read root .gitignore: %w", err)
	}

	lines := strings.Split(string(raw), "\n")
	entryCount := countGitignoreEntries(lines)
	rules := normalizeRules(cfg.Rules)
	suggestions := collectSuggestions(deps, lines, rules)

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Line < suggestions[j].Line
	})

	return &GitignoreHealthResponse{
		RootEntryCount: entryCount,
		Suggestions:    suggestions,
	}, nil
}

// countGitignoreEntries counts non-blank, non-comment lines.
func countGitignoreEntries(lines []string) int {
	count := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			count++
		}
	}
	return count
}

// normalizeRules normalizes prefixes to have trailing slashes.
func normalizeRules(rules []GroupingRule) []normalizedRule {
	var result []normalizedRule
	for _, r := range rules {
		var normed []string
		for _, p := range r.Prefixes {
			normed = append(normed, normalizePrefix(p))
		}
		result = append(result, normalizedRule{rule: r, prefixes: normed})
	}
	return result
}

// collectSuggestions scans gitignore lines against rules and collects suggestions.
func collectSuggestions(deps HealthDeps, lines []string, rules []normalizedRule) []GitignoreSuggestion {
	var suggestions []GitignoreSuggestion
	for lineIdx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
			continue
		}
		lineNum := lineIdx + 1
		for _, nr := range rules {
			suggestions = matchRulePrefixes(deps, trimmed, lineNum, nr, suggestions)
		}
	}
	return suggestions
}

// matchRulePrefixes checks a single line against all prefixes of a rule.
func matchRulePrefixes(deps HealthDeps, trimmed string, lineNum int, nr normalizedRule, suggestions []GitignoreSuggestion) []GitignoreSuggestion {
	for _, prefix := range nr.prefixes {
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		mode := nr.rule.Mode
		if mode == "" {
			mode = "prefix"
		}
		remainder := trimmed[len(prefix):]

		if mode == "prefix" {
			suggestions = append(suggestions, buildPrefixSuggestion(deps, trimmed, lineNum, nr.rule.Label, prefix, remainder))
		} else {
			suggestions = appendSegmentSuggestion(deps, trimmed, lineNum, nr.rule.Label, prefix, remainder, suggestions)
		}
	}
	return suggestions
}

// buildPrefixSuggestion builds a single_group suggestion for prefix mode.
func buildPrefixSuggestion(deps HealthDeps, pattern string, lineNum int, label, groupDir, targetPattern string) GitignoreSuggestion {
	return GitignoreSuggestion{
		Line:          lineNum,
		Pattern:       pattern,
		Type:          "single_group",
		GroupLabel:    label,
		GroupDir:      groupDir,
		TargetPattern: targetPattern,
		HasGitignore:  groupGitignoreExists(deps, groupDir),
	}
}

// appendSegmentSuggestion handles segment mode matching and appends suggestions.
func appendSegmentSuggestion(deps HealthDeps, pattern string, lineNum int, label, prefix, remainder string, suggestions []GitignoreSuggestion) []GitignoreSuggestion {
	if remainder == "" {
		return suggestions
	}
	slashIdx := strings.Index(remainder, "/")
	var segment, after string
	if slashIdx < 0 {
		segment = remainder
	} else {
		segment = remainder[:slashIdx]
		after = remainder[slashIdx+1:]
	}

	if containsWildcard(segment) {
		suggestions = append(suggestions, GitignoreSuggestion{
			Line:          lineNum,
			Pattern:       pattern,
			Type:          "cross_group",
			GroupLabel:    label,
			GroupDir:      prefix,
			TargetPattern: remainder,
			HasGitignore:  groupGitignoreExists(deps, prefix),
		})
	} else {
		groupDir := prefix + segment + "/"
		suggestions = append(suggestions, GitignoreSuggestion{
			Line:          lineNum,
			Pattern:       pattern,
			Type:          "single_group",
			GroupLabel:    segment,
			GroupDir:      groupDir,
			TargetPattern: after,
			HasGitignore:  groupGitignoreExists(deps, groupDir),
		})
	}
	return suggestions
}

// MoveGitignoreEntry moves a single entry from the root .gitignore into a
// group-level .gitignore, validating the line number and pattern before acting.
func MoveGitignoreEntry(deps HealthDeps, req GitignoreMoveRequest) (*GitignoreMoveResponse, error) {
	if !isCleanSubpath(req.GroupDir) {
		return &GitignoreMoveResponse{Success: false, Error: "invalid group directory"}, nil
	}

	targetGitignore := filepath.Join(deps.RepoDir, req.GroupDir, ".gitignore")
	if err := ensureIgnoreEntriesFS(deps.FS, targetGitignore, []string{req.TargetPattern}); err != nil {
		return nil, fmt.Errorf("add to group .gitignore: %w", err)
	}

	rootGitignore := filepath.Join(deps.RepoDir, ".gitignore")
	raw, err := deps.FS.ReadFile(rootGitignore)
	if err != nil {
		return nil, fmt.Errorf("read root .gitignore: %w", err)
	}

	lines := strings.Split(string(raw), "\n")
	if errMsg := validateLineMatch(lines, req.Line, req.Pattern); errMsg != "" {
		return &GitignoreMoveResponse{Success: false, Error: errMsg}, nil
	}

	lines = append(lines[:req.Line-1], lines[req.Line:]...)
	newContent := strings.Join(collapseBlankLines(lines), "\n")
	if err := deps.FS.WriteFile(rootGitignore, []byte(newContent), 0o644); err != nil {
		return nil, fmt.Errorf("write root .gitignore: %w", err)
	}

	return &GitignoreMoveResponse{Success: true, RemovedFrom: rootGitignore, AddedTo: targetGitignore}, nil
}

// validateLineMatch checks that the line number is in range and the pattern matches.
func validateLineMatch(lines []string, lineNum int, pattern string) string {
	idx := lineNum - 1
	if idx < 0 || idx >= len(lines) {
		return fmt.Sprintf("line %d out of range (file has %d lines)", lineNum, len(lines))
	}
	if strings.TrimSpace(lines[idx]) != strings.TrimSpace(pattern) {
		return fmt.Sprintf("pattern mismatch at line %d: expected %q, got %q", lineNum, pattern, strings.TrimSpace(lines[idx]))
	}
	return ""
}

// collapseBlankLines removes consecutive blank lines from a slice.
func collapseBlankLines(lines []string) []string {
	var collapsed []string
	prevBlank := false
	for _, l := range lines {
		blank := strings.TrimSpace(l) == ""
		if blank && prevBlank {
			continue
		}
		collapsed = append(collapsed, l)
		prevBlank = blank
	}
	return collapsed
}

// ensureIgnoreEntriesFS is like ensureIgnoreEntries but uses FileIO instead of
// os directly, enabling test injection.
func ensureIgnoreEntriesFS(fs FileIO, gitignorePath string, entries []string) error {
	content, err := readOrCreateGitignore(fs, gitignorePath)
	if err != nil {
		return err
	}

	existing := parseExistingEntries(content)
	toAdd := filterNewEntries(entries, existing)
	if len(toAdd) == 0 {
		return nil
	}

	newContent := appendEntries(content, toAdd)
	if err := fs.WriteFile(gitignorePath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}

// readOrCreateGitignore reads an existing gitignore or creates its parent directory.
func readOrCreateGitignore(fs FileIO, gitignorePath string) (string, error) {
	raw, err := fs.ReadFile(gitignorePath)
	if err == nil {
		return string(raw), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read .gitignore: %w", err)
	}
	if mkErr := fs.MkdirAll(filepath.Dir(gitignorePath), 0o755); mkErr != nil {
		return "", fmt.Errorf("create directory: %w", mkErr)
	}
	return "", nil
}

// parseExistingEntries extracts normalized entry names from gitignore content.
func parseExistingEntries(content string) map[string]bool {
	existing := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		existing[strings.TrimPrefix(trimmed, "/")] = true
	}
	return existing
}

// filterNewEntries returns entries not already present in the existing set.
func filterNewEntries(entries []string, existing map[string]bool) []string {
	var toAdd []string
	for _, entry := range entries {
		normalized := strings.TrimPrefix(strings.TrimSpace(entry), "/")
		if normalized == "" || existing[normalized] {
			continue
		}
		existing[normalized] = true
		toAdd = append(toAdd, normalized)
	}
	return toAdd
}

// appendEntries appends new entries to existing gitignore content.
func appendEntries(content string, toAdd []string) string {
	var builder strings.Builder
	builder.WriteString(content)
	if content != "" && !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}
	for _, entry := range toAdd {
		builder.WriteString(entry)
		builder.WriteString("\n")
	}
	return builder.String()
}

// normalizePrefix ensures the prefix ends with a trailing slash.
func normalizePrefix(p string) string {
	if p == "" {
		return ""
	}
	if !strings.HasSuffix(p, "/") {
		return p + "/"
	}
	return p
}

// isCleanSubpath validates that path has no ".." components and is non-empty.
func isCleanSubpath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	cleaned := filepath.Clean(path)
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return false
		}
	}
	return true
}

// containsWildcard returns true if s contains glob characters.
func containsWildcard(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// groupGitignoreExists checks whether a .gitignore file exists in the given
// group directory (relative to repo root).
func groupGitignoreExists(deps HealthDeps, groupDir string) bool {
	p := filepath.Join(deps.RepoDir, groupDir, ".gitignore")
	_, err := deps.FS.Stat(p)
	return err == nil
}
