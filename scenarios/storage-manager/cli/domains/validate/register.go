package validate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "validate"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.ValidateScenario": h.validateScenario,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "prove-isolation",
		Description: "Prove routed storage isolation without starting the target API",
		NeedsAPI:    false,
		Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true, Description: "Scenario id"}}},
		RunCtx:      h.proveIsolation,
	})
	return group, nil
}
