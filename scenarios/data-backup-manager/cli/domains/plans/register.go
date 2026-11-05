// Package plans is the CLI's plans-domain command surface. Mirrors the API's
// Connect-RPC PlansService. Operators use create/get/list/update/delete to manage
// backup plans that bind targets to destinations on a schedule.
//
// The manifest (cli/manifest.json) is the single source of truth for the command
// shape (flags, positionals, governance, RPC bindings); this package only wires
// bindings to handlers in handlers.go.
package plans

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "plans"

// Register builds the plans subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"PlansService.CreatePlan": h.create,
		"PlansService.GetPlan":    h.get,
		"PlansService.ListPlans":  h.list,
		"PlansService.UpdatePlan": h.update,
		"PlansService.DeletePlan": h.delete,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("plans: load from manifest: %w", err)
	}
	return group, nil
}
