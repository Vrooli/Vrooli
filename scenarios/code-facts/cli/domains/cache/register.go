package cache

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "cache"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CodeFactsService.GetCacheStatus": h.status,
		"CodeFactsService.InspectCache":   h.inspect,
		"CodeFactsService.ClearCache":     h.clear,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("cache: load from manifest: %w", err)
	}
	return group, nil
}
