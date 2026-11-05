package shortcuts

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "shortcuts"

// Register builds the `shortcuts` subcommand group covering the effective
// shortcut view and per-profile CRUD from the embedded manifest and wires
// Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ShortcutsService.GetEffective":  h.effective,
		"ShortcutsService.ListProfiles":  h.list,
		"ShortcutsService.UpsertProfile": h.upsert,
		"ShortcutsService.DeleteProfile": h.delete,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("shortcuts: load from manifest: %w", err)
	}
	return group, nil
}
