package docsearch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const maxLineLength = 500

var defaultFileTypes = []string{"md", "markdown", "mdx", "txt", "json"}

// SearchText performs full-text search across documentation files.
func (s *Service) SearchText(ctx context.Context, req TextSearchRequest) ([]TextSearchMatch, error) {
	if s == nil {
		return nil, fmt.Errorf("doc search service unavailable")
	}
	if err := req.normalize(); err != nil {
		return nil, err
	}
	matcher, err := compileQuery(req.Query, req.CaseSensitive)
	if err != nil {
		return nil, err
	}
	fileTypes := normalizeFileTypes(req.FileTypes)
	files, err := s.collectDocFiles(ctx, req.Scope, req.Scenario, req.BasePath)
	if err != nil {
		return nil, err
	}
	results := make([]TextSearchMatch, 0, min(len(files), req.Limit))
	for _, file := range files {
		if !matchesFileType(file.Path, fileTypes) {
			continue
		}
		matches, err := searchFile(file.Path, matcher, req.ContextLines)
		if err != nil {
			continue
		}
		for _, match := range matches {
			match.Path = s.repoRelative(file.Path)
			match.RelativePath = s.relPath(file.RelBase, file.Path)
			match.Scenario = file.Scenario
			results = append(results, match)
			if len(results) >= req.Limit {
				break
			}
		}
		if len(results) >= req.Limit {
			break
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Path == results[j].Path {
			return results[i].LineNumber < results[j].LineNumber
		}
		return results[i].Path < results[j].Path
	})
	return results, nil
}

func compileQuery(query string, caseSensitive bool) (*regexp.Regexp, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrQueryRequired
	}
	if !caseSensitive {
		query = "(?i)" + query
	}
	return regexp.Compile(query)
}

func normalizeFileTypes(types []string) map[string]struct{} {
	if len(types) == 0 {
		types = defaultFileTypes
	}
	set := make(map[string]struct{}, len(types))
	for _, t := range types {
		t = strings.TrimSpace(strings.ToLower(strings.TrimPrefix(t, ".")))
		if t == "" {
			continue
		}
		set[t] = struct{}{}
	}
	return set
}

func matchesFileType(path string, types map[string]struct{}) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		return false
	}
	_, ok := types[ext]
	return ok
}

func searchFile(path string, matcher *regexp.Regexp, contextLines int) ([]TextSearchMatch, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	matches := make([]TextSearchMatch, 0)
	for i, line := range lines {
		if !matcher.MatchString(line) {
			continue
		}
		match := TextSearchMatch{
			LineNumber: i + 1,
			Content:    truncateLine(strings.TrimRight(line, "\r"), maxLineLength),
		}
		if contextLines > 0 {
			beforeStart := i - contextLines
			if beforeStart < 0 {
				beforeStart = 0
			}
			afterEnd := i + 1 + contextLines
			if afterEnd > len(lines) {
				afterEnd = len(lines)
			}
			if beforeStart < i {
				match.ContextBefore = truncateLines(lines[beforeStart:i], maxLineLength)
			}
			if i+1 < afterEnd {
				match.ContextAfter = truncateLines(lines[i+1:afterEnd], maxLineLength)
			}
		}
		matches = append(matches, match)
	}
	return matches, nil
}

func truncateLine(line string, max int) string {
	if max <= 0 {
		return line
	}
	if len(line) <= max {
		return line
	}
	if max <= 3 {
		return line[:max]
	}
	return line[:max-3] + "..."
}

func truncateLines(lines []string, max int) string {
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed = append(trimmed, truncateLine(strings.TrimRight(line, "\r"), max))
	}
	return strings.Join(trimmed, "\n")
}
