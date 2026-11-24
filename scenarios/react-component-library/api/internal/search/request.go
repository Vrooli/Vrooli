package search

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
)

const (
	defaultSearchLimit  = 20
	maxSearchLimit      = 100
	defaultSuggestLimit = 8
	maxSuggestLimit     = 25
)

// Search answers one ranked query. It applies the requested kind filter before
// the limit and reports the pre-limit match count, so the UI can say "showing
// 20 of 137" honestly.
func (s *Service) Search(ctx context.Context, req *searchv1.SearchRequest) (*searchv1.SearchResponse, error) {
	started := time.Now()
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	if req == nil {
		req = &searchv1.SearchRequest{}
	}
	if req.GetAsk() && (strings.TrimSpace(req.GetQuery()) == "" || len(req.GetQuery()) > maxAskQuery) {
		return nil, fmt.Errorf("%w: question must be between 1 and %d characters", ErrAskInvalidQuery, maxAskQuery)
	}
	plan := planQuery(req.GetQuery())
	kinds := requestedKinds(req.GetKinds())

	var candidates []candidate
	var err error
	browse := plan.empty()
	if browse {
		candidates, err = s.browse(ctx)
	} else {
		candidates, err = s.match(ctx, plan, kinds)
		candidates = prioritizeExactPageType(candidates, plan)
		if err == nil {
			candidates, err = s.expandRequiredCatalogAssets(ctx, candidates)
		}
	}
	if err != nil {
		return nil, err
	}
	// Rank an exact single-token asset match ahead of semantic neighbors. For a
	// multiword query, let a stronger matching workspace beat an exact pattern
	// name: users are usually looking for an outcome unless they search a
	// component by its direct name.
	if !browse && len(kinds) == 0 {
		if len(plan.words) == 1 {
			hasExactWorkspace := false
			for i := range candidates {
				if KindText(candidates[i].doc.Kind) == kindWorkspace && identifierTitle(candidates[i].doc.Title) == plan.normalized {
					hasExactWorkspace = true
					break
				}
			}
			for i := range candidates {
				if !hasExactWorkspace && KindText(candidates[i].doc.Kind) == kindAsset && identifierTitle(candidates[i].doc.Title) == plan.normalized {
					candidates[i].intentRank = 1
				}
			}
		} else {
			exactAssetScore := -1.0
			for i := range candidates {
				if KindText(candidates[i].doc.Kind) == kindAsset && identifierTitle(candidates[i].doc.Title) == plan.normalized && candidates[i].score > exactAssetScore {
					exactAssetScore = candidates[i].score
				}
			}
			if exactAssetScore >= 0 {
				for i := range candidates {
					if KindText(candidates[i].doc.Kind) == kindWorkspace && candidates[i].score > exactAssetScore {
						candidates[i].intentRank = 1
					}
				}
			}
		}
	}
	candidates = dedupeCandidates(candidates)
	if len(kinds) > 0 {
		candidates = filterKinds(candidates, kinds)
	} else if browse {
		// The curated browse set is the five top-level destinations. Displays,
		// states and evidence are addressable through their parent page or the
		// Review queue, so they are omitted unless a caller explicitly filters
		// for them.
		candidates = filterKinds(candidates, defaultBrowseKinds)
	}

	total := len(candidates)
	facets := buildFacets(candidates)
	if browse {
		sortBrowse(candidates)
	} else {
		sortRanked(candidates)
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	if req.GetAsk() && len(candidates) > maxAskEvidence {
		candidates = candidates[:maxAskEvidence]
	}

	status, err := s.Status(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*searchv1.SearchResult, 0, len(candidates))
	for _, cand := range candidates {
		results = append(results, &searchv1.SearchResult{
			Id:           cand.doc.ID,
			Kind:         cand.doc.Kind,
			Title:        cand.doc.Title,
			Subtitle:     cand.doc.Subtitle,
			Snippet:      snippet(cand.doc.Body, plan.words, 160),
			Href:         cand.doc.Href,
			Score:        cand.score,
			Availability: cand.doc.Availability,
			Badges:       cand.doc.Badges,
			Relations:    cand.doc.Relations,
		})
	}
	response := &searchv1.SearchResponse{
		Query:   req.GetQuery(),
		Results: results,
		Facets:  facets,
		Total:   int32(total),
		TookMs:  time.Since(started).Milliseconds(),
		Status:  status,
	}
	if req.GetAsk() {
		if s.answerer == nil {
			return nil, ErrAskUnavailable
		}
		if len(candidates) == 0 {
			response.Answer = &searchv1.AskAnswer{Text: "No indexed results matched this question."}
			return response, nil
		}
		evidence := make([]Evidence, 0, len(candidates))
		for _, candidate := range candidates {
			if len(candidate.doc.ID) > maxEvidenceID {
				return nil, fmt.Errorf("%w: indexed result identity exceeds the Ask evidence bound", ErrAskProviderUnavailable)
			}
			evidence = append(evidence, Evidence{
				ID: candidate.doc.ID, Kind: KindText(candidate.doc.Kind),
				Title: boundedEvidence(candidate.doc.Title, maxEvidenceTitle), Subtitle: boundedEvidence(candidate.doc.Subtitle, maxEvidenceSubtitle),
				Body: boundedEvidence(candidate.doc.Body, maxEvidencePerResult),
			})
		}
		answer, err := s.answerer.Answer(ctx, req.GetQuery(), evidence)
		if err != nil {
			return nil, fmt.Errorf("answer indexed search question: %w", err)
		}
		allowed := make(map[string]struct{}, len(evidence))
		for _, item := range evidence {
			allowed[item.ID] = struct{}{}
		}
		seen := make(map[string]struct{}, len(answer.Citations))
		citations := make([]*searchv1.AskCitation, 0, len(answer.Citations))
		for _, id := range answer.Citations {
			if _, ok := allowed[id]; !ok {
				return nil, fmt.Errorf("%w: answer cited an unsupplied result", ErrAskInvalidAnswer)
			}
			if _, duplicate := seen[id]; duplicate {
				return nil, fmt.Errorf("%w: answer repeated a citation", ErrAskInvalidAnswer)
			}
			seen[id] = struct{}{}
			citations = append(citations, &searchv1.AskCitation{ResultId: id})
		}
		answerText := strings.TrimSpace(answer.Text)
		if answerText == "" || len(answerText) > 8000 || len(answer.Citations) == 0 || isOpaqueResultIdentifier(answerText) {
			return nil, fmt.Errorf("%w: answer text or citations are missing or exceed bounds", ErrAskInvalidAnswer)
		}
		response.Answer = &searchv1.AskAnswer{Text: answerText, Citations: citations}
	}
	return response, nil
}

// isOpaqueResultIdentifier rejects a machine identity presented as a user-facing
// answer. Identifiers belong in the validated citation list, while answer text
// must explain the indexed result in ordinary language.
func isOpaqueResultIdentifier(answer string) bool {
	return !strings.ContainsAny(answer, " \t\r\n") && strings.Contains(answer, ":")
}

// Suggest completes from indexed titles and tags. It intentionally queries the
// projection rather than the query grammar: a completion is a known string from
// the corpus, not a new search expression.
func (s *Service) Suggest(ctx context.Context, req *searchv1.SuggestRequest) (*searchv1.SuggestResponse, error) {
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	if req == nil {
		req = &searchv1.SuggestRequest{}
	}
	prefix := strings.ToLower(strings.TrimSpace(req.GetQuery()))
	if prefix == "" {
		return &searchv1.SuggestResponse{}, nil
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultSuggestLimit
	}
	if limit > maxSuggestLimit {
		limit = maxSuggestLimit
	}
	rows, err := s.db.QueryContext(ctx, `SELECT title, kind, badges_json FROM search_docs`)
	if err != nil {
		if isMissingTable(err) {
			return nil, ErrIndexNotInitialized
		}
		return nil, fmt.Errorf("read search suggestions: %w", err)
	}
	defer rows.Close()

	type suggestion struct {
		text   string
		prefix bool
		hits   int
		kind   searchv1.SearchKind
	}
	seen := map[string]*suggestion{}
	add := func(text string, kind searchv1.SearchKind) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		lower := strings.ToLower(text)
		startsWith := strings.HasPrefix(lower, prefix)
		if !startsWith && !strings.Contains(lower, prefix) {
			return
		}
		key := lower
		item, ok := seen[key]
		if !ok {
			seen[key] = &suggestion{text: text, prefix: startsWith, hits: 1, kind: kind}
			return
		}
		item.hits++
		if startsWith {
			item.prefix = true
		}
		if kindPrior(KindText(kind)) > kindPrior(KindText(item.kind)) {
			item.kind = kind
		}
	}
	for rows.Next() {
		var title, kindText, badgesJSON string
		if err := rows.Scan(&title, &kindText, &badgesJSON); err != nil {
			return nil, fmt.Errorf("scan search suggestion: %w", err)
		}
		kind := ParseKind(kindText)
		add(title, kind)
		for _, badge := range unmarshalStrings(badgesJSON) {
			add(badge, kind)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read search suggestions: %w", err)
	}

	items := make([]*suggestion, 0, len(seen))
	for _, item := range seen {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].prefix != items[j].prefix {
			return items[i].prefix
		}
		if items[i].hits != items[j].hits {
			return items[i].hits > items[j].hits
		}
		if len(items[i].text) != len(items[j].text) {
			return len(items[i].text) < len(items[j].text)
		}
		return strings.ToLower(items[i].text) < strings.ToLower(items[j].text)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.text)
	}
	return &searchv1.SuggestResponse{Suggestions: out}, nil
}

