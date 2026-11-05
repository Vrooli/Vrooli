package capture

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "capture"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"capture": h.run,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("capture: load from manifest: %w", err)
	}
	return group, nil
}
