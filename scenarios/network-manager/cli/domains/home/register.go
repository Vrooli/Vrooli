package home

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "home"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"HomeIntegrationService.ListActions":      h.actions,
		"HomeIntegrationService.InvokeAction":     h.invoke,
		"HomeIntegrationService.ListRecentEvents": h.events,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("home: load from manifest: %w", err)
	}
	return group, nil
}
