package conversationsearch

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
)

// conversationDenseStrongThreshold is deliberately domain-specific. Conversation
// embeddings below this cosine similarity are useful as recall candidates, but
// live calibration showed that unrelated prose also occupies this range. Keep the
// candidates for transparent ranking evidence while marking them weak so callers
// can distinguish plausible recall from background nearest neighbors.
const conversationDenseStrongThreshold = 0.70

type sharedSearchEngine interface {
	Search(context.Context, aisearch.SearchQuery, ...aisearch.SearchOption) (aisearch.SearchResponse, error)
}

type SharedSemanticRetriever struct {
	engine       sharedSearchEngine
	projection   ProjectionRepository
	admission    aisearch.Admission
	searchWeight int64
}

func NewSharedSemanticRetriever(engine sharedSearchEngine, projection ProjectionRepository, admission aisearch.Admission) (*SharedSemanticRetriever, error) {
	if engine == nil || projection == nil || admission == nil {
		return nil, fmt.Errorf("shared semantic engine, projection, and admission are required")
	}
	return &SharedSemanticRetriever{engine: engine, projection: projection, admission: admission, searchWeight: 1}, nil
}

func (r *SharedSemanticRetriever) SearchSemantic(ctx context.Context, request SemanticSearchRequest) ([]SemanticCandidate, error) {
	if len(request.Filters.ContentClasses) == 0 {
		request.Filters.ContentClasses = []ContentClass{ContentClassProse, ContentClassQuotedProse}
	}
	release, err := r.admission.Acquire(ctx, r.searchWeight)
	if err != nil {
		return nil, err
	}
	defer release()
	response, err := r.engine.Search(ctx, aisearch.SearchQuery{
		// The conversation service owns the outer lexical+semantic fusion. Keep
		// this inner leg dense so its cosine score remains calibrated for weak-hit
		// labeling; feeding a rank-normalized RRF score into another fusion makes
		// the top nearest neighbor look strong even for gibberish.
		Query: request.Query, Mode: aisearch.ModeDense, Limit: request.Limit, Filter: semanticQueryFilter(request.Filters),
	})
	if err != nil {
		return nil, classifySemanticError(err)
	}
	output := make([]SemanticCandidate, 0, len(response.Results))
	for index, result := range response.Results {
		documentID := strings.TrimSpace(result.SourceID)
		if documentID == "" {
			documentID, _ = result.Payload["source_id"].(string)
		}
		if documentID == "" {
			documentID, _ = result.Payload["document_id"].(string)
		}
		if documentID == "" {
			continue
		}
		document, getErr := r.projection.GetDocument(ctx, documentID)
		if errors.Is(getErr, ErrNotFound) {
			continue
		}
		if getErr != nil {
			return nil, fmt.Errorf("project semantic document %q: %w", documentID, getErr)
		}
		if !document.Visible || !matchesSearchFilters(document, request.Filters) {
			continue
		}
		evidence := semanticRankEvidence(response, index+1, result.Score)
		output = append(output, SemanticCandidate{Document: document, Score: result.Score, Rank: index + 1, Evidence: evidence, Weak: result.Weak || result.Score < conversationDenseStrongThreshold})
	}
	return output, nil
}

func semanticRankEvidence(response aisearch.SearchResponse, rank int, score float64) []RankEvidence {
	var evidence []RankEvidence
	switch response.Method {
	case "hybrid":
		evidence = append(evidence,
			RankEvidence{Leg: SearchLegDense, Rank: rank, Score: score, Explanation: "Qdrant dense contribution to server-side fusion"},
			RankEvidence{Leg: SearchLegSparse, Rank: rank, Score: score, Explanation: "BM25 sparse contribution to server-side fusion"},
		)
	default:
		evidence = append(evidence, RankEvidence{Leg: SearchLegDense, Rank: rank, Score: score, Explanation: "dense semantic retrieval rank"})
	}
	if response.Reranker != "" && response.Reranker != "none" {
		evidence = append(evidence, RankEvidence{Leg: SearchLegRerank, Rank: rank, Score: score, Explanation: "reranked by " + response.Reranker})
	}
	return evidence
}

