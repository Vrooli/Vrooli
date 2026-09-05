package dochealth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PlaceholderFixFile is one markdown file the placeholder fixer would (or
// did) rewrite. Before/After are complete file contents so callers can render
// a diff without re-reading the working tree.
type PlaceholderFixFile struct {
	Path     string // path relative to the scenario root
	AbsPath  string
	Before   string
	After    string
	FixCount int
	Lines    []int // 1-based lines the fixer touched, ascending
}

// PlaceholderFixResult reports the deterministic quoted-placeholder fix pass.
type PlaceholderFixResult struct {
	Scenario string
	DryRun   bool
	Files    []PlaceholderFixFile
	Skipped  []string // human-readable notes for fixes that could not be applied
}

// PlaceholderFix applies (or previews, with dryRun) the byte-exact
// quoted-placeholder fixes cli-health computes for unquoted_placeholder
// findings across a scenario's markdown files. The fix text is applied
// verbatim — never recomputed — so preview and apply select identical files
// and lines, and a second pass after apply is a no-op (the quoted form no
// longer yields unquoted_placeholder issues).
func (s *Service) PlaceholderFix(ctx context.Context, scenarioName string, dryRun bool) (*PlaceholderFixResult, error) {
	if s.commandValidator == nil {
		return nil, fmt.Errorf("cli-health command validator is not configured")
	}
	root, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, err
	}
	cfg := s.staticCfg.withOptions(DocHealthOptions{})
	files, err := collectMarkdownFiles(root, cfg)
	if err != nil {
		return nil, fmt.Errorf("collect markdown files: %w", err)
	}
	sort.Strings(files)

	result := &PlaceholderFixResult{Scenario: scenarioName, DryRun: dryRun}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		before := string(raw)
		after, touched, skipped := s.fixFilePlaceholders(ctx, root, file, before)
		result.Skipped = append(result.Skipped, skipped...)
		if after == before || len(touched) == 0 {
			continue
		}
		rel, relErr := filepath.Rel(root, file)
		if relErr != nil {
			rel = file
		}
		result.Files = append(result.Files, PlaceholderFixFile{
			Path:     rel,
			AbsPath:  file,
			Before:   before,
			After:    after,
			FixCount: len(touched),
			Lines:    touched,
		})
		if !dryRun {
			info, statErr := os.Stat(file)
			mode := os.FileMode(0o644)
			if statErr == nil {
				mode = info.Mode()
			}
			if writeErr := os.WriteFile(file, []byte(after), mode); writeErr != nil {
				return nil, fmt.Errorf("write %s: %w", rel, writeErr)
			}
		}
	}
	return result, nil
}

// fixFilePlaceholders computes the fixed content for one markdown file:
// every snippet with an unquoted_placeholder issue has its command text
// replaced in place by the issue's byte-exact fix. Returns the new content,
// the 1-based line numbers touched, and notes for fixes that could not land.
func (s *Service) fixFilePlaceholders(ctx context.Context, scenarioDir, file, content string) (string, []int, []string) {
	lines := strings.Split(content, "\n")
	var touched []int
	var skipped []string
	for _, snippet := range extractCommandSnippets(scenarioDir, file, content) {
		result, err := s.commandValidator.ValidateCommandReference(ctx, CommandReferenceRequest{CommandText: snippet.Command})
		if err != nil {
			continue
		}
		fix := ""
		for _, issue := range result.Issues {
			if issue.Code == "unquoted_placeholder" && issue.Fix != "" {
				fix = issue.Fix
				break
			}
		}
		if fix == "" {
			continue
		}
		idx := snippet.Line - 1
		if idx < 0 || idx >= len(lines) {
			continue
		}
		if !strings.Contains(lines[idx], snippet.Command) {
			skipped = append(skipped, fmt.Sprintf("%s:%d: snippet text not found on its line; left unchanged", file, snippet.Line))
			continue
		}
		lines[idx] = strings.Replace(lines[idx], snippet.Command, fix, 1)
		touched = append(touched, snippet.Line)
	}
	return strings.Join(lines, "\n"), touched, skipped
}

// UnifiedDiff renders a minimal unified diff for the fixer's line-local
// edits: one hunk per changed line.
func (f PlaceholderFixFile) UnifiedDiff() string {
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", f.Path, f.Path)
	beforeLines := strings.Split(f.Before, "\n")
	afterLines := strings.Split(f.After, "\n")
	for _, line := range f.Lines {
		if line-1 >= len(beforeLines) || line-1 >= len(afterLines) {
			continue
		}
		fmt.Fprintf(&b, "@@ -%d +%d @@\n-%s\n+%s\n", line, line, beforeLines[line-1], afterLines[line-1])
	}
	return b.String()
}
