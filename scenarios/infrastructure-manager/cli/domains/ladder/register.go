package ladder

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "ladder"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"LadderService.GetLadder":    cliapp.ProtoList(h.statusCall, h.statusReport),
		"LadderService.ListCells":    cliapp.ProtoList(h.cellsCall, h.cellsReport),
		"LadderService.ListSources":  cliapp.ProtoList(h.sourcesCall, h.sourcesReport),
		"LadderService.RankFindings": cliapp.ProtoList(h.findingsCall, h.findingsReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ladder: load from manifest: %w", err)
	}
	return group, nil
}
