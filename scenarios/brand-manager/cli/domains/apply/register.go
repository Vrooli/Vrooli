// Package apply is the CLI's apply-domain command surface. Mirrors the API's
// Connect-RPC ApplyService and the UI's api/apply.ts client.
//
// The manifest (cli/manifest.json) carries the declarative surface (governance,
// flags, RPC bindings) and is the SINGLE source of truth for the command-line
// shape; handlers in handlers.go are wired via the bindings map.
package apply

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "apply"

// Register builds the apply subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ApplyService.PreviewApply": h.preview,
		"ApplyService.ApplyBrand":   h.run,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("apply: load from manifest: %w", err)
	}
	return group, nil
}
