package conversationsearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

const (
	defaultSearchPageSize = 25
	maximumSearchPageSize = 100
	maximumSnippetBytes   = 2048
)

var ErrInvalidRequest = errors.New("invalid conversation search request")

func invalidRequest(message string) error {
	return errors.Join(ErrInvalidRequest, errors.New(message))
}

type SearchFilters struct {
	OccurredAfter   *time.Time
	OccurredBefore  *time.Time
	Roles           []string
	Harnesses       []string
	ProviderOrigins []string
	ProjectScopes   []string
	CWDScopes       []string
	Runners         []string
	Models          []string
	Profiles        []string
	RunStatuses     []string
	Tags            []string
	Workloads       []string
	ContentClasses  []ContentClass
}

type TextSearchRequest struct {
	Query           string
	Sort            SearchSort
	Filters         SearchFilters
	PageSize        int
	Cursor          string
	fingerprintMode string
	matchAllTerms   bool
}

type SearchLeg int

const (
	SearchLegLexical SearchLeg = iota + 1
	SearchLegRegex
	SearchLegDense
	SearchLegSparse
	SearchLegRerank
)

type RankEvidence struct {
	Leg         SearchLeg
	Rank        int
	Score       float64
	Explanation string
}

type DegradationReason string

const (
	DegradationSemanticUnavailable    DegradationReason = "semantic_unavailable"
	DegradationEmbeddingUnavailable   DegradationReason = "embedding_unavailable"
	DegradationVectorStoreUnavailable DegradationReason = "vector_store_unavailable"
	DegradationIndexLayoutMismatch    DegradationReason = "index_layout_mismatch"
	DegradationRerankUnavailable      DegradationReason = "rerank_unavailable"
	DegradationDeadline               DegradationReason = "deadline"
)

type Degradation struct {
	Reason    DegradationReason
	Leg       SearchLeg
	Detail    string
	Retryable bool
}

type Highlight struct {
	StartRune int
	EndRune   int
}

type SearchHit struct {
	Document   Document
	Snippet    string
	Highlights []Highlight
	Score      float64
	Rank       int
	Leg        SearchLeg
	Evidence   []RankEvidence
	DeepLink   string
	Weak       bool
}

type TextSearchResponse struct {
	Hits                     []SearchHit
	NextCursor               string
	CanonicalVisibleMessages uint64
	CatalogDocuments         uint64
	LexicalDocuments         uint64
	PartialReason            RegexLimitReason
	ScannedCandidates        int
	ScannedBytes             int
	Degradations             []Degradation
}

type Service struct {
	candidates       CandidateRepository
	projection       ProjectionRepository
	status           StatusRepository
	regex            RegexCandidateRepository
	cursors          cursorCodec
	regexPolicy      RegexPolicy
	semantic         SemanticRetriever
	telemetry        TelemetryRepository
	telemetryKey     []byte
	telemetryAppends atomic.Uint64
}

type ServiceOption func(*Service)

func WithSemanticRetriever(retriever SemanticRetriever) ServiceOption {
	return func(service *Service) { service.semantic = retriever }
}

func NewService(candidates CandidateRepository, projection ProjectionRepository, status StatusRepository, cursorKey []byte, options ...ServiceOption) (*Service, error) {
	if candidates == nil || projection == nil || status == nil {
		return nil, errors.New("conversation search repositories are required")
	}
	codec, err := newCursorCodec(cursorKey)
	if err != nil {
		return nil, err
	}
	regex, _ := candidates.(RegexCandidateRepository)
	telemetry, _ := projection.(TelemetryRepository)
	telemetryDigest := sha256.Sum256(append(append([]byte(nil), cursorKey...), []byte("conversation-search-telemetry")...))
	service := &Service{candidates: candidates, projection: projection, status: status, regex: regex, cursors: codec, regexPolicy: DefaultRegexPolicy(), telemetry: telemetry, telemetryKey: telemetryDigest[:]}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service, nil
}

