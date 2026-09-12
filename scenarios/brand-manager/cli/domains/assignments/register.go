// Package assignments is the CLI's assignments-domain command surface. Mirrors
// the API's Connect-RPC AssignmentsService and the UI's api/assignments.ts
// client.
//
// The manifest (cli/manifest.json) carries the declarative surface (governance,
// flags, positionals, RPC bindings) and is the SINGLE source of truth for the
// command-line shape; handlers in handlers.go are wired via the bindings map.
package assignments

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "assignments"

// Register builds the assignments subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AssignmentsService.ListAssignments":   h.list,
		"AssignmentsService.AssignBrand":       h.assign,
		"AssignmentsService.GetScenarioStatus": h.status,
		"AssignmentsService.UnassignScenario":  h.unassign,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("assignments: load from manifest: %w", err)
	}
	return group, nil
}
