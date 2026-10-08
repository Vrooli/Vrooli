// Package graph is the CLI's graph-domain command surface. It mirrors
// the API's Connect-RPC GraphService.ExtractGraph method.
//
// New domain packages copy this shape: a Register(core, manifest)
// returning a cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand
// in handlers.go. The manifest carries the declarative surface
// (governance, flags, positionals, RPC bindings) and is the SINGLE
// source of truth for the command-line shape.
package graph

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "graph"

// Register builds the graph subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"GraphService.ExtractGraph":             h.extract,
		"GraphService.PreviewSnapshotRetention": h.retentionPreview,
		"GraphService.ApplySnapshotRetention":   h.retentionApply,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("graph: load from manifest: %w", err)
	}
	return group, nil
}
