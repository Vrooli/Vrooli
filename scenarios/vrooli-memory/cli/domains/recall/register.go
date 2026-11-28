package recall

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "recall"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"RecallService.Recall":            cliapp.ProtoList(h.recallCall, h.recallReport),
		"RecallService.Zoom":              cliapp.ProtoList(h.zoomCall, h.zoomReport),
		"RecallService.Wake":              cliapp.ProtoList(h.wakeCall, h.wakeReport),
		"RecallService.ListSiblingEvents": cliapp.ProtoList(h.siblingsCall, h.siblingsReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("recall: load manifest: %w", err)
	}
	g.DefaultSubcommand = "recall"
	return g, nil
}
