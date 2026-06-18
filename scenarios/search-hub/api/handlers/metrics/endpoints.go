package metrics

import (
	"search-hub/internal/module"

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
		Description: "Aggregates per-query telemetry over an optional recent window into federation-health signals: total/zero-result/degraded/reranked query counts, zero-result rate, p50/p95 latency, and per-provider utilization (including registered-but-never-routed leaves flagged under_utilized). The router records one hashed-query telemetry row per Query; this is the read side.",
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
				"total_queries":       "int64 — queries recorded in the window",
				"zero_result_queries": "int64 — queries that returned no results",
				"zero_result_rate":    "double — zero_result_queries / total_queries",
				"degraded_queries":    "int64 — queries where a provider degraded",
				"reranked_queries":    "int64 — queries that produced a unified rerank",
				"latency_p50_ms":      "int64 — median latency",
				"latency_p95_ms":      "int64 — p95 latency",
				"providers":           "array<ProviderUtilization> — per-provider routed/hit totals + under_utilized flag",
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
}
