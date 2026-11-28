package drift

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "drift"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"search": cliapp.Action(h.searchCall, h.searchReport)})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("drift: load manifest: %w", err)
	}
	return group, nil
}
