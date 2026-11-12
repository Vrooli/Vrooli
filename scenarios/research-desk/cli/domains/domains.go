package domains

import (
	"research-desk/cli/domains/activity"
	"research-desk/cli/domains/launch"
	"research-desk/cli/domains/operator"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups returns each API domain's commands. Every domain group in
// cli/manifest.json is flat: commands dispatch at the top level
// (`research-desk now`), one group per domain.
func CommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.CommandGroup, error) {
	activityGroup, err := activity.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	launchGroup, err := launch.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	operatorGroup, err := operator.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	return []cliapp.CommandGroup{activityGroup, launchGroup, operatorGroup}, nil
}

// TopLevel flattens the groups' commands for the primitive-evidence export.
func TopLevel(groups []cliapp.CommandGroup) []cliapp.Command {
	var out []cliapp.Command
	for _, g := range groups {
		out = append(out, g.Commands...)
	}
	return out
}
