package search

import (
	"log"

	"ui-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search/search_v1connect"
)

var ProtoFile = searchv1.File_ui_health_v1_search_search_proto

func Module(logger *log.Logger, searcher Searcher) module.Module {
	connectPath, connectHandler := searchconnect.NewSearchServiceHandler(NewConnectHandler(Deps{
		Logger:   logger,
		Searcher: searcher,
	}))
	return module.Module{
		Name: "search",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "search_query",
		Path:        searchconnect.SearchServiceSearchProcedure,
		Method:      "POST",
		Summary:     "Search UI surface across scenarios",
		Description: "Semantic (AI) and text-fallback search across every scenario's UI surface (components, pages, features, hooks).",
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
			},
		},
		Examples: []module.Example{
			{Name: "Search", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.search.SearchService/Search -H 'Content-Type: application/json' -d '{\"query\":\"date picker\",\"limit\":10}'"},
		},
	},
	{
		ID:          "search_status",
		Path:        searchconnect.SearchServiceStatusProcedure,
		Method:      "POST",
		Summary:     "Report search backend availability",
		Description: "Reports whether ollama and qdrant are reachable, indexed doc count, and the most recent reconcile outcome.",
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
			},
		},
		Examples: []module.Example{
			{Name: "Status", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.search.SearchService/Status -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
