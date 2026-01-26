package docsearch

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"knowledge-observatory/internal/services/search"
)

// SemanticSearcher exposes the semantic search service dependency.
type SemanticSearcher interface {
	Search(ctx context.Context, req search.Request) (search.Response, error)
}

const (
	unifiedSourceFile     = "file"
	unifiedSourceText     = "text"
	unifiedSourceSemantic = "semantic"
)

// SearchUnified combines file, text, and semantic search results.
func (s *Service) SearchUnified(ctx context.Context, req UnifiedSearchRequest) (*UnifiedSearchResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("doc search service unavailable")
	}
	start := time.Now()
	if err := req.normalize(); err != nil {
		return nil, err
	}

	results := make([]UnifiedSearchResult, 0)
	pattern := strings.TrimSpace(req.Pattern)
	if pattern == "" {
		pattern = inferPattern(req.Query)
	}
	if pattern != "" {
		fileResults, err := s.SearchFiles(ctx, FileSearchRequest{
			Pattern:        pattern,
			Scope:          req.Scope,
			Scenario:       req.Scenario,
			BasePath:       req.BasePath,
			Limit:          req.Limit,
			IncludeContent: req.IncludeContent,
		})
		if err == nil {
			for _, res := range fileResults {
				results = append(results, UnifiedSearchResult{
					Source:       unifiedSourceFile,
					Score:        scoreFileResult(res, pattern),
					Path:         res.Path,
					RelativePath: res.RelativePath,
					Scenario:     res.Scenario,
					Snippet:      res.ContentPreview,
					DocType:      res.DocType,
				})
			}
		}
	}

	if strings.TrimSpace(req.Query) != "" {
		textResults, err := s.SearchText(ctx, TextSearchRequest{
			Query:         req.Query,
			Scope:         req.Scope,
			Scenario:      req.Scenario,
			BasePath:      req.BasePath,
			FileTypes:     req.FileTypes,
			CaseSensitive: req.CaseSensitive,
			Limit:         req.Limit,
			ContextLines:  req.ContextLines,
		})
		if err == nil {
			for _, res := range textResults {
				results = append(results, UnifiedSearchResult{
					Source:       unifiedSourceText,
					Score:        scoreTextResult(res),
					Path:         res.Path,
					RelativePath: res.RelativePath,
					Scenario:     res.Scenario,
					LineNumber:   res.LineNumber,
					Snippet:      res.Content,
					Content:      buildContextSnippet(res),
				})
			}
		}
	}

	if shouldUseSemantic(req.UseSemantic, s.Semantic != nil) && strings.TrimSpace(req.Query) != "" {
		limit := req.SemanticLimit
		if limit <= 0 {
			limit = req.Limit
		}
		threshold := req.SemanticThreshold
		if threshold <= 0 {
			threshold = 0.3
		}
		semanticResults, err := s.Semantic.Search(ctx, search.Request{
			Query:      req.Query,
			Collection: req.SemanticCollection,
			Namespaces: req.SemanticNamespaces,
			Visibility: req.SemanticVisibility,
			Tags:       req.SemanticTags,
			Limit:      limit,
			Threshold:  threshold,
		})
		if err == nil {
			for _, res := range semanticResults.Results {
				path, scenario := extractSemanticPath(res.Metadata)
				results = append(results, UnifiedSearchResult{
					Source:   unifiedSourceSemantic,
					Score:    res.Score,
					Path:     path,
					Scenario: scenario,
					ID:       res.ID,
					Content:  res.Content,
					Metadata: res.Metadata,
				})
			}
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Source < results[j].Source
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return &UnifiedSearchResponse{
		Results: results,
		Query:   req.Query,
		TookMS:  time.Since(start).Milliseconds(),
	}, nil
}

func inferPattern(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	if strings.ContainsAny(query, "*?[]") || strings.Contains(query, "/") || strings.Contains(query, "\\") {
		return query
	}
	if strings.Contains(query, ".") {
		return "**/*" + query + "*"
	}
	return ""
}

func scoreFileResult(result FileSearchResult, pattern string) float64 {
	base := strings.ToLower(strings.TrimSpace(filepathBase(result.RelativePath)))
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == base || strings.Trim(pattern, "*") == base {
		return 0.9
	}
	if strings.Contains(base, strings.Trim(pattern, "*")) {
		return 0.7
	}
	return 0.6
}

func scoreTextResult(result TextSearchMatch) float64 {
	score := 0.65
	if result.LineNumber > 0 && result.LineNumber <= 5 {
		score += 0.1
	}
	return score
}

func buildContextSnippet(match TextSearchMatch) string {
	parts := make([]string, 0, 3)
	if match.ContextBefore != "" {
		parts = append(parts, match.ContextBefore)
	}
	if match.Content != "" {
		parts = append(parts, match.Content)
	}
	if match.ContextAfter != "" {
		parts = append(parts, match.ContextAfter)
	}
	return strings.Join(parts, "\n")
}

func extractSemanticPath(metadata map[string]interface{}) (string, string) {
	if metadata == nil {
		return "", ""
	}
	pathKeys := []string{"path", "source_path", "file_path", "doc_path"}
	for _, key := range pathKeys {
		if value, ok := metadata[key]; ok {
			if asString, ok := value.(string); ok {
				return asString, extractScenario(metadata)
			}
		}
	}
	return "", extractScenario(metadata)
}

func extractScenario(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}
	keys := []string{"scenario", "source_scenario"}
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			if asString, ok := value.(string); ok {
				return asString
			}
		}
	}
	return ""
}

func filepathBase(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[idx+1:]
	}
	return path
}
