package docsearch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"knowledge-observatory/internal/doctemplates"
)

const maxPreviewBytes = 400

// SearchFiles performs glob-based documentation file search.
func (s *Service) SearchFiles(ctx context.Context, req FileSearchRequest) ([]FileSearchResult, error) {
	if s == nil {
		return nil, fmt.Errorf("doc search service unavailable")
	}
	if err := req.normalize(); err != nil {
		return nil, err
	}
	matcher, err := compileGlob(req.Pattern)
	if err != nil {
		return nil, err
	}
	files, err := s.collectDocFiles(ctx, req.Scope, req.Scenario, req.BasePath)
	if err != nil {
		return nil, err
	}
	results := make([]FileSearchResult, 0, min(len(files), req.Limit))
	for _, file := range files {
		rel := s.relPath(file.RelBase, file.Path)
		if !matcher.MatchString(rel) {
			continue
		}
		info, err := os.Stat(file.Path)
		if err != nil {
			continue
		}
		result := FileSearchResult{
			Path:         s.repoRelative(file.Path),
			RelativePath: rel,
			Scenario:     file.Scenario,
			Size:         info.Size(),
			ModifiedAt:   info.ModTime(),
		}
		if docType := s.contractDocType(file.Scenario, rel); docType != "" {
			result.DocType = docType
		}
		if req.IncludeContent {
			result.ContentPreview = readPreview(file.Path)
		}
		results = append(results, result)
		if len(results) >= req.Limit {
			break
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].RelativePath < results[j].RelativePath
	})
	return results, nil
}

func (s *Service) contractDocType(scenario, rel string) string {
	if scenario == "" {
		return ""
	}
	scenarioPath, err := s.scenarioPath(scenario)
	if err != nil {
		return ""
	}
	resolved, err := doctemplates.NewResolverFromScenariosRoot(s.scenariosRoot).ResolveScenario(scenarioPath)
	if err != nil || resolved == nil || resolved.Contract == nil {
		return ""
	}
	doc, ok := resolved.Contract.ResolvePath(rel)
	if !ok {
		return ""
	}
	return doc.DocType
}

func readPreview(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(content) > maxPreviewBytes {
		content = content[:maxPreviewBytes]
	}
	return strings.TrimSpace(string(content))
}

func compileGlob(pattern string) (*regexp.Regexp, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, ErrPatternRequired
	}
	pattern = filepath.ToSlash(pattern)
	pattern = strings.TrimPrefix(pattern, "./")
	pattern = strings.TrimPrefix(pattern, "/")
	if !strings.Contains(pattern, "/") {
		pattern = "**/" + pattern
	}
	var b strings.Builder
	b.WriteString("^")
	escape := func(ch byte) {
		switch ch {
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
		}
		b.WriteByte(ch)
	}
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString("(?:.*/)?")
					i += 2
					continue
				}
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '/':
			b.WriteByte('/')
		default:
			escape(ch)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
