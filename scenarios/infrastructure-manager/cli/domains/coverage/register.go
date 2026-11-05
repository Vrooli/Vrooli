package coverage

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "coverage"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"CoverageService.GetCoverage":       cliapp.ProtoList(h.statusCall, h.statusReport),
		"CoverageService.ListCells":         cliapp.ProtoList(h.cellsCall, h.cellsReport),
		"CoverageService.ListOpenLoopCells": cliapp.ProtoList(h.openLoopCall, h.cellsReport),
		"CoverageService.GetProjection":     cliapp.ProtoList(h.showCall, h.showReport),
		"CoverageService.ValidateSetpoint":  cliapp.ProtoList(h.validateCall, h.validateReport),
		"CoverageService.GetDrift":          cliapp.ProtoList(h.driftCall, h.driftReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("coverage: load from manifest: %w", err)
	}
	return group, nil
}
