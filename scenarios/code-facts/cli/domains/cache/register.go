package cache

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "cache"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"CodeFactsService.GetCacheStatus": cliapp.ProtoList(h.statusCall, h.statusReport),
		"CodeFactsService.InspectCache":   cliapp.ProtoList(h.inspectCall, h.statusReport),
		"CodeFactsService.ClearCache":     cliapp.ProtoMutation(h.clearCall, h.clearReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("cache: load from manifest: %w", err)
	}
	return group, nil
}
