package search

import (
	"log"

	"cli-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search/search_v1connect"
)

// ProtoFile exposes the search domain's proto FileDescriptor so the global
// parity test can walk it without importing the gen/go package directly.
var ProtoFile = searchv1.File_cli_health_v1_search_search_proto

// Module returns the search domain's contribution to the API: the Connect
// SearchService handler mounted at the generated procedure paths. Searcher
// may be nil; the handler then returns Unimplemented (Phase 1 stub path).
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

// Schema returns "" — Phase 1 stub keeps search stateless. Phase 3 may
// add tables for indexed-document tracking or reconcile checkpoints.
func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "search_query",
		Path:        searchconnect.SearchServiceSearchProcedure,
		Method:      "POST",
		Summary:     "Search CLI commands across scenarios",
		Description: "Semantic (AI) and text-fallback search across every scenario's CLI manifest. Phase 1 stub returns Unimplemented.",
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
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Phase 1 stub; Phase 3 wires AI + text retrieval"},
		},
		Examples: []module.Example{
			{Name: "Search", Curl: "curl http://localhost:${API_PORT}/vrooli.cli_health.v1.search.SearchService/Search -H 'Content-Type: application/json' -d '{\"query\":\"list goldens\",\"limit\":10}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "cli-health search query",
			Args:    []string{"<text>"},
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
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Phase 1 stub; Phase 3 wires the backend probes"},
		},
		Examples: []module.Example{
			{Name: "Status", Curl: "curl http://localhost:${API_PORT}/vrooli.cli_health.v1.search.SearchService/Status -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "cli-health search status",
		},
	},
}
