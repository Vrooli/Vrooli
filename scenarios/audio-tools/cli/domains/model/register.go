// Package model hosts the `audio-tools model ...` subtree, mirroring
// vrooli.audio_tools.v1.models.ModelRegistryService.
//
// The command surface is declared in cli/manifest.json — the single
// source of truth. Register loads the "model" group and wires each
// binding to a handler in handlers.go.
package model

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "model"

// Register builds the model subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, modelBindings(h))
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("model: load from manifest: %w", err)
	}
	return group, nil
}

// modelBindings is shared with the coverage test so the manifest and the
// handler registry cannot drift apart.
func modelBindings(h *handlers) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"ModelRegistryService.ListModels":   cliapp.ProtoList(h.listCall, h.listReport),
		"ModelRegistryService.ResolveModel": cliapp.ProtoList(h.resolveCall, h.resolveReport),
	}
}
