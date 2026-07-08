// Package insights is the CLI's metrics-insights command surface. It mirrors
// the API's MetricsService.Insights Connect-RPC: per-query telemetry aggregated
// into federation-health signals (utilization, zero-result rate, latency
// percentiles).
//
// The command-line shape is declared in cli/manifest.json (the single source of
// truth) and loaded via cliapp.LoadFromManifestPrimitives; the handler lives in
// handlers.go. The insights verb is the group default so `search-hub insights`
// is shorthand for `search-hub insights insights`.
package insights

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "insights"

// Register builds the insights subcommand group from the embedded manifest and
// wires the MetricsService.Insights binding to the handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"MetricsService.Insights": cliapp.ProtoList(h.insightsCall, h.insightsReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("insights: load from manifest: %w", err)
	}
	group.DefaultSubcommand = "insights"
	return group, nil
}
