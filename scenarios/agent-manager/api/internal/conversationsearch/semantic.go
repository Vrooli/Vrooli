package conversationsearch

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

const semanticLegTimeout = 1200 * time.Millisecond

var (
	ErrSemanticUnavailable    = errors.New("semantic retrieval unavailable")
	ErrEmbeddingUnavailable   = errors.New("embedding unavailable")
	ErrVectorStoreUnavailable = errors.New("vector store unavailable")
	ErrIndexLayoutMismatch    = errors.New("semantic index layout mismatch")
	ErrRerankUnavailable      = errors.New("reranker unavailable")
)

type SemanticSearchRequest struct {
	Query   string
	Filters SearchFilters
	Limit   int
}

type SemanticCandidate struct {
	Document Document
	Score    float64
	Rank     int
	Evidence []RankEvidence
	Weak     bool
}

// SemanticRetriever is the narrow seam between the conversation domain and
// the shared dense+sparse search engine. Implementations must return stable
// DocumentIDs shared with the lexical projection.
type SemanticRetriever interface {
	SearchSemantic(context.Context, SemanticSearchRequest) ([]SemanticCandidate, error)
}

func (s *Service) SearchSemantic(ctx context.Context, request TextSearchRequest) (TextSearchResponse, error) {
	request.Query = strings.TrimSpace(request.Query)
	request.fingerprintMode = "semantic"
	if request.Query == "" {
		return TextSearchResponse{}, invalidRequest("semantic search requires a query")
	}
	if request.Sort == 0 {
		request.Sort = SearchSortRelevance
	}
	if request.Sort != SearchSortRelevance {
		return TextSearchResponse{}, invalidRequest("semantic search supports relevance sort only")
	}
	pageSize, fingerprint, after, err := s.prepareRelevanceRequest(request)
	if err != nil {
		return TextSearchResponse{}, err
	}
	if s.semantic == nil {
		return TextSearchResponse{}, ErrSemanticUnavailable
	}
	candidates, err := s.semantic.SearchSemantic(ctx, SemanticSearchRequest{Query: request.Query, Filters: request.Filters, Limit: maximumSearchPageSize})
	if err != nil {
		return TextSearchResponse{}, err
	}
	hits := semanticHits(candidates, request.Query)
	return s.pageRelevanceHits(ctx, hits, pageSize, fingerprint, after)
}

func (s *Service) SearchHybrid(ctx context.Context, request TextSearchRequest) (TextSearchResponse, error) {
	request.Query = strings.TrimSpace(request.Query)
	request.fingerprintMode = "hybrid"
	if request.Query == "" {
		// Filtered browsing has no semantic query; preserve the exact lexical
		// browse semantics without manufacturing an optional-leg degradation.
		return s.SearchText(ctx, request)
	}
	if request.Sort == 0 {
		request.Sort = SearchSortRelevance
	}
	if request.Sort != SearchSortRelevance {
		return TextSearchResponse{}, invalidRequest("hybrid search supports relevance sort only")
	}
	pageSize, fingerprint, after, err := s.prepareRelevanceRequest(request)
	if err != nil {
		return TextSearchResponse{}, err
	}

	lexical := &conversationLexicalLeg{service: s, request: request}
	var semantic aisearch.SemanticSearcher
	semanticLeg := &conversationSemanticLeg{service: s, request: request}
	if s.semantic != nil {
		semantic = semanticLeg
	}
	fusion, fusionErr := (aisearch.ConcurrentFusion{Lexical: lexical, Semantic: semantic}).Search(ctx, aisearch.SearchQuery{
		Query: request.Query, Mode: aisearch.ModeHybrid, Limit: maximumSearchPageSize,
	})
	if fusionErr != nil {
		return TextSearchResponse{}, fusionErr
	}

	hits := make([]SearchHit, 0, len(fusion.Results))
	for index, fused := range fusion.Results {
		hit, ok := fused.Result.Payload["conversation_hit"].(SearchHit)
		if !ok {
			continue
		}
		hit.Score = fused.Result.Score
		hit.Rank = index + 1
		hit.Evidence = nil
		for _, evidence := range fused.Evidence {
			if evidence.Leg == "semantic" {
				candidate, candidateOK := fused.Result.Payload["semantic_candidate"].(SemanticCandidate)
				if candidateOK && len(candidate.Evidence) > 0 {
					hit.Evidence = append(hit.Evidence, candidate.Evidence...)
					continue
				}
				hit.Evidence = append(hit.Evidence, RankEvidence{Leg: SearchLegDense, Rank: evidence.Rank, Score: evidence.Score, Explanation: "semantic retrieval rank"})
				continue
			}
			hit.Evidence = append(hit.Evidence, RankEvidence{Leg: SearchLegLexical, Rank: evidence.Rank, Score: evidence.Score, Explanation: "SQLite FTS lexical rank"})
		}
		hits = append(hits, hit)
	}
	response, err := s.pageRelevanceHits(ctx, hits, pageSize, fingerprint, after)
	if err != nil {
		return TextSearchResponse{}, err
	}
	for _, leg := range fusion.Degraded {
		if leg == "semantic" {
			response.Degradations = append(response.Degradations, degradationForError(semanticLeg.failure()))
		}
	}
	if s.semantic == nil {
		response.Degradations = append(response.Degradations, degradationForError(ErrSemanticUnavailable))
	}
	return response, nil
}

