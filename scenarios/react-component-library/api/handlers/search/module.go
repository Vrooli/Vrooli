// Package search exposes the unified SearchService over Connect-RPC.
//
// The handler is intentionally thin: it owns transport concerns (Connect error
// mapping and endpoint descriptors) and leaves ranking, indexing and status to
// internal/search. One package owns the index; this package owns the wire.
package search

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search/search_v1connect"
	"react-component-library/internal/module"
	domain "react-component-library/internal/search"
)

// Handler adapts the search service to the generated Connect interface.
type Handler struct {
	Service *domain.Service
}

// NewHandler is the explicit constructor used by Module and tests.
func NewHandler(service *domain.Service) Handler {
	return Handler{Service: service}
}

// Search returns ranked results for one query.
func (h Handler) Search(ctx context.Context, req *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	if h.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, domain.ErrNotConfigured)
	}
	response, err := h.Service.Search(ctx, req.Msg)
	if err != nil {
		return nil, searchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

// Suggest returns completion strings from the indexed corpus.
func (h Handler) Suggest(ctx context.Context, req *connect.Request[searchv1.SuggestRequest]) (*connect.Response[searchv1.SuggestResponse], error) {
	if h.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, domain.ErrNotConfigured)
	}
	response, err := h.Service.Suggest(ctx, req.Msg)
	if err != nil {
		return nil, searchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

// GetSearchStatus reports index freshness and the semantic leg state.
func (h Handler) GetSearchStatus(ctx context.Context, _ *connect.Request[searchv1.GetSearchStatusRequest]) (*connect.Response[searchv1.SearchStatus], error) {
	if h.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, domain.ErrNotConfigured)
	}
	status, err := h.Service.Status(ctx)
	if err != nil {
		return nil, searchConnectError(err)
	}
	return connect.NewResponse(status), nil
}

// Reindex rebuilds the lexical index. force=false is a no-op for a current
// index, matching the proto's documented intent.
func (h Handler) Reindex(ctx context.Context, req *connect.Request[searchv1.ReindexRequest]) (*connect.Response[searchv1.ReindexResponse], error) {
	if h.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, domain.ErrNotConfigured)
	}
	count, indexedAt, err := h.Service.Reindex(ctx, req.Msg.GetForce())
	if err != nil {
		return nil, searchConnectError(err)
	}
	return connect.NewResponse(&searchv1.ReindexResponse{
		DocumentCount: int32(count),
		IndexedAt:     indexedAt.UTC().Format(time.RFC3339Nano),
	}), nil
}

// searchConnectError is the single code mapping for this service. The index
// being absent is a wiring/precondition problem, not an internal failure, and a
// caller disconnect is not logged as a server fault.
func searchConnectError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	case errors.Is(err, context.DeadlineExceeded):
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	case errors.Is(err, domain.ErrNotConfigured):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, domain.ErrIndexNotInitialized):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "no such table") {
			return connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}
}

// Endpoints is the static descriptor set consumed by the endpoint inventory.
// Every procedure in the generated SearchService is listed exactly once.
var Endpoints = []module.EndpointDescriptor{
	{ID: "search_query", Path: searchconnect.SearchServiceSearchProcedure, Method: "POST", Summary: "Search the unified workbench index", Category: "search"},
	{ID: "search_suggest", Path: searchconnect.SearchServiceSuggestProcedure, Method: "POST", Summary: "Suggest indexed titles and tags", Category: "search"},
	{ID: "search_status", Path: searchconnect.SearchServiceGetSearchStatusProcedure, Method: "POST", Summary: "Report search index freshness and semantic leg state", Category: "search"},
	{ID: "search_reindex", Path: searchconnect.SearchServiceReindexProcedure, Method: "POST", Summary: "Rebuild the unified search index", Category: "search"},
}

// Module wires the search service into the scenario server. scenarioRoot is the
// scenario directory that contains catalog/, not the repository root.
func Module(db *sql.DB, scenarioRoot string, availability domain.AvailabilityFunc) module.Module {
	return ModuleWithService(domain.NewService(db, scenarioRoot, availability))
}

// ModuleWithService mounts an already-built service. Main uses this when it
// wants one service instance to run the boot reindex and serve RPCs, instead of
// building a second instance behind the handler.
func ModuleWithService(service *domain.Service) module.Module {
	path, handler := searchconnect.NewSearchServiceHandler(NewHandler(service))
	return module.Module{
		Name: "search",
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the domain schema so the module registry can list search
// with every other domain's provider.
func Schema() string { return domain.Schema() }

// NewService re-exports the domain constructor for main wiring.
func NewService(db *sql.DB, scenarioRoot string, availability domain.AvailabilityFunc) *domain.Service {
	return domain.NewService(db, scenarioRoot, availability)
}

// Reindex re-exports the boot reindex helper for main wiring.
func Reindex(ctx context.Context, db *sql.DB, scenarioRoot string, availability domain.AvailabilityFunc, force bool) (int, time.Time, error) {
	return domain.Reindex(ctx, db, scenarioRoot, availability, force)
}
