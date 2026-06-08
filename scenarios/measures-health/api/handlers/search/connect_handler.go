// Package search is the Connect-RPC surface for the central measures index — the
// single search-hub "measure" provider. It is a thin translation layer: it
// delegates to internal/measureindex.Provider (the corpus + the shared
// measures-go Engine) and maps the resulting MeasureHits onto the generated proto
// messages. The wire shape (snake_case measure carrier via json_name) is the
// contract search-hub's generic result adapter reads verbatim.
package search

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	measures "github.com/vrooli/measures-go"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/search"
)

// maxSearchLimit clamps the per-query result count. A measure provider returns
// the single best answer; the cap is defensive headroom for a future
// multi-candidate matcher.
const maxSearchLimit = 10

// Searcher is the seam between the Connect handler and the measures provider.
// Unit tests inject a fake; production wires a *measureindex.Provider.
type Searcher interface {
	Query(ctx context.Context, question string, limit int) ([]*measures.MeasureHit, string, error)
	Status(ctx context.Context) (available, ollama, qdrant bool, indexed int, matcher string)
}

// Deps wires the Connect search handler.
type Deps struct {
	Searcher Searcher
	Logger   *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the handler. Searcher may be nil when no index is
// configured — Search/Status then return Unimplemented.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Search(ctx context.Context, req *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	if h.deps.Searcher == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("measures index not configured"))
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 1
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	hits, matcher, err := h.deps.Searcher.Query(ctx, req.Msg.GetQuery(), limit)
	if err != nil {
		// Degrade like a search-hub provider: log and return an honest empty group
		// rather than failing the federated query. (A two-hop measure answer that
		// errors should not sink the user's whole search.)
		h.deps.Logger.Printf("[measures-health] search query %q degraded: %v", req.Msg.GetQuery(), err)
		return connect.NewResponse(&searchv1.SearchResponse{Matcher: matcher}), nil
	}
	resp := &searchv1.SearchResponse{Matcher: matcher, Results: make([]*searchv1.MeasureResult, 0, len(hits))}
	for _, hit := range hits {
		if hit == nil {
			continue
		}
		resp.Results = append(resp.Results, &searchv1.MeasureResult{
			Score:   hit.Score,
			Measure: measureHitToProto(hit),
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) Status(ctx context.Context, _ *connect.Request[searchv1.StatusRequest]) (*connect.Response[searchv1.StatusResponse], error) {
	if h.deps.Searcher == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("measures index not configured"))
	}
	available, ollama, qdrant, indexed, matcher := h.deps.Searcher.Status(ctx)
	return connect.NewResponse(&searchv1.StatusResponse{
		Available:    available,
		Ollama:       ollama,
		Qdrant:       qdrant,
		IndexedCount: int32(indexed),
		Matcher:      matcher,
	}), nil
}

// measureHitToProto maps the engine's MeasureHit onto the wire carrier. The proto
// fields carry snake_case json_name so the emitted protojson matches search-hub's
// adapter contract; this mapping is the only place the two shapes meet.
func measureHitToProto(h *measures.MeasureHit) *searchv1.MeasureHit {
	return &searchv1.MeasureHit{
		MeasureId:     h.MeasureID,
		Scenario:      h.Scenario,
		Params:        h.Params,
		Answer:        h.Answer,
		Needs:         h.Needs,
		Effect:        h.Effect,
		ExecutedQuery: h.ExecutedQuery,
		Confidence:    h.Confidence,
	}
}
