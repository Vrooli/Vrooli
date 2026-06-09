// Package livesearch is the API's L0/L1 live web-search Connect-RPC surface. It
// mounts the generated LiveSearchService handler and exports the static
// EndpointDescriptor for codegen. Unlike findings it owns no SQLite tables, so
// it exposes no Schema() and is not registered in AllSchemas.
package livesearch

import (
	"log"

	"web-search/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	livesearchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/livesearch/livesearch_v1connect"

	internallivesearch "web-search/internal/livesearch"
)

// Module returns the live-search domain's contribution to the API: the
// generated Connect-RPC service handler. main.go owns the SearXNG client /
// cache / governor / synthesizer wiring and injects the constructed service;
// tests build their own service over fakes.
func Module(svc *internallivesearch.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := livesearchconnect.NewLiveSearchServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "livesearch",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Endpoints describes the live-search module's public surface. The Connect-RPC
// method path references the generated *Procedure constant, so renaming the RPC
// in livesearch.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "livesearch_search",
		Path:        livesearchconnect.LiveSearchServiceSearchProcedure,
		Method:      "POST",
		Summary:     "Live web search (L0) with optional snippet synthesis (L1)",
		Description: "Fans the query out to SearXNG and returns normalized results. With --synthesis, runs an optional always-cited LLM pass over the returned snippets (abstains rather than fabricates). Budget-governed: returns degraded without calling SearXNG when the live budget is exhausted.",
		Category:    "livesearch",
		CLIMapping:  &module.CLIMapping{Command: "web-search search", Args: []string{"<query>", "--limit", "<n>", "--synthesis"}},
	},
}
