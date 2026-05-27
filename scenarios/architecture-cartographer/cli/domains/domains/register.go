// Package domains is the CLI's domains-domain command surface. It mirrors
// the API's Connect-RPC DomainsService: derive and show a scenario's
// intended domain map from its on-disk sources, with zero per-scenario
// configuration.
//
// Follows the graph-domain shape: Register(core, manifest) builds a
// cliapp.SubcommandGroup from cli/manifest.json via LoadFromManifest, with
// one handler per Connect-RPC subcommand in handlers.go.
package domains

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "domains"

// Register builds the domains subcommand group from the embedded CLI
// manifest and wires every DomainsService Connect-RPC binding to a handler
// in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DomainsService.ExtractDomains":    h.extract,
		"DomainsService.GetDomainMap":      h.show,
		"DomainsService.ConvergenceReport": h.convergence,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("domains: load from manifest: %w", err)
	}
	return group, nil
}
