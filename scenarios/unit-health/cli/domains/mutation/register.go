package mutation

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "mutation"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ValidationService.RunMutationPilot": h.pilot,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("mutation: load from manifest: %w", err)
	}
	return group, nil
}
