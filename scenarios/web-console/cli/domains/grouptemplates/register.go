// Package grouptemplates is the CLI's group-template command surface. It
// mirrors the API's Connect-RPC GroupTemplatesService: list, upsert, and an
// idempotent delete over saved role recipes.
package grouptemplates

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "group-template"

// Register builds the `group-template` subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"GroupTemplatesService.ListTemplates":  h.list,
		"GroupTemplatesService.UpsertTemplate": h.upsert,
		"GroupTemplatesService.DeleteTemplate": h.delete,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("group-template: load from manifest: %w", err)
	}
	return group, nil
}
