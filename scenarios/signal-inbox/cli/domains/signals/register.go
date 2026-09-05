package signals

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "signals"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"SignalsService.CaptureSignal": cliapp.ProtoMutation(h.captureCall, h.captureReport),
		"SignalsService.GetSignal":     cliapp.ProtoList(h.getCall, h.getReport),
		"SignalsService.ListSignals":   cliapp.ProtoList(h.listCall, h.listReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("signals: load manifest: %w", err)
	}
	return group, nil
}
