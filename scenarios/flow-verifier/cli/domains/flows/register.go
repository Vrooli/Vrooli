// Package flows is the CLI's flow-discovery/lifecycle command surface,
// a thin wrapper over the Connect-RPC FlowsService. Command surface loads
// from cli/manifest.json via cliapp.LoadFromManifest.
package flows

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "flows"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"FlowsService.ListFlows":           h.list,
		"FlowsService.ValidateFlow":        h.validate,
		"FlowsService.CreateFlow":          h.create,
		"FlowsService.ExplainFlow":         h.explain,
		"FlowsService.GetFlow":             h.show,
		"FlowsService.CodegenFlow":         h.codegen,
		"FlowsService.ReconcileFlow":       h.reconcile,
		"FlowsService.GetNavigationStudio": h.studio,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("flows: load from manifest: %w", err)
	}
	return group, nil
}
