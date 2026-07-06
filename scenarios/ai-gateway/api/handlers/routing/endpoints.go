package routing

import (
	"ai-gateway/internal/module"

	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "routing_preview_route",
		Path:        routingconnect.RoutingServicePreviewRouteProcedure,
		Method:      "POST",
		Summary:     "Preview routing policy for a gateway request",
		Description: "Validates the gateway request and returns deterministic provider candidates, selected route, fallback eligibility, and policy reasons without running inference.",
		Category:    "routing",
		Request:     &module.Schema{Type: "PreviewRouteRequest", Properties: map[string]string{"request": "GatewayRequest"}},
		Response:    &module.Schema{Type: "PreviewRouteResponse", Properties: map[string]string{"valid": "bool", "issues": "array<ValidationIssue>", "candidates": "array<RouteCandidate>", "policy_reasons": "array<string>", "fallback_allowed": "bool", "route_plan_id": "string"}},
	},
	{
		ID:          "routing_execute_route",
		Path:        routingconnect.RoutingServiceExecuteRouteProcedure,
		Method:      "POST",
		Summary:     "Execute a routed gateway request",
		Description: "Executes a provider-neutral request through the selected resource command and persists redacted route evidence before returning provider output.",
		Category:    "routing",
		Request:     &module.Schema{Type: "ExecuteRouteRequest", Properties: map[string]string{"request": "GatewayRequest", "input_text": "string"}},
		Response:    &module.Schema{Type: "ExecuteRouteResponse", Properties: map[string]string{"valid": "bool", "issues": "array<ValidationIssue>", "evidence": "RouteEvidence", "output_text": "string", "policy_reasons": "array<string>"}},
	},
	{
		ID:          "routing_list_route_evidence",
		Path:        routingconnect.RoutingServiceListRouteEvidenceProcedure,
		Method:      "POST",
		Summary:     "List route evidence events",
		Description: "Lists redacted route evidence metadata for recent gateway executions.",
		Category:    "routing",
		Request:     &module.Schema{Type: "ListRouteEvidenceRequest", Properties: map[string]string{"limit": "int32", "scenario": "string"}},
		Response:    &module.Schema{Type: "ListRouteEvidenceResponse", Properties: map[string]string{"events": "array<RouteEvidence>"}},
	},
	{
		ID:          "routing_get_route_evidence",
		Path:        routingconnect.RoutingServiceGetRouteEvidenceProcedure,
		Method:      "POST",
		Summary:     "Inspect one route evidence event",
		Description: "Returns one redacted route evidence metadata event by ID.",
		Category:    "routing",
		Request:     &module.Schema{Type: "GetRouteEvidenceRequest", Properties: map[string]string{"event_id": "string"}},
		Response:    &module.Schema{Type: "GetRouteEvidenceResponse", Properties: map[string]string{"event": "RouteEvidence"}},
	},
	{
		ID:          "routing_list_provider_health",
		Path:        routingconnect.RoutingServiceListProviderHealthProcedure,
		Method:      "POST",
		Summary:     "List provider circuit-breaker health",
		Description: "Returns persisted provider circuit-breaker state per (provider, role, kind) with the effective state at read time, so operators can see which providers are suppressed and why without reading logs or database rows.",
		Category:    "routing",
		Request:     &module.Schema{Type: "ListProviderHealthRequest", Properties: map[string]string{}},
		Response:    &module.Schema{Type: "ListProviderHealthResponse", Properties: map[string]string{"items": "array<ProviderHealth>"}},
	},
}
