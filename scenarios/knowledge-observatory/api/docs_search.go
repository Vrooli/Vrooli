package main

// DOC: docs/reference/api-endpoints.md#documentation-search
import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"knowledge-observatory/internal/services/docsearch"
)

// FileSearchRequest defines the input schema for glob-based search.
type FileSearchRequest struct {
	Pattern        string `json:"pattern"`
	Scope          string `json:"scope,omitempty"`
	Scenario       string `json:"scenario,omitempty"`
	BasePath       string `json:"base_path,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	IncludeContent bool   `json:"include_content,omitempty"`
}

// FileSearchResult represents a matched file response.
type FileSearchResult struct {
	Path           string    `json:"path"`
	RelativePath   string    `json:"relative_path"`
	Scenario       string    `json:"scenario,omitempty"`
	Size           int64     `json:"size"`
	ModifiedAt     time.Time `json:"modified_at"`
	DocType        string    `json:"doc_type,omitempty"`
	ContentPreview string    `json:"content_preview,omitempty"`
}

// TextSearchRequest defines the input schema for text search.
type TextSearchRequest struct {
	Query         string   `json:"query"`
	Scope         string   `json:"scope,omitempty"`
	Scenario      string   `json:"scenario,omitempty"`
	BasePath      string   `json:"base_path,omitempty"`
	FileTypes     []string `json:"file_types,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	ContextLines  int      `json:"context_lines,omitempty"`
}

// TextSearchMatch represents a matched line with optional context.
type TextSearchMatch struct {
	Path          string `json:"path"`
	RelativePath  string `json:"relative_path,omitempty"`
	Scenario      string `json:"scenario,omitempty"`
	LineNumber    int    `json:"line_number"`
	Content       string `json:"content"`
	ContextBefore string `json:"context_before,omitempty"`
	ContextAfter  string `json:"context_after,omitempty"`
}

// UnifiedSearchRequest combines multiple search modes.
type UnifiedSearchRequest struct {
	Query              string   `json:"query"`
	Pattern            string   `json:"pattern,omitempty"`
	Scope              string   `json:"scope,omitempty"`
	Scenario           string   `json:"scenario,omitempty"`
	BasePath           string   `json:"base_path,omitempty"`
	Limit              int      `json:"limit,omitempty"`
	IncludeContent     bool     `json:"include_content,omitempty"`
	FileTypes          []string `json:"file_types,omitempty"`
	CaseSensitive      bool     `json:"case_sensitive,omitempty"`
	ContextLines       int      `json:"context_lines,omitempty"`
	UseSemantic        *bool    `json:"use_semantic,omitempty"`
	SemanticCollection string   `json:"semantic_collection,omitempty"`
	SemanticNamespaces []string `json:"semantic_namespaces,omitempty"`
	SemanticVisibility []string `json:"semantic_visibility,omitempty"`
	SemanticTags       []string `json:"semantic_tags,omitempty"`
	SemanticLimit      int      `json:"semantic_limit,omitempty"`
	SemanticThreshold  float64  `json:"semantic_threshold,omitempty"`
}

