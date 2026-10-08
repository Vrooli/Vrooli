package monitor

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "monitor"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"MonitorService.GetMonitorStatus": cliapp.ProtoList(h.statusCall, h.statusReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("monitor: load from manifest: %w", err)
	}
	return group, nil
}
