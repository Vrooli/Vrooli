// Package conflicts is the CLI's conflicts-domain command surface — the
// agent-facing conflict workbench. It mirrors the API's Connect-RPC
// ConflictsService: drift detection, the per-conflict lifecycle state
// machine, and the Detector / Resolver registries.
//
// Like every domain package, it follows the graph-domain shape: a
// Register(core, manifest) returning a cliapp.SubcommandGroup built from
// cli/manifest.json via cliapp.LoadFromManifest, plus one handler per
// Connect-RPC subcommand in handlers.go. The manifest is the single
// source of truth for the command-line shape (governance, flags,
// positionals, RPC bindings).
package conflicts

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "conflicts"

// Register builds the conflicts subcommand group from the embedded
// manifest and wires every ConflictsService Connect-RPC binding to a
// handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ConflictsService.DetectConflicts":   h.detect,
		"ConflictsService.ListConflicts":     h.list,
		"ConflictsService.GetConflict":       h.show,
		"ConflictsService.AssignConflict":    h.assign,
		"ConflictsService.ResolveConflict":   h.resolve,
		"ConflictsService.ReopenConflict":    h.reopen,
		"ConflictsService.ValidateConflicts": h.validate,
		"ConflictsService.ListDetectors":     h.detectors,
		"ConflictsService.ListResolvers":     h.resolvers,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("conflicts: load from manifest: %w", err)
	}
	return group, nil
}
