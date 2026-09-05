// Package describe is the CLI's proto surface fact command surface.
package describe

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "describe"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ProtoHealthService.DescribeScenarioProtos": h.describeScenario,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("describe: load from manifest: %w", err)
	}
	return group, nil
}
