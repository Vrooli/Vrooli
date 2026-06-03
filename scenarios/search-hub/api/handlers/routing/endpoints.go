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
// asserts every rpc has exactly one entry here — including Status, which is
// mounted but Unimplemented until Phase 7.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "routing_query",
		Path:        routingconnect.RoutingServiceQueryProcedure,
		Method:      "POST",
		Summary:     "Federated query across registered providers",
		Description: "Fans out a query to providers selected by explicit --type, --all, or --group (Phase 4: no automatic classifier yet), maps each response through the generic adapter, and returns results grouped by provider. Degraded providers are skipped with a note, never failing the whole query.",
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
				"ranked":           "array<SearchHit> — unified ranked list (populated once rerank lands, Phase 6)",
				"corpora_searched": "array<string> — providers hit",
				"degraded":         "bool — any provider degraded",
				"reranked":         "bool — false until Phase 6",
				"latency_ms":       "int64",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Empty query, or no routing selector (pass --type/--all/--group; classifier is Phase 5)"},
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
		Summary:     "Federation status (reserved — Phase 7)",
		Description: "Reserved for the metrics domain (Phase 7): per-provider reachability/freshness + classifier/reranker availability. Returns Unimplemented until then; no CLI command binds to it yet (manifest omitted[]).",
		Category:    "routing",
		Request:     &module.Schema{Type: "object"},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"providers":            "array<ProviderHealth>",
				"classifier_available": "bool",
				"reranker_available":   "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Lands in Phase 7 (federation status + metrics)"},
		},
	},
}
