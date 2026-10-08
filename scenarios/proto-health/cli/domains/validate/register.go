// Package validate is the CLI's proto validation command surface.
package validate

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "validate"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	filteredManifest, err := manifestForRPCCommands(manifest)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: filter local commands: %w", err)
	}
	bindings := map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.ValidateScenario": h.validateScenario,
	}
	group, err := cliapp.LoadFromManifest(filteredManifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: load from manifest: %w", err)
	}
	owner := cliapp.Command{
		Name:             "owner",
		Description:      "Validate one schema owner directly from its source files",
		NeedsAPI:         false,
		NeedsAPIOverride: func() *bool { value := false; return &value }(),
		Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{
			Name:        "owner",
			Description: "Schema owner under packages/proto/schemas/",
			Required:    true,
		}}},
		RunCtx: h.validateOwner,
	}
	group.Subcommands = append(group.Subcommands, owner)
	return group, nil
}

// manifestForRPCCommands leaves local commands discoverable in the published
// manifest while keeping them out of cli-core's Connect binding loader. Local
// commands are appended with their native handlers after the RPC commands load.
func manifestForRPCCommands(raw []byte) ([]byte, error) {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	groups, ok := document["groups"].([]any)
	if !ok {
		return raw, nil
	}
	for _, rawGroup := range groups {
		group, ok := rawGroup.(map[string]any)
		if !ok || group["name"] != GroupName {
			continue
		}
		commands, ok := group["commands"].([]any)
		if !ok {
			continue
		}
		filtered := make([]any, 0, len(commands))
		for _, rawCommand := range commands {
			command, ok := rawCommand.(map[string]any)
			if !ok {
				filtered = append(filtered, rawCommand)
				continue
			}
			binding, _ := command["binding"].(map[string]any)
			if binding["kind"] == "local" {
				continue
			}
			filtered = append(filtered, rawCommand)
		}
		group["commands"] = filtered
	}
	return json.Marshal(document)
}