func (s *Service) SearchText(ctx context.Context, request TextSearchRequest) (TextSearchResponse, error) {
	request.Query = strings.TrimSpace(request.Query)
	request.fingerprintMode = "text"
	if request.Sort == 0 {
		request.Sort = SearchSortRelevance
	}
	if request.Query == "" {
		if request.Sort == SearchSortRelevance {
			return TextSearchResponse{}, invalidRequest("query may be empty only for newest or oldest browsing")
		}
		if !hasSearchFilters(request.Filters) {
			return TextSearchResponse{}, invalidRequest("at least one structured filter is required when query is empty")
		}
	}
	if request.PageSize == 0 {
		request.PageSize = defaultSearchPageSize
	}
	if request.PageSize < 1 || request.PageSize > maximumSearchPageSize {
		return TextSearchResponse{}, invalidRequest("page_size must be between 1 and 100")
	}
	fingerprint, err := searchFingerprint(request)
	if err != nil {
		return TextSearchResponse{}, err
	}
	var after *CandidateCursor
	if request.Cursor != "" {
		payload, err := s.cursors.decode(request.Cursor, fingerprint, request.Sort)
		if err != nil {
			return TextSearchResponse{}, errors.Join(ErrInvalidRequest, err)
		}
		after = &CandidateCursor{Score: payload.Score, OccurredAt: payload.OccurredAt, DocumentID: payload.DocumentID}
	}
	candidates, err := s.candidates.LexicalCandidates(ctx, CandidateQuery{
		Query: request.Query, MatchAllTerms: request.matchAllTerms, Limit: request.PageSize + 1, Sort: request.Sort, After: after,
		OccurredAfter: request.Filters.OccurredAfter, OccurredBefore: request.Filters.OccurredBefore,
		Roles: request.Filters.Roles, Harnesses: request.Filters.Harnesses,
		ProviderOrigins: request.Filters.ProviderOrigins, ProjectScopes: request.Filters.ProjectScopes,
		CWDScopes: request.Filters.CWDScopes, Runners: request.Filters.Runners, Models: request.Filters.Models,
		Profiles: request.Filters.Profiles, RunStatuses: request.Filters.RunStatuses,
		Tags: request.Filters.Tags, Workloads: request.Filters.Workloads, ContentClasses: request.Filters.ContentClasses,
	})
	if err != nil {
		return TextSearchResponse{}, err
	}
	response := TextSearchResponse{}
	if len(candidates) > request.PageSize {
		candidates = candidates[:request.PageSize]
		last := candidates[len(candidates)-1]
		response.NextCursor, err = s.cursors.encode(cursorPayload{Fingerprint: fingerprint, Sort: request.Sort, Score: last.Score, OccurredAt: last.Document.OccurredAt, DocumentID: last.Document.DocumentID})
		if err != nil {
			return TextSearchResponse{}, err
		}
	}
	for _, candidate := range candidates {
		snippet, highlights := boundedSnippet(candidate.Document.Content, request.Query, maximumSnippetBytes)
		response.Hits = append(response.Hits, SearchHit{Document: candidate.Document, Snippet: snippet, Highlights: highlights, Score: candidate.Score, Rank: candidate.Rank, Leg: SearchLegLexical, Evidence: []RankEvidence{{Leg: SearchLegLexical, Rank: candidate.Rank, Score: candidate.Score, Explanation: "SQLite FTS lexical rank"}}, DeepLink: "/runs/" + candidate.Document.SourceRunID + "?event=" + candidate.Document.SourceEventID, Weak: weakLexicalCoverage(candidate.Document.Content, request.Query)})
	}
	response.CanonicalVisibleMessages, response.CatalogDocuments, response.LexicalDocuments, err = s.status.CountCoverage(ctx)
	if err != nil {
		return TextSearchResponse{}, err
	}
	return response, nil
}

// weakLexicalCoverage distinguishes a real multi-term match from an OR-query
// hit caused by one generic word. BM25 remains raw rank evidence; confidence is
// derived independently so corpus-size-dependent score magnitudes are never
// mistaken for calibrated relevance.
func weakLexicalCoverage(content, query string) bool {
	terms := searchableTerms(query)
	if len(terms) < 3 {
		return false
	}
	lowerContent := strings.ToLower(content)
	matched := 0
	for _, term := range terms {
		if strings.Contains(lowerContent, term) {
			matched++
		}
	}
	return matched < 2 || matched*2 < len(terms)
}

