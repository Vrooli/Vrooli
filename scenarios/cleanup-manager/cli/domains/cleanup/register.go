package cleanup

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "cleanup"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"CleanupService.ListProviders":    cliapp.ProtoList(h.listProvidersCall, h.listProvidersReport),
		"CleanupService.GetPolicy":        cliapp.ProtoList(h.getPolicyCall, h.getPolicyReport),
		"CleanupService.SetPolicyProfile": cliapp.ProtoMutation(h.setPolicyProfileCall, h.setPolicyProfileReport),
		"CleanupService.CreatePlan":       cliapp.ProtoOperational(h.createPlanCall, h.createPlanReport),
		"CleanupService.ApplyPlan":        cliapp.ProtoMutation(h.applyPlanCall, h.applyPlanReport),
		"CleanupService.ListAudit":        cliapp.ProtoList(h.listAuditCall, h.listAuditReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("cleanup: load from manifest: %w", err)
	}
	return group, nil
}
