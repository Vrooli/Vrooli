package runs

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "runs"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ValidationRunService.RunTemplateValidation": cliapp.ProtoMutation(h.runCall, h.runReport),
		"ValidationRunService.ListValidationRuns":    cliapp.ProtoList(h.listCall, h.listReport),
		"ValidationRunService.GetValidationRun":      cliapp.ProtoList(h.showCall, h.showReport),
		"ValidationRunService.RecordFleetDrift":      cliapp.ProtoMutation(h.driftRecordCall, h.driftRecordReport),
		"ValidationRunService.ListDriftSnapshots":    cliapp.ProtoList(h.driftCall, h.driftReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("runs: load from manifest: %w", err)
	}
	return group, nil
}
