// Package search hosts the Connect-RPC handler for ui-health's
// SearchService. Wires to the aisearch service (AI-first with text fallback).
package search

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"ui-health/internal/aisearch"

	provenancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/contracts/provenance"
	widgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/contracts/widget"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search"
)

// Searcher is the seam between the Connect handler and the aisearch service.
type Searcher interface {
	Search(ctx context.Context, query string, limit int, mode aisearch.SearchMode) (*aisearch.SearchResponse, error)
	Status(ctx context.Context) aisearch.StatusReport
}

type Deps struct {
	Logger   *log.Logger
	Searcher Searcher
}

type connectHandler struct {
	deps Deps
}

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
	resp, err := h.deps.Searcher.Search(ctx, r.GetQuery(), int(r.GetLimit()), protoModeToService(r.GetMode()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	wire := &searchv1.SearchResponse{ModeUsed: serviceMethodToProto(resp.Method)}
	for _, hit := range resp.Results {
		wire.Results = append(wire.Results, &searchv1.SearchResult{
			Scenario:    hit.Scenario,
			Slot:        hit.Slot,
			Kind:        kindFromString(hit.Kind),
			DisplayName: hit.DisplayName,
			Description: hit.Description,
			FilePath:    hit.FilePath,
			Score:       hit.Score,
			Provenance:  provenanceToProto(hit.Provenance),
			Widget:      widgetToProto(hit.Widget),
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

func kindFromString(s string) searchv1.SurfaceKind {
	if v, ok := searchv1.SurfaceKind_value[s]; ok {
		return searchv1.SurfaceKind(v)
	}
	return searchv1.SurfaceKind_SURFACE_KIND_UNSPECIFIED
}

func provenanceToProto(p *aisearch.ProvenancePayload) *provenancev1.ComponentProvenance {
	if p == nil {
		return nil
	}
	pv := provenancev1.Provenance_PROVENANCE_UNSPECIFIED
	if v, ok := provenancev1.Provenance_value[p.Provenance]; ok {
		pv = provenancev1.Provenance(v)
	}
	return &provenancev1.ComponentProvenance{
		Provenance:     pv,
		Library:        p.Library,
		LibraryVersion: p.LibraryVersion,
		ComponentName:  p.ComponentName,
		AdoptionId:     p.AdoptionID,
		AppliedAt:      p.AppliedAt,
		SourceSha256:   p.SourceSha256,
		DriftHash:      p.DriftHash,
		FilePath:       p.FilePath,
	}
}

func widgetToProto(w *aisearch.WidgetPayload) *widgetv1.WidgetDeclaration {
	if w == nil {
		return nil
	}
	slot := widgetv1.WidgetSlot_WIDGET_SLOT_UNSPECIFIED
	if v, ok := widgetv1.WidgetSlot_value[w.Slot]; ok {
		slot = widgetv1.WidgetSlot(v)
	}
	scope := widgetv1.WidgetScope_WIDGET_SCOPE_UNSPECIFIED
	if v, ok := widgetv1.WidgetScope_value[w.Scope]; ok {
		scope = widgetv1.WidgetScope(v)
	}
	return &widgetv1.WidgetDeclaration{
		WidgetId:        w.WidgetID,
		ComponentName:   w.ComponentName,
		PropsSchemaJson: w.PropsSchemaJSON,
		Slot:            slot,
		Scope:           scope,
		Description:     w.Description,
		FilePath:        w.FilePath,
	}
}
