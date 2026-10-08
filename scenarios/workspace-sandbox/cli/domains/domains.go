package domains

import (
	"workspace-sandbox/cli/domains/changes"
	"workspace-sandbox/cli/domains/maintenance"
	"workspace-sandbox/cli/domains/process"
	"workspace-sandbox/cli/domains/provenance"
	"workspace-sandbox/cli/domains/sandbox"
	"workspace-sandbox/cli/domains/typed"
	"workspace-sandbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The root /health probe is
// served by cli-core's built-in `status` command, so no status/health/system
// domain is registered here.
func CommandGroups(_ support.Dependencies) []cliapp.CommandGroup {
	return nil
}

func SubcommandGroups(deps support.Dependencies, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	typedGroups, err := typed.Register(deps.ScenarioApp(), manifest)
	if err != nil {
		return nil, err
	}
	legacySandbox := sandbox.Register(deps)
	legacyChanges := changes.Register(deps)
	legacySandbox.Subcommands = filterCommands(legacySandbox.Subcommands, "create")
	legacyChanges.Subcommands = filterCommands(legacyChanges.Subcommands, "diff")
	typedGroups[0].Subcommands = append(typedGroups[0].Subcommands, legacySandbox.Subcommands...)
	typedGroups[1].Subcommands = append(typedGroups[1].Subcommands, legacyChanges.Subcommands...)
	return append(typedGroups,
		process.Register(deps),
		maintenance.Register(deps),
		provenance.Register(deps),
	), nil
}

func filterCommands(commands []cliapp.Command, excluded ...string) []cliapp.Command {
	result := make([]cliapp.Command, 0, len(commands))
	for _, command := range commands {
		skip := false
		for _, name := range excluded {
			if command.Name == name {
				skip = true
				break
			}
		}
		if !skip {
			result = append(result, command)
		}
	}
	return result
}
