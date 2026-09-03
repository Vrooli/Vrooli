// Package workflows exposes the durable RCL-owned assisted-work ledger.
package workflows

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "workflows"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"WorkflowsService.StartWorkflow":         h.start,
		"WorkflowsService.ListWorkflows":         h.list,
		"WorkflowsService.GetWorkflow":           h.get,
		"WorkflowsService.GetPromotionReadiness": h.promotionReadiness,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("workflows: load from manifest: %w", err)
	}
	return group, nil
}
