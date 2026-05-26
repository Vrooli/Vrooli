// Package restores is the CLI's restores-domain command surface. Mirrors the API's
// Connect-RPC RestoresService. Operators use restore/verify/get/list to manage
// restore and verification operations on backup snapshots.
//
// The manifest (cli/manifest.json) is the single source of truth for the command
// shape (flags, positionals, governance, RPC bindings); this package only wires
// bindings to handlers in handlers.go.
package restores

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "restores"

// Register builds the restores subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"RestoresService.RestoreTarget": h.restore,
		"RestoresService.VerifyTarget":  h.verify,
		"RestoresService.GetRestore":    h.get,
		"RestoresService.ListRestores":  h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("restores: load from manifest: %w", err)
	}
	return group, nil
}
