package search

import (
	"log"

	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/search/search_v1connect"
)

// ProtoFile exposes the search domain's proto FileDescriptor so a global parity
// test can walk it without importing the gen/go package directly.
var ProtoFile = searchv1.File_architecture_cartographer_v1_search_search_proto

// Module returns the search domain's contribution to the API router: the Connect
// SearchService handler mounted at the generated procedure paths. Searcher may
// be nil; the handler then returns Unimplemented. overrides, when non-nil,
// enables the token-gated query-time override channel.
func Module(logger *log.Logger, searcher Searcher, overrides *OverrideGate) module.Module {
	pattern, connectHandler := searchconnect.NewSearchServiceHandler(NewConnectHandler(Deps{
		Logger:    logger,
		Searcher:  searcher,
		Overrides: overrides,
	}))
	return module.Module{
		Name: "search",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — search is stateless; the index lives in Qdrant, not the
// scenario database.
func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "search_query",
		Path:        searchconnect.SearchServiceSearchProcedure,
		Method:      "POST",
		Summary:     "Search scenario domains across the fleet",
		Description: "Term-agnostic semantic (AI) and text-fallback search over the derived domain map of every scenario (e.g. \"how does authoring work in plan-manager\").",
		Category:    "search",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"query": "string",
				"limit": "int32",
				"mode":  "Mode",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"results":   "array<SearchResult>",
				"mode_used": "Mode",
				"reranker":  "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Returned only when no search backend is configured"},
		},
		Examples: []module.Example{
			{Name: "Search", Curl: "curl http://localhost:${API_PORT}/vrooli.architecture_cartographer.v1.search.SearchService/Search -H 'Content-Type: application/json' -d '{\"query\":\"how does authoring work in plan-manager\",\"limit\":10}'"},
		},
	},
	{
		ID:          "search_status",
		Path:        searchconnect.SearchServiceStatusProcedure,
		Method:      "POST",
		Summary:     "Report search backend availability",
		Description: "Reports whether ollama and qdrant are reachable, indexed domain count, and the most recent reconcile outcome.",
		Category:    "search",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"available":              "boolean",
				"ollama":                 "boolean",
				"qdrant":                 "boolean",
				"indexed_count":          "int32",
				"last_reconcile_at":      "string",
				"last_reconcile_outcome": "string",
				"reranker":               "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Returned only when no search backend is configured"},
		},
		Examples: []module.Example{
			{Name: "Status", Curl: "curl http://localhost:${API_PORT}/vrooli.architecture_cartographer.v1.search.SearchService/Status -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
