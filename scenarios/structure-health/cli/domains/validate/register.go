package validate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this domain binds to.
const GroupName = "validate"

// Register wires the `validate` group's manifest commands to their handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	parsed, err := cliapp.ParseManifest(manifest)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: parse manifest: %w", err)
	}
	manifestGroup := parsed.FindGroup(GroupName)
	if manifestGroup == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: manifest group %q is missing", GroupName)
	}
	group := cliapp.SubcommandGroup{Name: GroupName, Description: manifestGroup.Description, NeedsAPI: true}
	for _, declaration := range manifestGroup.Commands {
		var run func(cliapp.RunContext) error
		switch declaration.Name {
		case "scenario":
			run = h.validateScenario
		case "all":
			run = h.validateAll
		default:
			run = h.validateTarget(declaration.Name)
		}
		args, err := cliapp.ManifestArgs(declaration)
		if err != nil {
			return cliapp.SubcommandGroup{}, fmt.Errorf("validate: command %q: %w", declaration.Name, err)
		}
		group.Subcommands = append(group.Subcommands, cliapp.Command{
			Name: declaration.Name, Description: declaration.Description, NeedsAPI: true,
			Args: args, RunCtx: run, Architecture: declaration.Architecture.CommandArchitecture(),
		})
	}
	return group, nil
}
