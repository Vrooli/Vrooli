package analytics

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics/analytics_v1connect"
)

// Endpoints describes the analytics domain's Connect-RPC routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "analytics.list-events",
		Path:        analytics_v1connect.AnalyticsServiceListEventsProcedure,
		Method:      "POST",
		Summary:     "List append-only events",
		Description: "Cursor-paginated event log with optional kind + since filters.",
		Category:    "analytics",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart history"},
	},
	{
		ID:          "analytics.get-stats",
		Path:        analytics_v1connect.AnalyticsServiceGetStatsProcedure,
		Method:      "POST",
		Summary:     "Get stats roll-up",
		Description: "Returns counts + verdict success rate with N<5 suppression.",
		Category:    "analytics",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart stats"},
	},
	{
		ID:          "analytics.list-placements",
		Path:        analytics_v1connect.AnalyticsServiceListPlacementsProcedure,
		Method:      "POST",
		Summary:     "List placement outcomes",
		Description: "Cursor-paginated placement rows for a scenario.",
		Category:    "analytics",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart placements list"},
	},
	{
		ID:          "analytics.record-override",
		Path:        analytics_v1connect.AnalyticsServiceRecordOverrideProcedure,
		Method:      "POST",
		Summary:     "Record an operator override",
		Description: "Appends an Override row. Honors Idempotency-Key and X-Dry-Run.",
		Category:    "analytics",
		CLIMapping:  &module.CLIMapping{Command: "arch-cart override record"},
	},
}
