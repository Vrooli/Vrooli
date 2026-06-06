package aisearch

import "context"

// typed.go is the typed-projection convenience over the concrete *Service: an
// adopter that wants its own struct type H back — instead of reading fields off
// each SearchResult's Payload at the call site — passes a projector and gets a
// []H page, with all pipeline ordering still owned by Service.
//
// Why a generic FUNCTION and not a generic *Service[H]: the pipeline stages the
// Service owns (project-in-place, post-filter, rerank, floor, decorate,
// weak-label) all operate on the uniform SearchResult and MUST run before the
// page is cut; only the FINAL shape handed to the caller is corpus-specific. A
// generic struct would have to thread H through Status, the reindex job control,
// and the in-place enrichment Projector — none of which are typed — for no gain,
// and would fracture the single result type the federation contract depends on.
// Projecting to H at the boundary captures the whole ergonomic win (typed hits,
// no Payload spelunking in the adopter) with zero blast radius on the rest of the
// surface. This is the answer to the graduation retrospective's "generic
// Service[H] would be cleaner" note (lesson 4): the clean form is a boundary
// projection, not a generic engine.

// TypedProjector maps one finished SearchResult (already reranked, floored, and
// weak-labeled) into an adopter's own hit type H. It runs after the page is cut,
// so it sees exactly the results the caller will return.
type TypedProjector[H any] func(SearchResult) H

// SearchTyped runs s.Search and projects each result of the returned page into
// the adopter's hit type H via project. It returns the typed page plus the raw
// SearchResponse so the caller still has Method / Reranker / Total for its wire
// envelope. On a search error it returns the (zero) response and a nil page.
//
// This is the recommended adoption shape for a non-doc corpus that keeps its
// fields in Payload (commands, surfaces, …): the adopter no longer writes a
// manual `for _, r := range resp.Results { ... }` projection loop and cannot
// drift from the Service's result ordering.
func SearchTyped[H any](ctx context.Context, s *Service, q SearchQuery, project TypedProjector[H]) ([]H, SearchResponse, error) {
	resp, err := s.Search(ctx, q)
	if err != nil {
		return nil, resp, err
	}
	out := make([]H, len(resp.Results))
	for i := range resp.Results {
		out[i] = project(resp.Results[i])
	}
	return out, resp, nil
}
