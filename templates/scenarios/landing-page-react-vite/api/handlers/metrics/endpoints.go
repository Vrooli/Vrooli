package metrics

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the metrics module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "metrics_track_event", Path: landingconnect.MetricsServiceTrackEventProcedure, Method: "POST",
		Summary: "Track event", Description: "Records a single analytics event (public, idempotent).", Category: "metrics",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"event_type": "string", "variant_id": "int64", "session_id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"success": "bool", "message": "string"}},
	},
	{
		ID: "metrics_get_analytics_summary", Path: landingconnect.MetricsServiceGetAnalyticsSummaryProcedure, Method: "POST",
		Summary: "Get analytics summary", Description: "Returns the aggregate analytics summary for a window (admin).", Category: "metrics",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"start_date": "string", "end_date": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"total_visitors": "int64", "variant_stats": "VariantStats[]"}},
	},
	{
		ID: "metrics_get_variant_stats", Path: landingconnect.MetricsServiceGetVariantStatsProcedure, Method: "POST",
		Summary: "Get variant stats", Description: "Returns per-variant funnel stats for a window (admin).", Category: "metrics",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"start_date": "string", "end_date": "string", "variant": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"stats": "VariantStats[]"}},
	},
}
