package routing

import (
	"search-hub/internal/module"

	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
)

// Endpoints is the machine-readable description of the routing module's public
// surface. Connect-RPC method paths reference generated *Procedure constants,
// and the endpoint catalog is consumed by API discovery and generated docs.
// Keep its field and fallback descriptions aligned with the typed runtime
// contract; proto/Connect parity is checked separately.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "routing_query",
		Path:        routingconnect.RoutingServiceQueryProcedure,
		Method:      "POST",
		Summary:     "Federated query across registered providers",
		Description: "Routes a query to explicit provider scope (--type, --all, or --group) or, without an explicit selector, through the active provider-selection strategy over eligible provider descriptions. Semantic selection requires a verified current caller context; when it is unavailable, the supported lexical fallback remains available. Provider results retain per-part evidence and degradation details. No-route and partial responses may contain no provider groups.",
		Category:    "routing",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"query":   "string (required) — natural-language query text",
				"types":   "array<string> — explicit leaf types to route to",
				"all":     "bool — fan out to every active provider",
				"limit":   "int32 — per-provider result cap (default 10)",
				"group":   "string — scope to one scenario's leaves",
				"scope":   "string — optional corpus facet passed to scoped providers",
				"explain": "bool — include routing rationale",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"groups":                 "array<ProviderResultGroup> — provider summaries when providers return results; may be empty when no provider is routed or returns results",
				"ranked":                 "array<SearchHit> — complete hit payload when reranking succeeds",
				"corpora_searched":       "array<string> — providers queried",
				"routing_explanation":    "array<string> — populated when explain=true",
				"degraded":               "bool — any provider or retrieval leg degraded",
				"partial":                "bool — deadline expired after at least one provider returned usable results",
				"pending_providers":      "int32 — provider groups that did not complete before the deadline",
				"reranked":               "bool — unified result list was reranked",
				"reranker_leg":           "string — reranker leg that produced ranked results, or none",
				"routing_degrade_reason": "string — stable reason for routing-path degradation",
				"ordered_by":             "string — score field used to order ranked results",
				"selector_leg":           "string — provider-selection stage used",
				"routing_trace":          "RoutingTrace — diagnostic selection trace for federated evaluation",
				"degradations":           "array<SearchDegradation> — per-provider and retrieval-leg failure/fallback details",
				"query_parts":            "array<QueryPartEvidence> — per-request-part searched owners, evidence references, status, and coverage gaps",
				"correction_candidates":  "array<CorrectionCandidate> — evidence-backed identifier suggestions; may be empty",
				"latency_ms":             "int64 — total query latency",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Empty query"},
			{Status: 500, Code: "internal", Description: "Registry read failure"},
		},
		Examples: []module.Example{
			{Name: "Explicit-type query", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.routing.RoutingService/Query -H 'Content-Type: application/json' -d '{\"query\":\"restart a scenario\",\"types\":[\"command\"],\"limit\":5}'"},
		},
	},
	{
		ID:          "routing_status",
		Path:        routingconnect.RoutingServiceStatusProcedure,
		Method:      "POST",
		Summary:     "Federation status (provider health, active strategy, and reranker availability)",
		Description: "Reports ACTIVE provider reachability, circuit-open share/quorum, lifecycle and quality exclusions, the active provider-selection strategy and strategy catalog, and shared reranker state. The retired classifier field is retained as false for older clients. An unreachable leaf is reported degraded rather than failing the call; only a registry read failure errors. Backs `search-hub federation status`.",
		Category:    "routing",
		Request:     &module.Schema{Type: "object"},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"providers":            "array<ProviderHealth> — active per-leaf reachability and probed index age",
				"audit_providers":      "array<ProviderHealth> — accounting rows without current registry descriptors; not presented as active",
				"active_strategy":      "string — configured strategy used for serving provider selection",
				"strategies":           "array<RetrievalStrategyInfo> — registered data-defined strategy catalog",
				"classifier_available": "bool — retained compatibility field; false after LLM classifier retirement",
				"reranker_available":   "bool — unified-rerank model reachable (false ⇒ degraded ranking mode)",
				"reranker_leg":         "string — current shared reranker leg, or none",
				"incubating":           "array<IncubatingProvider> — registered providers not yet active",
				"circuit_open_share":   "double — active leaves with an open transport circuit",
				"circuit_open_quorum":  "double — declared federation-degraded threshold",
				"federation_degraded":  "bool — circuit-open share meets quorum",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Registry read failure"},
		},
		Examples: []module.Example{
			{Name: "Federation status", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.routing.RoutingService/Status -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "routing_provider_status",
		Path:        routingconnect.RoutingServiceProviderStatusProcedure,
		Method:      "POST",
		Summary:     "Selected provider health",
		Description: "Resolves exactly one active provider's scenario route and reads its declared status/freshness endpoint when available; it does not sweep the federation or execute a search. Use this when selected-owner health can change the next action, then exercise the provider for the actual task.",
		Category:    "routing",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"provider_id": "string (required) — exact active provider selected for the pending task",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"provider_id": "string — exact provider probed",
			"health":      "ProviderHealth — live reachability, automatic eligibility and index freshness",
			"observed_at": "timestamp — time this selected-provider probe ran",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "provider_id is required"},
			{Status: 404, Code: "not_found", Description: "provider is not currently active"},
			{Status: 500, Code: "internal", Description: "Registry read failure"},
		},
		Examples: []module.Example{
			{Name: "Selected provider readiness", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.routing.RoutingService/ProviderStatus -H 'Content-Type: application/json' -d '{\"provider_id\":\"cli-health.commands\"}'"},
		},
	},
	{
		ID:          "routing_repromote",
		Path:        routingconnect.RoutingServiceRepromoteProcedure,
		Method:      "POST",
		Summary:     "Clear a provider's graded-empty demotion evidence",
		Description: "Explicit operator escape hatch for a verified or repaired provider. Clears zero-yield demotion evidence only; transport breaker state remains authoritative.",
		Category:    "routing",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"provider_id": "string (required) — registered active provider leaf",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"provider_id": "string", "reset": "bool", "message": "string",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "provider_id is required"},
			{Status: 500, Code: "internal", Description: "Provider is not active or registry read failed"},
		},
	},
}