func (s *Service) prepareRelevanceRequest(request TextSearchRequest) (int, string, *CandidateCursor, error) {
	if request.PageSize == 0 {
		request.PageSize = defaultSearchPageSize
	}
	if request.PageSize < 1 || request.PageSize > maximumSearchPageSize {
		return 0, "", nil, invalidRequest("page_size must be between 1 and 100")
	}
	fingerprint, err := searchFingerprint(request)
	if err != nil {
		return 0, "", nil, err
	}
	var after *CandidateCursor
	if request.Cursor != "" {
		payload, decodeErr := s.cursors.decode(request.Cursor, fingerprint, SearchSortRelevance)
		if decodeErr != nil {
			return 0, "", nil, errors.Join(ErrInvalidRequest, decodeErr)
		}
		after = &CandidateCursor{Score: payload.Score, DocumentID: payload.DocumentID}
	}
	return request.PageSize, fingerprint, after, nil
}

func (s *Service) pageRelevanceHits(ctx context.Context, hits []SearchHit, pageSize int, fingerprint string, after *CandidateCursor) (TextSearchResponse, error) {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Document.DocumentID < hits[j].Document.DocumentID
		}
		return hits[i].Score > hits[j].Score
	})
	if after != nil {
		start := len(hits)
		for index, hit := range hits {
			if hit.Score < after.Score || (hit.Score == after.Score && hit.Document.DocumentID > after.DocumentID) {
				start = index
				break
			}
		}
		hits = hits[start:]
	}
	response := TextSearchResponse{}
	if len(hits) > pageSize {
		hits = hits[:pageSize]
		last := hits[len(hits)-1]
		cursor, err := s.cursors.encode(cursorPayload{Fingerprint: fingerprint, Sort: SearchSortRelevance, Score: last.Score, DocumentID: last.Document.DocumentID})
		if err != nil {
			return TextSearchResponse{}, err
		}
		response.NextCursor = cursor
	}
	response.Hits = hits
	var err error
	response.CanonicalVisibleMessages, response.CatalogDocuments, response.LexicalDocuments, err = s.status.CountCoverage(ctx)
	return response, err
}

func semanticHits(candidates []SemanticCandidate, query string) []SearchHit {
	byID := make(map[string]SemanticCandidate, len(candidates))
	for _, candidate := range candidates {
		if candidate.Document.DocumentID == "" || !candidate.Document.Visible {
			continue
		}
		if previous, ok := byID[candidate.Document.DocumentID]; !ok || candidate.Score > previous.Score {
			byID[candidate.Document.DocumentID] = candidate
		}
	}
	ordered := make([]SemanticCandidate, 0, len(byID))
	for _, candidate := range byID {
		ordered = append(ordered, candidate)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Score == ordered[j].Score {
			return ordered[i].Document.DocumentID < ordered[j].Document.DocumentID
		}
		return ordered[i].Score > ordered[j].Score
	})
	hits := make([]SearchHit, 0, len(ordered))
	for index, candidate := range ordered {
		rank := candidate.Rank
		if rank <= 0 {
			rank = index + 1
		}
		evidence := candidate.Evidence
		if len(evidence) == 0 {
			evidence = []RankEvidence{{Leg: SearchLegDense, Rank: rank, Score: candidate.Score, Explanation: "semantic retrieval rank"}}
		}
		snippet, highlights := boundedSnippet(candidate.Document.Content, query, maximumSnippetBytes)
		hits = append(hits, SearchHit{Document: candidate.Document, Snippet: snippet, Highlights: highlights, Score: candidate.Score, Rank: rank, Leg: evidence[0].Leg, Evidence: evidence, DeepLink: "/runs/" + candidate.Document.SourceRunID + "?event=" + candidate.Document.SourceEventID, Weak: candidate.Weak})
	}
	return hits
}

