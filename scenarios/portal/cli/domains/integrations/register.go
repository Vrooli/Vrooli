package integrations

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "integrations"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"IntegrationsService.Status":         h.status,
		"IntegrationsService.UpdateOverride": h.override,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("integrations: load from manifest: %w", err)
	}
	return group, nil
}
