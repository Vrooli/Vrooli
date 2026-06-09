// Package research is the CLI's research-domain command surface. It mirrors the
// API's Connect-RPC ResearchService one command per RPC, built from
// cli/manifest.json via cliapp.LoadFromManifest (the single source of truth for
// the command-line shape) with one handler per subcommand in handlers.go.
package research

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "research"

// Register builds the research subcommand group from the embedded manifest and
// wires every Connect-RPC binding to a handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ResearchService.RunL2":             h.l2,
		"ResearchService.RunL3":             h.l3,
		"ResearchService.GetResearchStatus": h.status,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("research: load from manifest: %w", err)
	}
	return group, nil
}