// UnifiedSearchResult represents a normalized match.
type UnifiedSearchResult struct {
	Source       string                 `json:"source"`
	Score        float64                `json:"score"`
	Path         string                 `json:"path,omitempty"`
	RelativePath string                 `json:"relative_path,omitempty"`
	Scenario     string                 `json:"scenario,omitempty"`
	LineNumber   int                    `json:"line_number,omitempty"`
	Snippet      string                 `json:"snippet,omitempty"`
	Content      string                 `json:"content,omitempty"`
	DocType      string                 `json:"doc_type,omitempty"`
	ID           string                 `json:"id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UnifiedSearchResponse wraps unified results.
type UnifiedSearchResponse struct {
	Results []UnifiedSearchResult `json:"results"`
	Query   string                `json:"query"`
	TookMS  int64                 `json:"took_ms"`
}

// ScenarioSummary describes documentation status for a scenario.
type ScenarioSummary struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	DocCount     int       `json:"doc_count"`
	HealthScore  float64   `json:"health_score"`
	HasManifest  bool      `json:"has_manifest"`
	HasReadme    bool      `json:"has_readme"`
	LastModified time.Time `json:"last_modified"`
}

func (s *Server) handleDocsSearchFiles(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docSearchService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation search service unavailable")
		return
	}
	var req FileSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	results, err := s.docSearchService.SearchFiles(r.Context(), docsearch.FileSearchRequest{
		Pattern:        req.Pattern,
		Scope:          req.Scope,
		Scenario:       req.Scenario,
		BasePath:       req.BasePath,
		Limit:          req.Limit,
		IncludeContent: req.IncludeContent,
	})
	if err != nil {
		respondDocSearchError(w, err)
		return
	}
	payload := make([]FileSearchResult, 0, len(results))
	for _, res := range results {
		payload = append(payload, FileSearchResult{
			Path:           res.Path,
			RelativePath:   res.RelativePath,
			Scenario:       res.Scenario,
			Size:           res.Size,
			ModifiedAt:     res.ModifiedAt,
			DocType:        res.DocType,
			ContentPreview: res.ContentPreview,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleDocsSearchText(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docSearchService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation search service unavailable")
		return
	}
	var req TextSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	results, err := s.docSearchService.SearchText(r.Context(), docsearch.TextSearchRequest{
		Query:         req.Query,
		Scope:         req.Scope,
		Scenario:      req.Scenario,
		BasePath:      req.BasePath,
		FileTypes:     req.FileTypes,
		CaseSensitive: req.CaseSensitive,
		Limit:         req.Limit,
		ContextLines:  req.ContextLines,
	})
	if err != nil {
		respondDocSearchError(w, err)
		return
	}
	payload := make([]TextSearchMatch, 0, len(results))
	for _, res := range results {
		payload = append(payload, TextSearchMatch{
			Path:          res.Path,
			RelativePath:  res.RelativePath,
			Scenario:      res.Scenario,
			LineNumber:    res.LineNumber,
			Content:       res.Content,
			ContextBefore: res.ContextBefore,
			ContextAfter:  res.ContextAfter,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleDocsSearchUnified(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docSearchService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation search service unavailable")
		return
	}
	var req UnifiedSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	resp, err := s.docSearchService.SearchUnified(r.Context(), docsearch.UnifiedSearchRequest{
		Query:              req.Query,
		Pattern:            req.Pattern,
		Scope:              req.Scope,
		Scenario:           req.Scenario,
		BasePath:           req.BasePath,
		Limit:              req.Limit,
		IncludeContent:     req.IncludeContent,
		FileTypes:          req.FileTypes,
		CaseSensitive:      req.CaseSensitive,
		ContextLines:       req.ContextLines,
		UseSemantic:        req.UseSemantic,
		SemanticCollection: req.SemanticCollection,
		SemanticNamespaces: req.SemanticNamespaces,
		SemanticVisibility: req.SemanticVisibility,
		SemanticTags:       req.SemanticTags,
		SemanticLimit:      req.SemanticLimit,
		SemanticThreshold:  req.SemanticThreshold,
	})
	if err != nil {
		respondDocSearchError(w, err)
		return
	}
	payload := UnifiedSearchResponse{
		Query:  resp.Query,
		TookMS: resp.TookMS,
	}
	payload.Results = make([]UnifiedSearchResult, 0, len(resp.Results))
	for _, res := range resp.Results {
		payload.Results = append(payload.Results, UnifiedSearchResult{
			Source:       res.Source,
			Score:        res.Score,
			Path:         res.Path,
			RelativePath: res.RelativePath,
			Scenario:     res.Scenario,
			LineNumber:   res.LineNumber,
			Snippet:      res.Snippet,
			Content:      res.Content,
			DocType:      res.DocType,
			ID:           res.ID,
			Metadata:     res.Metadata,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docExplorerService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation explorer service unavailable")
		return
	}
	results, err := s.docExplorerService.ListScenarios(r.Context())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to list scenarios")
		return
	}
	payload := make([]ScenarioSummary, 0, len(results))
	repoRoot := ""
	if s.config != nil && s.config.ScenariosRoot != "" {
		repoRoot = filepath.Dir(s.config.ScenariosRoot)
	}
	for _, res := range results {
		path := res.Path
		if repoRoot != "" {
			if rel, err := filepath.Rel(repoRoot, res.Path); err == nil && !strings.HasPrefix(rel, "..") {
				path = filepath.ToSlash(rel)
			}
		}
		payload = append(payload, ScenarioSummary{
			Name:         res.Name,
			Path:         path,
			DocCount:     res.DocCount,
			HealthScore:  res.HealthScore,
			HasManifest:  res.HasManifest,
			HasReadme:    res.HasReadme,
			LastModified: res.LastModified,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func respondDocSearchError(w http.ResponseWriter, err error) {
	switch {
	case err == docsearch.ErrPatternRequired,
		err == docsearch.ErrQueryRequired,
		err == docsearch.ErrScopeInvalid,
		err == docsearch.ErrScenarioRequired,
		err == docsearch.ErrBasePathRequired,
		err == docsearch.ErrBasePathInvalid:
		respondWithError(w, http.StatusBadRequest, err)
	case err == docsearch.ErrScenarioRootEmpty:
		respondWithError(w, http.StatusServiceUnavailable, err)
	default:
		respondWithError(w, http.StatusInternalServerError, err)
	}
}