type conversationLexicalLeg struct {
	service *Service
	request TextSearchRequest
}

func (l *conversationLexicalLeg) SearchLexical(ctx context.Context, _ aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	request := l.request
	request.PageSize = maximumSearchPageSize
	request.Cursor = ""
	request.matchAllTerms = true
	response, err := l.service.SearchText(ctx, request)
	if err != nil {
		return nil, err
	}
	results := make([]aisearch.SearchResult, 0, len(response.Hits))
	for _, hit := range response.Hits {
		results = append(results, aisearch.SearchResult{ID: hit.Document.DocumentID, Score: hit.Score, SourceID: hit.Document.DocumentID, Payload: map[string]any{"conversation_hit": hit}})
	}
	return results, nil
}

type conversationSemanticLeg struct {
	service *Service
	request TextSearchRequest
	mu      sync.Mutex
	err     error
}

func (l *conversationSemanticLeg) SearchSemantic(ctx context.Context, _ aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	semanticCtx, cancel := context.WithTimeout(ctx, semanticLegTimeout)
	defer cancel()
	type semanticResult struct {
		candidates []SemanticCandidate
		err        error
	}
	completed := make(chan semanticResult, 1)
	go func() {
		candidates, err := l.service.semantic.SearchSemantic(semanticCtx, SemanticSearchRequest{Query: l.request.Query, Filters: l.request.Filters, Limit: maximumSearchPageSize})
		completed <- semanticResult{candidates: candidates, err: err}
	}()
	var candidates []SemanticCandidate
	var err error
	select {
	case result := <-completed:
		candidates, err = result.candidates, result.err
	case <-semanticCtx.Done():
		err = semanticCtx.Err()
	}
	l.mu.Lock()
	l.err = err
	l.mu.Unlock()
	if err != nil {
		return nil, err
	}
	hits := semanticHits(candidates, l.request.Query)
	results := make([]aisearch.SearchResult, 0, len(hits))
	for index, hit := range hits {
		candidate := candidates[0]
		for _, current := range candidates {
			if current.Document.DocumentID == hit.Document.DocumentID {
				candidate = current
				break
			}
		}
		candidate.Rank = index + 1
		results = append(results, aisearch.SearchResult{ID: hit.Document.DocumentID, Score: hit.Score, SourceID: hit.Document.DocumentID, Weak: hit.Weak, Payload: map[string]any{"conversation_hit": hit, "semantic_candidate": candidate}})
	}
	return results, nil
}

func (l *conversationSemanticLeg) failure() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.err
}

func degradationForError(err error) Degradation {
	degradation := Degradation{Reason: DegradationSemanticUnavailable, Leg: SearchLegDense, Detail: "semantic retrieval is unavailable; lexical results remain available", Retryable: true}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		degradation.Reason = DegradationDeadline
		degradation.Detail = "semantic retrieval exceeded its request budget; lexical results remain available"
	case errors.Is(err, ErrEmbeddingUnavailable):
		degradation.Reason = DegradationEmbeddingUnavailable
		degradation.Detail = "the resolved embedding resource is unavailable; lexical results remain available"
	case errors.Is(err, ErrVectorStoreUnavailable):
		degradation.Reason = DegradationVectorStoreUnavailable
		degradation.Detail = "the vector store is unavailable; lexical results remain available"
	case errors.Is(err, ErrIndexLayoutMismatch):
		degradation.Reason = DegradationIndexLayoutMismatch
		degradation.Detail = "the vector index layout does not match the resolved embedding binding; no collection was dropped"
		degradation.Retryable = false
	case errors.Is(err, ErrRerankUnavailable):
		degradation.Reason = DegradationRerankUnavailable
		degradation.Leg = SearchLegRerank
		degradation.Detail = "the reranker is unavailable; base retrieval results remain available"
	}
	if err != nil && !errors.Is(err, ErrSemanticUnavailable) {
		degradation.Detail = fmt.Sprintf("%s: %v", degradation.Detail, err)
	}
	return degradation
}
