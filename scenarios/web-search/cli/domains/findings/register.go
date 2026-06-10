// Package findings is the CLI's findings-domain command surface. It mirrors
// the API's Connect-RPC FindingsService one command per RPC, built from
// cli/manifest.json via cliapp.LoadFromManifest (the single source of truth for
// the command-line shape) with one handler per subcommand in handlers.go.
package findings

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "findings"

// Register builds the findings subcommand group from the embedded manifest and
// wires every Connect-RPC binding to a handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"FindingsService.ListFindings":      h.list,
		"FindingsService.GetFinding":        h.get,
		"FindingsService.AddFinding":        h.add,
		"FindingsService.EditFinding":       h.edit,
		"FindingsService.SupersedeFinding":  h.supersede,
		"FindingsService.FlagFinding":       h.flag,
		"FindingsService.PruneFindings":     h.prune,
		"FindingsService.SearchFindings":    h.search,
		"FindingsService.CountFindings":     h.count,
		"FindingsService.ListEffectiveness": h.effectiveness,
		"FindingsService.RecordUsage":       h.use,
		"FindingsService.RunGC":             h.gc,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("findings: load from manifest: %w", err)
	}
	return group, nil
}
