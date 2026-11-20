// Package model hosts the `audio-tools model ...` subtree, mirroring
// vrooli.audio_tools.v1.models.ModelRegistryService.
//
// The command surface is declared in cli/manifest.json — the single
// source of truth. Register loads the "model" group and wires each
// binding to a handler in handlers.go.
package model

import (
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "model"

// Register builds the model subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	return climanifest.LoadGroup(manifest, GroupName, modelBindings(newHandlers(core)))
}

// modelBindings is shared with the coverage test so the manifest and the
// handler registry cannot drift apart.
func modelBindings(h *handlers) map[string]func(cliapp.RunContext) error {
	return map[string]func(cliapp.RunContext) error{
		"ModelRegistryService.ListModels":   h.list,
		"ModelRegistryService.ResolveModel": h.resolve,
	}
}
