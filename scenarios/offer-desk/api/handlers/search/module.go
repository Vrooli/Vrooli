// Package search exposes Offer Desk's authoritative catalog through the shared
// federated search leaf. It projects the live catalog (every node kind plus
// relationships, triggers, facts, evaluations, proposals, and audit rows) and
// never invents a second catalog or writes to the store.
package search

import (
	"context"
	"net/http"
	"time"

	"offer-desk/internal/aisearch"
	"offer-desk/internal/catalog"
	"offer-desk/internal/module"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/search/search_v1connect"
)

type handler struct {
	service aisearch.Searcher
}

var _ searchconnect.SearchServiceHandler = handler{}

func (h handler) Search(ctx context.Context, req *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	response, err := h.service.Search(ctx, req.Msg.GetQuery(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &searchv1.SearchResponse{
		Generation:     response.Generation,
		MaterializedAt: timestampString(response.MaterializedAt),
		Results:        make([]*searchv1.SearchResult, 0, len(response.Hits)),
	}
	for _, hit := range response.Hits {
		out.Results = append(out.Results, &searchv1.SearchResult{
			Id:         hit.ID,
			Title:      hit.Title,
			Snippet:    hit.Snippet,
			Score:      hit.Score,
			Kind:       hit.Kind,
			FollowUp:   hit.FollowUp,
			Freshness:  hit.Freshness,
			Historical: hit.Historical,
		})
	}
	return connect.NewResponse(out), nil
}

func (h handler) Status(ctx context.Context, _ *connect.Request[searchv1.StatusRequest]) (*connect.Response[searchv1.StatusResponse], error) {
	status, err := h.service.Status(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&searchv1.StatusResponse{
		Available:     status.Available,
		IndexedCount:  int32(status.IndexedCount),
		LastIndexedAt: timestampString(status.LastIndexedAt),
		Generation:    status.Generation,
	}), nil
}

func timestampString(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.UTC().Format(time.RFC3339Nano)
}

func Module(db *database.RoutedDB, clock schedule.Clock, searcher aisearch.Searcher) module.Module {
	if clock == nil {
		clock = schedule.System()
	}
	if searcher == nil {
		// Fallback for a caller that has not assembled the shared engine: serve
		// the authoritative lexical projection directly. A normal boot passes an
		// engine-backed LiveSearch.
		source := aisearch.NewStoreSource(catalog.NewStore(db, clock.Now), clock.Now)
		searcher = aisearch.NewLexicalSearch(aisearch.NewService(source))
	}
	h := handler{service: searcher}
	path, connectHandler := searchconnect.NewSearchServiceHandler(h)
	return module.Module{
		Name: "search",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema is empty: the search leaf projects the catalog read model and owns no
// tables of its own.
func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{ID: "search_query", Path: searchconnect.SearchServiceSearchProcedure, Method: http.MethodPost, Summary: "Search the authoritative offer catalog for federation", Category: "search"},
	{ID: "search_status", Path: searchconnect.SearchServiceStatusProcedure, Method: http.MethodPost, Summary: "Report truthful live catalog search status", Category: "search"},
}
