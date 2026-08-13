package search

import (
	"context"
	"strings"
	"time"

	"asset-studio/internal/module"
	core "asset-studio/internal/studio"
	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/search/search_v1connect"
)

type handler struct{ store core.StateStore }

func (h handler) Search(ctx context.Context, req *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	state, err := h.store.Load(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	q := strings.ToLower(strings.TrimSpace(req.Msg.Query))
	limit := int(req.Msg.Limit)
	if limit < 1 || limit > 50 {
		limit = 20
	}
	out := &searchv1.SearchResponse{}
	for _, id := range state.Identities {
		text := id.Name + " " + string(id.Kind)
		for k, v := range id.Traits {
			text += " " + k + " " + v
		}
		if q == "" || strings.Contains(strings.ToLower(text), q) {
			out.Results = append(out.Results, &searchv1.SearchResult{Id: id.ID, Title: id.Name, Snippet: text, Score: 1, Kind: "identity"})
			if len(out.Results) >= limit {
				return connect.NewResponse(out), nil
			}
		}
	}
	for _, a := range state.Assets {
		if a.Status != core.Released {
			continue
		}
		text := a.AltText + " " + a.Disclosure + " " + a.MediaType + " " + a.DerivationOperation
		if q == "" || strings.Contains(strings.ToLower(text), q) {
			out.Results = append(out.Results, &searchv1.SearchResult{Id: a.ID, Title: a.AltText, Snippet: text, Score: .9, Kind: "released-asset"})
			if len(out.Results) >= limit {
				break
			}
		}
	}
	return connect.NewResponse(out), nil
}

func (h handler) Status(ctx context.Context, _ *connect.Request[searchv1.StatusRequest]) (*connect.Response[searchv1.StatusResponse], error) {
	s, e := h.store.Load(ctx)
	if e != nil {
		return nil, connect.NewError(connect.CodeInternal, e)
	}
	return connect.NewResponse(&searchv1.StatusResponse{
		Available:     true,
		IndexedCount:  int32(len(s.Identities) + len(s.Assets)),
		LastIndexedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}), nil
}

func Module(db *database.RoutedDB) module.Module {
	p, h := searchconnect.NewSearchServiceHandler(handler{store: core.NewSQLiteStore(db)})
	return module.Module{Name: "search", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: p, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{{ID: "search_query", Path: searchconnect.SearchServiceSearchProcedure, Method: "POST", Category: "search", Summary: "Search identities and released asset metadata"}, {ID: "search_status", Path: searchconnect.SearchServiceStatusProcedure, Method: "POST", Category: "search", Summary: "Report Asset Studio search metadata status"}}
