package debt

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "debt"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"DebtService.ListDebt": cliapp.ProtoList(h.listCall, h.listReport),
		"DebtService.GetDebt":  cliapp.ProtoList(h.showCall, h.showReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("debt: load from manifest: %w", err)
	}
	return group, nil
}