func semanticQueryFilter(filters SearchFilters) *aisearch.QueryFilter {
	filter := &aisearch.QueryFilter{}
	appendAny := func(key string, values []string) {
		values = compactStrings(values)
		if len(values) == 0 {
			return
		}
		anyOf := make([]any, 0, len(values))
		for _, value := range values {
			anyOf = append(anyOf, value)
		}
		filter.Must = append(filter.Must, aisearch.FieldMatch{Key: key, AnyOf: anyOf})
	}
	appendAny("role", filters.Roles)
	appendAny("harness", filters.Harnesses)
	appendAny("provider_origin", filters.ProviderOrigins)
	appendAny("project_scope", filters.ProjectScopes)
	appendAny("cwd_scope", filters.CWDScopes)
	appendAny("runner", filters.Runners)
	appendAny("model", filters.Models)
	appendAny("profile", filters.Profiles)
	appendAny("run_status", filters.RunStatuses)
	appendAny("tags", filters.Tags)
	appendAny("workloads", filters.Workloads)
	classes := filters.ContentClasses
	if len(classes) == 0 {
		classes = []ContentClass{ContentClassProse, ContentClassQuotedProse}
	}
	if len(classes) > 0 {
		values := make([]any, 0, len(classes))
		for _, class := range classes {
			values = append(values, int(class))
		}
		filter.Must = append(filter.Must, aisearch.FieldMatch{Key: "content_class", AnyOf: values})
	}
	if filters.OccurredAfter != nil || filters.OccurredBefore != nil {
		rangeFilter := aisearch.FieldRange{Key: "occurred_at_unix"}
		if filters.OccurredAfter != nil {
			value := float64(filters.OccurredAfter.Unix())
			rangeFilter.GTE = &value
		}
		if filters.OccurredBefore != nil {
			value := float64(filters.OccurredBefore.Unix())
			rangeFilter.LTE = &value
		}
		filter.Ranges = append(filter.Ranges, rangeFilter)
	}
	if len(filter.Must) == 0 && len(filter.Ranges) == 0 {
		return nil
	}
	return filter
}

func matchesSearchFilters(document Document, filters SearchFilters) bool {
	if filters.OccurredAfter != nil && document.OccurredAt.Before(*filters.OccurredAfter) {
		return false
	}
	if filters.OccurredBefore != nil && document.OccurredAt.After(*filters.OccurredBefore) {
		return false
	}
	contains := func(values []string, value string) bool {
		if len(values) == 0 {
			return true
		}
		for _, candidate := range values {
			if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(value)) {
				return true
			}
		}
		return false
	}
	intersects := func(values, actual []string) bool {
		if len(values) == 0 {
			return true
		}
		for _, value := range values {
			if contains(actual, value) {
				return true
			}
		}
		return false
	}
	if !contains(filters.Roles, document.Role) || !contains(filters.Harnesses, document.Harness) ||
		!contains(filters.ProviderOrigins, document.ProviderOrigin) || !contains(filters.ProjectScopes, document.ProjectScope) ||
		!contains(filters.CWDScopes, document.CWDScope) || !contains(filters.Runners, document.Runner) ||
		!contains(filters.Models, document.Model) || !contains(filters.Profiles, document.Profile) ||
		!contains(filters.RunStatuses, document.RunStatus) || !intersects(filters.Tags, document.Tags) || !intersects(filters.Workloads, document.Workloads) {
		return false
	}
	if len(filters.ContentClasses) > 0 {
		matched := false
		for _, class := range filters.ContentClasses {
			matched = matched || class == document.ContentClass
		}
		if !matched {
			return false
		}
	}
	return true
}

func classifySemanticError(err error) error {
	lower := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case strings.Contains(lower, "embed") || strings.Contains(lower, "ollama"):
		return errors.Join(ErrEmbeddingUnavailable, err)
	case strings.Contains(lower, "layout") || strings.Contains(lower, "dimension") || strings.Contains(lower, "sentinel"):
		return errors.Join(ErrIndexLayoutMismatch, err)
	case strings.Contains(lower, "qdrant") || strings.Contains(lower, "vector") || strings.Contains(lower, "connection refused"):
		return errors.Join(ErrVectorStoreUnavailable, err)
	case strings.Contains(lower, "rerank"):
		return errors.Join(ErrRerankUnavailable, err)
	default:
		return errors.Join(ErrSemanticUnavailable, err)
	}
}

var _ SemanticRetriever = (*SharedSemanticRetriever)(nil)
