package condition

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "condition"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ConditionService.GetCondition":         cliapp.ProtoList(h.getCall, h.getReport),
		"ConditionService.GetTrustDistribution": cliapp.ProtoList(h.trustCall, h.trustReport),
		"ConditionService.ExplainCell":          cliapp.ProtoList(h.explainCall, h.explainReport),
		"ConditionService.GetHistory":           cliapp.ProtoList(h.historyCall, h.historyReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("condition: load from manifest: %w", err)
	}
	return group, nil
}
