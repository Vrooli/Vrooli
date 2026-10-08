package capabilities

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "capabilities"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{"CapabilityStatusService.GetStatus": h.status})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("capabilities: load from manifest: %w", err)
	}
	return group, nil
}
