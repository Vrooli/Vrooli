// Package golden is the CLI's golden-domain command surface. Mirrors
// the API's Connect-RPC GoldenService.
//
// Subcommand shape (positionals, flags, governance, RPC binding) is loaded
// from cli/manifest.json via cliapp.LoadFromManifest; handlers live in
// handlers.go and are wired via the bindings map below.
package golden

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "goldens"

// Register builds the goldens subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"GoldenService.ListGoldens":      h.list,
		"GoldenService.GetGolden":        h.get,
		"GoldenService.RegisterGolden":   h.register,
		"GoldenService.UpdateGolden":     h.update,
		"GoldenService.DeleteGolden":     h.delete,
		"GoldenService.RegenerateGolden": h.regenerate,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("golden: load from manifest: %w", err)
	}
	return group, nil
}
