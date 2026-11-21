package focus

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "focus"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"FocusService.GetNext":     cliapp.ProtoList(h.nextCall, h.nextReport),
		"FocusService.GetFinding":  cliapp.ProtoList(h.findingCall, h.findingReport),
		"FocusService.ListSources": cliapp.ProtoList(h.sourcesCall, h.sourcesReport),
		"FocusService.GetEfficacy": cliapp.ProtoList(h.efficacyCall, h.efficacyReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("focus: load from manifest: %w", err)
	}
	return group, nil
}
