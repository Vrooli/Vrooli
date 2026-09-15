// Package signals is the CLI's signals-domain command surface. It
// mirrors the API's Connect-RPC SignalsService: per-signal scoring, the
// explainable aggregator verdict, and the registered-signal registry.
//
// Follows the graph-domain shape: Register(core, manifest) builds a
// cliapp.SubcommandGroup from cli/manifest.json via LoadFromManifest,
// with one handler per Connect-RPC subcommand in handlers.go.
package signals

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "signals"

// Register builds the signals subcommand group from the embedded CLI
// manifest and wires every SignalsService Connect-RPC binding to a
// handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SignalsService.ScoreChunk":     h.score,
		"SignalsService.ExplainVerdict": h.explain,
		"SignalsService.ListSignals":    h.list,
		"SignalsService.BoundaryHealth": h.boundaries,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("signals: load from manifest: %w", err)
	}
	return group, nil
}
