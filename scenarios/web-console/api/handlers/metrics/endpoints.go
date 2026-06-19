// Package metrics exposes the canonical metadata for the MetricsService
// Connect-RPC endpoint. gen-endpoints consumes this slice to keep
// `.vrooli/endpoints.json` in sync with the wire surface.
package metrics

import (
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics/metrics_v1connect"

	"web-console/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "metrics_get",
		Path:        metricsconnect.MetricsServiceGetProcedure,
		Method:      "POST",
		Summary:     "Get operational metrics snapshot",
		Description: "Returns a point-in-time snapshot of all operational counters (sessions, connections, messages, reattach, recovery, AI, voice) plus uptime.",
		Category:    "system",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"sessions":                      "SessionMetrics",
				"connections":                   "ConnectionMetrics",
				"messages":                      "MessageMetrics",
				"reattach":                      "ReattachMetrics",
				"recovery":                      "RecoveryMetrics",
				"ai_generations":                "int64",
				"ai_suggestions":                "int64",
				"stdin_before_ready_total":      "int64",
				"voice_skip_verification_total": "int64",
				"uptime":                        "string",
			},
		},
	},
}
