package portability

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "portability"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"PortabilityService.GetGrid":        cliapp.ProtoList(h.gridCall, h.gridReport),
		"PortabilityService.GetCapability":  cliapp.ProtoList(h.capabilityCall, h.capabilityReport),
		"PortabilityService.ListSituations": cliapp.ProtoList(h.situationsCall, h.situationsReport),
		"PortabilityService.GetFleet":       cliapp.ProtoList(h.fleetCall, h.fleetReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("portability: load from manifest: %w", err)
	}
	return group, nil
}