// Status reports the projection's own state. indexed is false for an empty
// projection even when a successful reindex wrote indexed_at: an index that can
// answer nothing is not "indexed" from the reader's point of view.
func (s *Service) Status(ctx context.Context) (*searchv1.SearchStatus, error) {
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	state, err := s.readState(ctx)
	if err != nil {
		return nil, err
	}
	count := 0
	if raw := strings.TrimSpace(state[stateDocumentCount]); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			count = parsed
		}
	}
	indexedAt := strings.TrimSpace(state[stateIndexedAt])
	semantic := searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED
	if s.semantic != nil && s.semantic.Enabled() {
		semantic = searchv1.SemanticLegState_SEMANTIC_LEG_STATE_READY
		if readiness, ok := s.semantic.(interface{ Ready(context.Context) bool }); ok && !readiness.Ready(ctx) {
			semantic = searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED
		}
		s.semanticFailureMu.RLock()
		degradedByQuery := s.semanticFailure != ""
		s.semanticFailureMu.RUnlock()
		if degradedByQuery {
			semantic = searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED
		}
	}
	return &searchv1.SearchStatus{
		Indexed:       count > 0 && indexedAt != "",
		DocumentCount: int32(count),
		IndexedAt:     indexedAt,
		Stale: strings.TrimSpace(state[stateLastError]) != "" ||
			strings.TrimSpace(state[stateProjectionVersion]) != projectionVersion,
		LastError:   state[stateLastError],
		SemanticLeg: semantic,
	}, nil
}

func (s *Service) recordSemanticFailure(err error) {
	if err == nil {
		return
	}
	s.semanticFailureMu.Lock()
	s.semanticFailure = err.Error()
	s.semanticFailureMu.Unlock()
	if s.logger != nil {
		s.logger.Printf("[search] semantic fusion degraded; lexical results remain available: %v", err)
	}
}

func (s *Service) clearSemanticFailure() {
	s.semanticFailureMu.Lock()
	s.semanticFailure = ""
	s.semanticFailureMu.Unlock()
}

func (s *Service) readState(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM search_state`)
	if err != nil {
		if isMissingTable(err) {
			// Status must stay answerable before the first index exists.
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read search state: %w", err)
	}
	defer rows.Close()
	state := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan search state: %w", err)
		}
		state[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read search state: %w", err)
	}
	return state, nil
}
