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

	// Count non-blank, non-comment lines.
	entryCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			entryCount++
		}
	}

	// Normalize prefixes to have trailing slash.
	type normalizedRule struct {
		rule     GroupingRule
		prefixes []string
	}
	var rules []normalizedRule
	for _, r := range cfg.Rules {
		var normed []string
		for _, p := range r.Prefixes {
			normed = append(normed, normalizePrefix(p))
		}
		rules = append(rules, normalizedRule{rule: r, prefixes: normed})
	}

	var suggestions []GitignoreSuggestion

	for lineIdx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
			continue
		}
		lineNum := lineIdx + 1

		for _, nr := range rules {
			for _, prefix := range nr.prefixes {
				mode := nr.rule.Mode
				if mode == "" {
					mode = "prefix"
				}

				if !strings.HasPrefix(trimmed, prefix) {
					continue
				}

				remainder := trimmed[len(prefix):]

				if mode == "prefix" {
					groupDir := prefix
					targetPattern := remainder
					hasGI := groupGitignoreExists(deps, groupDir)
					suggestions = append(suggestions, GitignoreSuggestion{
						Line:          lineNum,
						Pattern:       trimmed,
						Type:          "single_group",
						GroupLabel:    nr.rule.Label,
						GroupDir:      groupDir,
						TargetPattern: targetPattern,
						HasGitignore:  hasGI,
					})
				} else {
					// segment mode
					if remainder == "" {
						continue
					}
					slashIdx := strings.Index(remainder, "/")
					var segment, after string
					if slashIdx < 0 {
						segment = remainder
						after = ""
					} else {
						segment = remainder[:slashIdx]
						after = remainder[slashIdx+1:]
					}

					if containsWildcard(segment) {
						// cross_group: wildcard in segment
						hasGI := groupGitignoreExists(deps, prefix)
						suggestions = append(suggestions, GitignoreSuggestion{
							Line:          lineNum,
							Pattern:       trimmed,
							Type:          "cross_group",
							GroupLabel:    nr.rule.Label,
							GroupDir:      prefix,
							TargetPattern: remainder,
							HasGitignore:  hasGI,
						})
					} else {
						// single_group
						groupDir := prefix + segment + "/"
						targetPattern := after
						hasGI := groupGitignoreExists(deps, groupDir)
						suggestions = append(suggestions, GitignoreSuggestion{
							Line:          lineNum,
							Pattern:       trimmed,
							Type:          "single_group",
							GroupLabel:    segment,
							GroupDir:      groupDir,
							TargetPattern: targetPattern,
							HasGitignore:  hasGI,
						})
					}
				}
			}
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Line < suggestions[j].Line
	})

	return &GitignoreHealthResponse{
		RootEntryCount: entryCount,
		Suggestions:    suggestions,
	}, nil
}

// MoveGitignoreEntry moves a single entry from the root .gitignore into a
// group-level .gitignore, validating the line number and pattern before acting.
func MoveGitignoreEntry(deps HealthDeps, req GitignoreMoveRequest) (*GitignoreMoveResponse, error) {
	if !isCleanSubpath(req.GroupDir) {
		return &GitignoreMoveResponse{
			Success: false,
			Error:   "invalid group directory",
		}, nil
	}

	targetGitignore := filepath.Join(deps.RepoDir, req.GroupDir, ".gitignore")

	// Add entry to group .gitignore first.
	if err := ensureIgnoreEntriesFS(deps.FS, targetGitignore, []string{req.TargetPattern}); err != nil {
		return nil, fmt.Errorf("add to group .gitignore: %w", err)
	}

	// Remove from root .gitignore.
	rootGitignore := filepath.Join(deps.RepoDir, ".gitignore")
	raw, err := deps.FS.ReadFile(rootGitignore)
	if err != nil {
		return nil, fmt.Errorf("read root .gitignore: %w", err)
	}

	lines := strings.Split(string(raw), "\n")
	targetIdx := req.Line - 1
	if targetIdx < 0 || targetIdx >= len(lines) {
		return &GitignoreMoveResponse{
			Success: false,
			Error:   fmt.Sprintf("line %d out of range (file has %d lines)", req.Line, len(lines)),
		}, nil
	}
	if strings.TrimSpace(lines[targetIdx]) != strings.TrimSpace(req.Pattern) {
		return &GitignoreMoveResponse{
			Success: false,
			Error:   fmt.Sprintf("pattern mismatch at line %d: expected %q, got %q", req.Line, req.Pattern, strings.TrimSpace(lines[targetIdx])),
		}, nil
	}

	// Remove the line.
	lines = append(lines[:targetIdx], lines[targetIdx+1:]...)

	// Collapse consecutive blank lines.
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

	newContent := strings.Join(collapsed, "\n")
	if err := deps.FS.WriteFile(rootGitignore, []byte(newContent), 0o644); err != nil {
		return nil, fmt.Errorf("write root .gitignore: %w", err)
	}

	return &GitignoreMoveResponse{
		Success:     true,
		RemovedFrom: rootGitignore,
		AddedTo:     targetGitignore,
	}, nil
}

// ensureIgnoreEntriesFS is like ensureIgnoreEntries but uses FileIO instead of
// os directly, enabling test injection.
func ensureIgnoreEntriesFS(fs FileIO, gitignorePath string, entries []string) error {
	var content string
	raw, err := fs.ReadFile(gitignorePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read .gitignore: %w", err)
		}
		// Create parent directory for new .gitignore files.
		if mkErr := fs.MkdirAll(filepath.Dir(gitignorePath), 0o755); mkErr != nil {
			return fmt.Errorf("create directory: %w", mkErr)
		}
	} else {
		content = string(raw)
	}

	existing := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		normalized := strings.TrimPrefix(trimmed, "/")
		existing[normalized] = true
	}

	var toAdd []string
	for _, entry := range entries {
		normalized := strings.TrimPrefix(strings.TrimSpace(entry), "/")
		if normalized == "" {
			continue
		}
		if existing[normalized] {
			continue
		}
		existing[normalized] = true
		toAdd = append(toAdd, normalized)
	}

	if len(toAdd) == 0 {
		return nil
	}

	var builder strings.Builder
	builder.WriteString(content)
	if content != "" && !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}
	for _, entry := range toAdd {
		builder.WriteString(entry)
		builder.WriteString("\n")
	}

	if err := fs.WriteFile(gitignorePath, []byte(builder.String()), 0o644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}

// normalizePrefix ensures the prefix ends with a trailing slash.
func normalizePrefix(p string) string {
	if p == "" {
		return "/"
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
