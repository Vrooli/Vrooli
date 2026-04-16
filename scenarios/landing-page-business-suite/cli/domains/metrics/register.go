package metrics

import (
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Metrics",
		Commands: deps.EndpointCommands([]support.EndpointDef{
			{Name: "metrics-track", Method: "POST", Path: "/metrics/track", Description: "Track metrics event"},
			{Name: "metrics-summary", Method: "GET", Path: "/metrics/summary", Description: "Get metrics summary (admin)"},
			{Name: "metrics-variants", Method: "GET", Path: "/metrics/variants", Description: "Get variant metrics (admin)"},
		}),
	}
}
