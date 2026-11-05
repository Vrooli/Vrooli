// Package models is the CLI's models-domain command surface. Mirrors the API's
// Connect-RPC ModelsService: the read + enable/disable verbs over the declarative
// model registry (list / get / operations / select / enable / blocklist).
//
// `models select` previews which model would run for an operation on this host
// without executing; `models enable` toggles the persisted overlay over the
// read-only seed catalog.
package models

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "models"

// Register builds the models subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ModelsService.ListModels":         h.list,
		"ModelsService.GetModel":           h.get,
		"ModelsService.ListOperations":     h.operations,
		"ModelsService.SelectModel":        h.selectModel,
		"ModelsService.ExplainResolution":  h.explain,
		"ModelsService.SetModelEnabled":    h.setEnabled,
		"ModelsService.ListBlocklist":      h.blocklist,
		"ModelsService.DoctorCatalog":      h.doctor,
		"ModelsService.InstallModel":       h.install,
		"ModelsService.RemoveModel":        h.remove,
		"ModelsService.AddCustomModel":     h.addCustom,
		"ModelsService.InspectModelSource": h.inspect,
		"ModelsService.ImportModel":        h.importModel,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("models: load from manifest: %w", err)
	}
	// `search` is a client-side convenience over ListModels. It reuses that RPC
	// rather than introducing a new one, so it can't be a manifest connect-rpc
	// binding (those are keyed by RPC method and `models list` already owns
	// ListModels). It is appended directly, mirroring the ai submit commands.
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "search",
		Description: "Find models by case-insensitive substring over id/name/operations",
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{
				{Name: "query", Required: true, Description: "Substring to match against id, name, or operations"},
			},
		},
		RunCtx: h.search,
	})
	return group, nil
}
