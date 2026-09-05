// Package workflows exposes the durable RCL-owned assisted-work ledger.
package workflows

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "workflows"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"WorkflowsService.StartWorkflow":         cliapp.ProtoMutation(h.startCall, h.startReport),
		"WorkflowsService.ListWorkflows":         cliapp.ProtoList(h.listCall, h.listReport),
		"WorkflowsService.GetWorkflow":           cliapp.ProtoList(h.getCall, h.getReport),
		"WorkflowsService.GetPromotionReadiness": cliapp.ProtoList(h.promotionReadinessCall, h.promotionReadinessReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("workflows: load from manifest: %w", err)
	}
	return group, nil
}
