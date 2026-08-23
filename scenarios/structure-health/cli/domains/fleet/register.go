package fleet

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this domain binds to.
const GroupName = "fleet"

// Register wires the `fleet` group's manifest commands to their handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	parsed, err := cliapp.ParseManifest(manifest)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fleet: parse manifest: %w", err)
	}
	manifestGroup := parsed.FindGroup(GroupName)
	if manifestGroup == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fleet: manifest group %q is missing", GroupName)
	}
	group := cliapp.SubcommandGroup{Name: GroupName, Description: manifestGroup.Description}
	for _, declaration := range manifestGroup.Commands {
		var run func(cliapp.RunContext) error
		needsAPI := false
		switch declaration.Name {
		case "scan":
			run = h.scan
			needsAPI = true
		case "census":
			run = h.census
		default:
			return cliapp.SubcommandGroup{}, fmt.Errorf("fleet: unknown command %q", declaration.Name)
		}
		args, err := cliapp.ManifestArgs(declaration)
		if err != nil {
			return cliapp.SubcommandGroup{}, fmt.Errorf("fleet: command %q: %w", declaration.Name, err)
		}
		group.Subcommands = append(group.Subcommands, cliapp.Command{
			Name: declaration.Name, Description: declaration.Description, NeedsAPI: needsAPI,
			Args: args, RunCtx: run, Architecture: declaration.Architecture.CommandArchitecture(),
		})
	}
	return group, nil
}
