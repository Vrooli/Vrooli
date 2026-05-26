// Package manifest is the CLI's manifest-domain command surface. It
// mirrors the API's Connect-RPC ManifestService: parse + validate the
// per-scenario architecture manifest, read the persisted definition, and
// list its declared domains.
//
// Follows the graph-domain shape: Register(core, manifest) builds a
// cliapp.SubcommandGroup from cli/manifest.json via LoadFromManifest,
// with one handler per Connect-RPC subcommand in handlers.go.
package manifest

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "manifest"

// Register builds the manifest subcommand group from the embedded CLI
// manifest and wires every ManifestService Connect-RPC binding to a
// handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ManifestService.ValidateManifest": h.validate,
		"ManifestService.GetManifest":      h.show,
		"ManifestService.ListDomains":      h.listDomains,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("manifest: load from manifest: %w", err)
	}
	return group, nil
}