func searchableTerms(value string) []string {
	seen := make(map[string]struct{})
	terms := make([]string, 0)
	for _, field := range strings.Fields(strings.ToLower(value)) {
		term := strings.Trim(field, `"'.,;:!?()[]{} `)
		if term == "" || isLexicalStopword(term) {
			continue
		}
		if _, exists := seen[term]; exists {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	return terms
}

func hasSearchFilters(filters SearchFilters) bool {
	return filters.OccurredAfter != nil || filters.OccurredBefore != nil ||
		len(filters.Roles) > 0 || len(filters.Harnesses) > 0 || len(filters.ProviderOrigins) > 0 ||
		len(filters.ProjectScopes) > 0 || len(filters.CWDScopes) > 0 || len(filters.Runners) > 0 ||
		len(filters.Models) > 0 || len(filters.Profiles) > 0 || len(filters.RunStatuses) > 0 ||
		len(filters.Tags) > 0 || len(filters.Workloads) > 0 || len(filters.ContentClasses) > 0
}

func (s *Service) Context(ctx context.Context, documentID string, before, after int) (Document, []Document, error) {
	hit, err := s.projection.GetDocument(ctx, documentID)
	if err != nil {
		return Document{}, nil, err
	}
	if !hit.Visible {
		return Document{}, nil, ErrNotFound
	}
	documents, err := s.projection.ContextDocuments(ctx, hit.SourceRunID, hit.EventSequence, before, after)
	if err != nil {
		return Document{}, nil, err
	}
	return hit, documents, nil
}

func (s *Service) Status(ctx context.Context) (visibleMessages, catalogDocuments, lexicalDocuments uint64, err error) {
	return s.status.CountCoverage(ctx)
}

func BoundedContext(content string, maximum int) (string, bool) {
	if maximum <= 0 || len(content) <= maximum {
		return content, false
	}
	end := maximum
	for end > 0 && !utf8.RuneStart(content[end]) {
		end--
	}
	return content[:end], true
}

func searchFingerprint(request TextSearchRequest) (string, error) {
	normalized := struct {
		Mode     string        `json:"mode"`
		Query    string        `json:"query"`
		Sort     SearchSort    `json:"sort"`
		PageSize int           `json:"page_size"`
		Filters  SearchFilters `json:"filters"`
	}{Mode: request.fingerprintMode, Query: strings.ToLower(strings.Join(strings.Fields(request.Query), " ")), Sort: request.Sort, PageSize: request.PageSize, Filters: normalizeSearchFilters(request.Filters)}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:]), nil
}

func normalizeSearchFilters(filters SearchFilters) SearchFilters {
	filters.Roles = sortedStrings(filters.Roles)
	filters.Harnesses = sortedStrings(filters.Harnesses)
	filters.ProviderOrigins = sortedStrings(filters.ProviderOrigins)
	filters.ProjectScopes = sortedStrings(filters.ProjectScopes)
	filters.CWDScopes = sortedStrings(filters.CWDScopes)
	filters.Runners = sortedStrings(filters.Runners)
	filters.Models = sortedStrings(filters.Models)
	filters.Profiles = sortedStrings(filters.Profiles)
	filters.RunStatuses = sortedStrings(filters.RunStatuses)
	filters.Tags = sortedStrings(filters.Tags)
	filters.Workloads = sortedStrings(filters.Workloads)
	filters.ContentClasses = append([]ContentClass(nil), filters.ContentClasses...)
	sort.Slice(filters.ContentClasses, func(i, j int) bool { return filters.ContentClasses[i] < filters.ContentClasses[j] })
	return filters
}

func sortedStrings(values []string) []string {
	values = compactStrings(values)
	sort.Strings(values)
	return values
}

func boundedSnippet(content, query string, maximum int) (string, []Highlight) {
	if len(content) <= maximum {
		return content, queryHighlights(content, query)
	}
	lowerContent := strings.ToLower(content)
	match := 0
	if terms := strings.Fields(query); len(terms) > 0 {
		match = strings.Index(lowerContent, strings.ToLower(terms[0]))
	}
	if match < 0 {
		match = 0
	}
	start := match - maximum/3
	if start < 0 {
		start = 0
	}
	end := start + maximum
	if end > len(content) {
		end = len(content)
		start = end - maximum
	}
	for start > 0 && !utf8.RuneStart(content[start]) {
		start--
	}
	for end < len(content) && !utf8.RuneStart(content[end]) {
		end--
	}
	snippet := content[start:end]
	return snippet, queryHighlights(snippet, query)
}

func queryHighlights(content, query string) []Highlight {
	lower := strings.ToLower(content)
	seen := make(map[string]struct{})
	var highlights []Highlight
	for _, term := range strings.Fields(strings.ToLower(query)) {
		term = strings.Trim(term, `"'.,;:!?()[]{} `)
		if term == "" {
			continue
		}
		if _, exists := seen[term]; exists {
			continue
		}
		seen[term] = struct{}{}
		if byteIndex := strings.Index(lower, term); byteIndex >= 0 {
			highlights = append(highlights, Highlight{StartRune: utf8.RuneCountInString(content[:byteIndex]), EndRune: utf8.RuneCountInString(content[:byteIndex+len(term)])})
		}
	}
	return highlights
}
