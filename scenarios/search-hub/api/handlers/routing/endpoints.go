package routing

import (
	"search-hub/internal/module"

	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
)

// Endpoints is the machine-readable description of the routing module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so adding or renaming an RPC in routing.proto breaks this file at
// compile time. The global parity test (TestProtoConnectParity in
// api/internal/modules/registry_test.go) walks the proto FileDescriptor and
// asserts every rpc has exactly one entry here — including Status, which
// reports federation health (per-provider reachability + classifier/reranker
// availability).
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "routing_query",
		Path:        routingconnect.RoutingServiceQueryProcedure,
		Method:      "POST",
		Summary:     "Federated query across registered providers",
		Description: "Fans out a query to providers selected by explicit --type, --all, or --group, or automatically via the Ollama classifier when no explicit selector is given; maps each response through the generic adapter, and returns results grouped by provider. Degraded providers are skipped with a note, never failing the whole query.",
		Category:    "routing",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"query":   "string (required) — natural-language query text",
				"types":   "array<string> — explicit leaf types to route to",
				"all":     "bool — fan out to every active provider",
				"limit":   "int32 — per-provider result cap (default 10)",
				"group":   "string — scope to one scenario's leaves",
				"explain": "bool — include routing rationale",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"groups":           "array<ProviderResultGroup> — by-provider grouping (always populated)",
				"ranked":           "array<SearchHit> — unified ranked list (populated when the reranker fuses the per-provider shortlists)",
				"corpora_searched": "array<string> — providers hit",
				"degraded":         "bool — any provider degraded",
				"reranked":         "bool — true when the unified list was reranked",
				"latency_ms":       "int64",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Empty query, or no routing selector while no classifier is wired (pass --type/--all/--group)"},
			{Status: 500, Code: "internal", Description: "Registry read failure"},
		},
		Examples: []module.Example{
			{Name: "Explicit-type query", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.routing.RoutingService/Query -H 'Content-Type: application/json' -d '{\"query\":\"restart a scenario\",\"types\":[\"command\"],\"limit\":5}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "search-hub query",
			Args:    []string{"<text>", "--type", "<types>", "--all", "--limit", "<n>"},
		},
	},
	{
		ID:          "routing_status",
		Path:        routingconnect.RoutingServiceStatusProcedure,
		Method:      "POST",
		Summary:     "Federation status (per-provider health + model availability)",
		Description: "Reports each ACTIVE provider's reachability plus whether the classifier and reranker models are available. An unreachable leaf is reported degraded rather than failing the call; only a registry read failure errors. Backs `search-hub status`.",
		Category:    "routing",
		Request:     &module.Schema{Type: "object"},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"providers":            "array<ProviderHealth> — per-leaf reachability + freshness note",
				"classifier_available": "bool — automatic routing model reachable",
				"reranker_available":   "bool — unified-rerank model reachable (false ⇒ degraded ranking mode)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Registry read failure"},
		},
		Examples: []module.Example{
			{Name: "Federation status", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.routing.RoutingService/Status -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "search-hub federation",
		},
	},
}
