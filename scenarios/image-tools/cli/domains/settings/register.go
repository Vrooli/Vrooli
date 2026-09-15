// Package settings is the CLI's settings-domain command surface. It exposes the
// per-operation default-model controls on the API's ModelsService:
//   - `settings list`         → ListDefaults (effective default per operation)
//   - `settings set-default`  → SetDefaultModel (pin an operator override)
//   - `settings clear-default`→ SetDefaultModel with an empty model_id (revert to seed)
//
// `clear-default` reuses SetDefaultModel rather than introducing a new RPC, so
// it can't be a manifest connect-rpc binding (those are keyed by RPC method and
// `set-default` already owns SetDefaultModel). It is appended directly.
package settings

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "settings"

// Register builds the settings subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ModelsService.ListDefaults":    h.list,
		"ModelsService.SetDefaultModel": h.setDefault,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("settings: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "clear-default",
		Description: "Clear the operator pin for an operation (revert to the seed default)",
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{
				{Name: "operation", Required: true, Description: "Operation whose default pin to clear"},
			},
		},
		RunCtx: h.clearDefault,
	})
	return group, nil
}
