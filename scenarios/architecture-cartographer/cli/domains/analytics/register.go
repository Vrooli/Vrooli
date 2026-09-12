// Package analytics is the CLI's analytics-domain command surface. It
// mirrors the API's Connect-RPC AnalyticsService: the append-only event
// log, the threshold-suppressed stats roll-up, placement outcomes, and
// operator overrides.
//
// Follows the graph-domain shape: Register(core, manifest) builds a
// cliapp.SubcommandGroup from cli/manifest.json via LoadFromManifest,
// with one handler per Connect-RPC subcommand in handlers.go.
package analytics

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "analytics"

// Register builds the analytics subcommand group from the embedded CLI
// manifest and wires every AnalyticsService Connect-RPC binding to a
// handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AnalyticsService.ListEvents":     h.events,
		"AnalyticsService.GetStats":       h.stats,
		"AnalyticsService.ListPlacements": h.placements,
		"AnalyticsService.RecordOverride": h.overrideRecord,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("analytics: load from manifest: %w", err)
	}
	return group, nil
}
