// Package search exposes Content Desk's authoritative live editorial search.
package search

import (
	"context"
	"strings"
	"time"

	"connectrpc.com/connect"
	internalartifacts "content-desk/internal/artifacts"
	internalledger "content-desk/internal/ledger"
	"content-desk/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/search/search_v1connect"
)

type handler struct {
	drafts internalartifacts.Repository
	ledger internalledger.Repository
}

var _ searchconnect.SearchServiceHandler = handler{}

func (h handler) Search(ctx context.Context, req *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	query := strings.ToLower(strings.TrimSpace(req.Msg.Query))
	limit := int(req.Msg.Limit)
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	out := &searchv1.SearchResponse{}
	drafts, err := h.drafts.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, d := range drafts {
		if query == "" || strings.Contains(strings.ToLower(d.ID+" "+d.Body+" "+d.Channel+" "+d.Lane+" "+d.SKU), query) {
			out.Results = append(out.Results, &searchv1.SearchResult{Id: d.ID, Title: d.ID, Snippet: d.Body, Score: 1, Kind: "draft"})
			if len(out.Results) >= limit {
				return connect.NewResponse(out), nil
			}
		}
	}
	records, err := h.ledger.ListPublishHistory(ctx, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, r := range records {
		if query == "" || strings.Contains(strings.ToLower(r.ID+" "+r.DraftID+" "+r.PublishedURL+" "+r.PlatformPostID), query) {
			out.Results = append(out.Results, &searchv1.SearchResult{Id: r.ID, Title: r.DraftID, Snippet: r.PublishedURL, Score: .9, Kind: "publish-record"})
			if len(out.Results) >= limit {
				break
			}
		}
	}
	return connect.NewResponse(out), nil
}

func (h handler) Status(ctx context.Context, _ *connect.Request[searchv1.StatusRequest]) (*connect.Response[searchv1.StatusResponse], error) {
	drafts, err := h.drafts.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	records, err := h.ledger.ListPublishHistory(ctx, 100)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&searchv1.StatusResponse{
		Available:     true,
		IndexedCount:  int32(len(drafts) + len(records)),
		LastIndexedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}), nil
}

func Module(db *database.RoutedDB) module.Module {
	path, h := searchconnect.NewSearchServiceHandler(handler{drafts: internalartifacts.NewSQLiteRepository(db), ledger: internalledger.NewSQLiteRepository(db)})
	return module.Module{Name: "search", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{{ID: "search_query", Path: searchconnect.SearchServiceSearchProcedure, Method: "POST", Summary: "Search drafts and publish history for federation", Category: "search"}, {ID: "search_status", Path: searchconnect.SearchServiceStatusProcedure, Method: "POST", Summary: "Report live editorial search status", Category: "search"}}
