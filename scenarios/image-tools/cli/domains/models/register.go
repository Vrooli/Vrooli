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
		"ModelsService.ListModels":      h.list,
		"ModelsService.GetModel":        h.get,
		"ModelsService.ListOperations":  h.operations,
		"ModelsService.SelectModel":     h.selectModel,
		"ModelsService.SetModelEnabled": h.setEnabled,
		"ModelsService.ListBlocklist":   h.blocklist,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("models: load from manifest: %w", err)
	}
	return group, nil
}
