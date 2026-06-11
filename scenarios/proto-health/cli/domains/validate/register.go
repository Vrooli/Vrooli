// Package validate is the CLI's proto validation command surface.
package validate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "validate"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ProtoHealthService.ValidateScenario": h.validateScenario,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: load from manifest: %w", err)
	}
	return group, nil
}
