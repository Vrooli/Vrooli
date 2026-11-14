package docsearch

import (
	"errors"
	"strings"
	"time"
)

const (
	ScopeGlobal   = "global"
	ScopeScenario = "scenario"
	ScopePath     = "path"
)

const (
	defaultFileSearchLimit  = 50
	maxFileSearchLimit      = 500
	defaultTextSearchLimit  = 50
	maxTextSearchLimit      = 500
	defaultUnifiedLimit     = 50
	maxUnifiedLimit         = 200
	defaultContextLineCount = 0
)

var (
	ErrPatternRequired   = errors.New("pattern is required")
	ErrQueryRequired     = errors.New("query is required")
	ErrScopeInvalid      = errors.New("scope must be global, scenario, or path")
	ErrScenarioRequired  = errors.New("scenario is required for scenario scope")
	ErrBasePathRequired  = errors.New("base_path is required for path scope")
	ErrBasePathInvalid   = errors.New("base_path must be within repo root")
	ErrScenarioRootEmpty = errors.New("scenarios root is not configured")
)

// FileSearchRequest defines glob-based file search input.
type FileSearchRequest struct {
	Pattern        string
	Scope          string
	Scenario       string
	BasePath       string
	Limit          int
	IncludeContent bool
}

// FileSearchResult describes a matched documentation file.
type FileSearchResult struct {
	Path           string
	RelativePath   string
	Scenario       string
	Size           int64
	ModifiedAt     time.Time
	DocType        string
	ContentPreview string
}

// TextSearchRequest defines full-text search input.
type TextSearchRequest struct {
	Query         string
	Scope         string
	Scenario      string
	BasePath      string
	FileTypes     []string
	CaseSensitive bool
	Limit         int
	ContextLines  int
}

// TextSearchMatch describes a text search match with optional context.
type TextSearchMatch struct {
	Path          string
	RelativePath  string
	Scenario      string
	LineNumber    int
	Content       string
	ContextBefore string
	ContextAfter  string
}

// UnifiedSearchRequest combines file, text, and semantic search.
// Note: the records-era semantic filter fields (SemanticCollection,
// SemanticNamespaces, SemanticVisibility, SemanticTags) were removed in the
// Phase-7 cutover; the hybrid engine does not use them.
type UnifiedSearchRequest struct {
	Query             string
	Pattern           string
	Scope             string
	Scenario          string
	BasePath          string
	Limit             int
	IncludeContent    bool
	FileTypes         []string
	CaseSensitive     bool
	ContextLines      int
	UseSemantic       *bool
	SemanticLimit     int
	SemanticThreshold float64
}

// UnifiedSearchResult represents a normalized match from any search source.
type UnifiedSearchResult struct {
	Source       string
	Score        float64
	Path         string
	RelativePath string
	Scenario     string
	LineNumber   int
	Snippet      string
	Content      string
	DocType      string
	ID           string
	Metadata     map[string]interface{}
}

// UnifiedSearchResponse wraps unified results with timing.
type UnifiedSearchResponse struct {
	Results []UnifiedSearchResult
	Query   string
	TookMS  int64
}

func normalizeScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return ScopeGlobal
	}
	return scope
}

func (r *FileSearchRequest) normalize() error {
	r.Scope = normalizeScope(r.Scope)
	r.Pattern = strings.TrimSpace(r.Pattern)
	r.Scenario = strings.TrimSpace(r.Scenario)
	r.BasePath = strings.TrimSpace(r.BasePath)
	if r.Pattern == "" {
		return ErrPatternRequired
	}
	switch r.Scope {
	case ScopeGlobal:
		// no-op
	case ScopeScenario:
		if r.Scenario == "" {
			return ErrScenarioRequired
		}
	case ScopePath:
		if r.BasePath == "" {
			return ErrBasePathRequired
		}
	default:
		return ErrScopeInvalid
	}
	if r.Limit <= 0 {
		r.Limit = defaultFileSearchLimit
	}
	if r.Limit > maxFileSearchLimit {
		r.Limit = maxFileSearchLimit
	}
	return nil
}

func (r *TextSearchRequest) normalize() error {
	r.Scope = normalizeScope(r.Scope)
	r.Query = strings.TrimSpace(r.Query)
	r.Scenario = strings.TrimSpace(r.Scenario)
	r.BasePath = strings.TrimSpace(r.BasePath)
	if r.Query == "" {
		return ErrQueryRequired
	}
	switch r.Scope {
	case ScopeGlobal:
	case ScopeScenario:
		if r.Scenario == "" {
			return ErrScenarioRequired
		}
	case ScopePath:
		if r.BasePath == "" {
			return ErrBasePathRequired
		}
	default:
		return ErrScopeInvalid
	}
	if r.Limit <= 0 {
		r.Limit = defaultTextSearchLimit
	}
	if r.Limit > maxTextSearchLimit {
		r.Limit = maxTextSearchLimit
	}
	if r.ContextLines < 0 {
		r.ContextLines = defaultContextLineCount
	}
	return nil
}

func (r *UnifiedSearchRequest) normalize() error {
	r.Scope = normalizeScope(r.Scope)
	r.Query = strings.TrimSpace(r.Query)
	r.Pattern = strings.TrimSpace(r.Pattern)
	r.Scenario = strings.TrimSpace(r.Scenario)
	r.BasePath = strings.TrimSpace(r.BasePath)
	if r.Query == "" && r.Pattern == "" {
		return ErrQueryRequired
	}
	switch r.Scope {
	case ScopeGlobal:
	case ScopeScenario:
		if r.Scenario == "" {
			return ErrScenarioRequired
		}
	case ScopePath:
		if r.BasePath == "" {
			return ErrBasePathRequired
		}
	default:
		return ErrScopeInvalid
	}
	if r.Limit <= 0 {
		r.Limit = defaultUnifiedLimit
	}
	if r.Limit > maxUnifiedLimit {
		r.Limit = maxUnifiedLimit
	}
	if r.ContextLines < 0 {
		r.ContextLines = defaultContextLineCount
	}
	return nil
}

func shouldUseSemantic(flag *bool, available bool) bool {
	if !available {
		return false
	}
	if flag == nil {
		return true
	}
	return *flag
}
