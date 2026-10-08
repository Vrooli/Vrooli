// Package impact is the CLI's proto contract-impact command surface.
package impact

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "impact"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ImpactService.GetImpact": h.getImpact,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("impact: load from manifest: %w", err)
	}
	return group, nil
}
