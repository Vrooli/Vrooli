package guidance

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "guidance"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"GuidanceService.NextGate": cliapp.ProtoList(h.nextCall, h.nextReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("guidance: load from manifest: %w", err)
	}
	return group, nil
}
