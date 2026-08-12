package metrics

import (
	"search-hub/internal/module"

	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/measures/measures_v1connect"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics/metrics_v1connect"
)

// Endpoints is the machine-readable description of the metrics module's public
// surface. The Connect-RPC method path references the generated *Procedure
// constant, so adding or renaming an RPC in metrics.proto breaks this file at
// compile time. The global parity test (TestProtoConnectParity in
// api/internal/modules/registry_test.go) walks the proto FileDescriptor and
// asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "metrics_insights",
		Path:        metricsconnect.MetricsServiceInsightsProcedure,
		Method:      "POST",
		Summary:     "Federation telemetry insights",
		Description: "Aggregates per-query telemetry over an optional recent window into federation-health signals: total/zero-result/degraded/reranked query counts, zero-result rate, p50/p95 latency, resolver-cache hit rate, report-only registry hygiene, and per-provider utilization (including registered-but-never-routed leaves flagged under_utilized). The router records one hashed-query telemetry row per Query; this is the read side.",
		Category:    "metrics",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"window_days": "int32 — restrict aggregates to the last N days (0 = all-time)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"total_queries":           "int64 — queries recorded in the window",
				"zero_result_queries":     "int64 — queries that returned no results",
				"zero_result_rate":        "double — zero_result_queries / total_queries",
				"degraded_queries":        "int64 — queries where a provider degraded",
				"reranked_queries":        "int64 — queries that produced a unified rerank",
				"latency_p50_ms":          "int64 — median latency",
				"latency_p95_ms":          "int64 — p95 latency",
				"resolver_cache_hits":     "int64 — successful address-cache lookups",
				"resolver_cache_misses":   "int64 — address-cache misses",
				"resolver_cache_hit_rate": "double — hits / (hits + misses)",
				"retirement_candidates":   "array<ProviderRetirementCandidate> — report-only zero-yield candidates",
				"group_advisories":        "array<ProviderGroupAdvisory> — report-only concentration warnings",
				"providers":               "array<ProviderUtilization> — per-provider routed/hit totals + under_utilized flag",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Telemetry or registry read failure"},
		},
		Examples: []module.Example{
			{Name: "All-time insights", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.metrics.MetricsService/Insights -H 'Content-Type: application/json' -d '{}'"},
			{Name: "Last 7 days", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.metrics.MetricsService/Insights -H 'Content-Type: application/json' -d '{\"window_days\":7}'"},
		},
	},
	{
		ID:          "metrics_measures_declarations",
		Path:        "/measures/declarations",
		Method:      "GET",
		Summary:     "List declared search-hub metrics measures",
		Description: "Measures-go registry declarations harvested by measures-health and the central measures index.",
		Category:    "measures",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "measures-go serves a framework-neutral harvest endpoint consumed without a generated client.",
		},
	},
	{
		ID:          "metrics_measure_federated_latency",
		Path:        measuresconnect.MeasuresServiceFederatedLatencyProcedure,
		Method:      "POST",
		Summary:     "Federated latency measure",
		Description: "Returns p95 and p50 federated query latency over a canonical time window, backed by query_telemetry.",
		Category:    "measures",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"window": "vrooli.measures.v1.TimeWindow (optional; defaults to this_week)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"p95_ms": "int64 — p95 federated query latency",
				"p50_ms": "int64 — p50 federated query latency",
			},
		},
	},
	{
		ID:          "metrics_measure_degraded_query_rate",
		Path:        measuresconnect.MeasuresServiceDegradedQueryRateProcedure,
		Method:      "POST",
		Summary:     "Degraded query rate measure",
		Description: "Returns degraded_queries / total_queries over a canonical time window.",
		Category:    "measures",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"window": "vrooli.measures.v1.TimeWindow (optional; defaults to this_week)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"rate":             "double — degraded query rate",
				"degraded_queries": "int64 — degraded query count",
				"total_queries":    "int64 — total query count",
			},
		},
	},
	{
		ID:          "metrics_measure_provider_degradation_rate",
		Path:        measuresconnect.MeasuresServiceProviderDegradationRateProcedure,
		Method:      "POST",
		Summary:     "Provider degradation rate measure",
		Description: "Returns degraded provider legs / routed provider legs over a canonical time window, optionally scoped to provider_id.",
		Category:    "measures",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"window":      "vrooli.measures.v1.TimeWindow (optional; defaults to this_week)",
				"provider_id": "string — optional provider id scope",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"rate":           "double — provider degradation rate",
				"degraded_count": "int64 — degraded provider-leg count",
				"times_routed":   "int64 — routed provider-leg count",
			},
		},
	},
	{
		ID:          "metrics_measures_execute",
		Path:        "/measures/execute",
		Method:      "POST",
		Summary:     "Execute a declared search-hub metrics measure",
		Description: "Measures-go registry execution endpoint used by measures-health behavioral probes and search-hub federation.",
		Category:    "measures",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "measures-go serves a uniform JSON execution endpoint shared across scenarios.",
		},
	},
}
