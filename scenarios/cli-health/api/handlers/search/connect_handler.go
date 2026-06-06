// Package search hosts the Connect-RPC handler for cli-health's
// SearchService, wired to the aisearch service (AI-first with text fallback).
package search

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"cli-health/internal/aisearch"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search"
)

// Searcher is the seam between the Connect handler and the aisearch service.
// Unit tests inject fakes; production wires a real *aisearch.Service.
type Searcher interface {
	Search(ctx context.Context, query string, limit int, mode aisearch.SearchMode) (*aisearch.SearchResponse, error)
	Status(ctx context.Context) aisearch.StatusReport
}

// Deps wires the seams the Connect search handler needs.
type Deps struct {
	Logger   *log.Logger
	Searcher Searcher
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler. Searcher may be nil when no
// search backend is configured — Search/Status then return Unimplemented.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Search(ctx context.Context, req *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	if h.deps.Searcher == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("search service not configured"))
	}
	r := req.Msg
	mode := protoModeToService(r.GetMode())
	resp, err := h.deps.Searcher.Search(ctx, r.GetQuery(), int(r.GetLimit()), mode)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	wire := &searchv1.SearchResponse{
		ModeUsed: serviceMethodToProto(resp.Method),
		Reranker: resp.Reranker,
	}
	for _, hit := range resp.Results {
		wire.Results = append(wire.Results, &searchv1.SearchResult{
			Origin:      hit.Origin,
			Group:       hit.Group,
			Name:        hit.Name,
			Description: hit.Description,
			Score:       hit.Score,
			Source:      hit.Source,
			FullPath:    hit.FullPath,
			Tags:        hit.Tags,
			Binding:     hit.Binding,
			Weak:        hit.Weak,
		})
	}
	return connect.NewResponse(wire), nil
}

func (h *connectHandler) Status(ctx context.Context, _ *connect.Request[searchv1.StatusRequest]) (*connect.Response[searchv1.StatusResponse], error) {
	if h.deps.Searcher == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("search service not configured"))
	}
	rep := h.deps.Searcher.Status(ctx)
	return connect.NewResponse(&searchv1.StatusResponse{
		Available:            rep.Available,
		Ollama:               rep.Ollama,
		Qdrant:               rep.Qdrant,
		IndexedCount:         int32(rep.IndexedCount),
		LastReconcileAt:      rep.LastReconcileAt,
		LastReconcileOutcome: rep.LastReconcileOutcome,
		Reranker:             rep.Reranker,
	}), nil
}

func protoModeToService(m searchv1.Mode) aisearch.SearchMode {
	switch m {
	case searchv1.Mode_MODE_AI:
		return aisearch.ModeAI
	case searchv1.Mode_MODE_TEXT:
		return aisearch.ModeText
	default:
		return aisearch.ModeAuto
	}
}

func serviceMethodToProto(method string) searchv1.Mode {
	switch method {
	case "ai":
		return searchv1.Mode_MODE_AI
	case "text":
		return searchv1.Mode_MODE_TEXT
	default:
		return searchv1.Mode_MODE_UNSPECIFIED
	}
}
