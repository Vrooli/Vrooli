// Package handoffrules is the CLI's capture-rule command surface. It mirrors
// the API's Connect-RPC HandoffRulesService: list, upsert, and an idempotent
// delete over the patterns that decide when a handoff is suggested.
//
// Nothing in this package can send a handoff. A rule only ever produces a
// suggestion the operator chooses to act on.
package handoffrules

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "handoff-rule"

// Register builds the `handoff-rule` subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"HandoffRulesService.ListRules":  h.list,
		"HandoffRulesService.UpsertRule": h.upsert,
		"HandoffRulesService.DeleteRule": h.delete,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("handoff-rule: load from manifest: %w", err)
	}
	return group, nil
}
