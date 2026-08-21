// Package routes is the CLI's routes-domain command surface. Mirrors the
// API's Connect-RPC RoutesService and the UI's api/routes.ts client.
//
// Follows the canonical domain shape: a Register(core, manifest) returning
// a cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go. The manifest is the single source of truth for the
// command-line shape.
package routes

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "routes"

// Register builds the routes subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"RoutesService.ListRoutes":  cliapp.ProtoList(h.listCall, h.listReport),
		"RoutesService.GetRoute":    cliapp.ProtoList(h.getCall, h.getReport),
		"RoutesService.CreateRoute": cliapp.ProtoMutation(h.createCall, h.createReport),
		"RoutesService.UpdateRoute": cliapp.ProtoMutation(h.updateCall, h.updateReport),
		"RoutesService.DeleteRoute": cliapp.ProtoMutation(h.deleteCall, h.deleteReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("routes: load from manifest: %w", err)
	}
	return group, nil
}
