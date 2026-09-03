package cleanup

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "cleanup"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	return register(core, manifest, GroupName)
}

// RegisterRecovery exposes the recovery lifecycle under its canonical
// operator-facing group. The handlers remain shared with cleanup so both
// command paths use the same server-owned RPCs and renderers.
func RegisterRecovery(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	group, err := register(core, manifest, "recovery")
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	h := newHandlers(core)
	chaos := cliapp.Action(h.chaosCall, chaosReport)
	for i := range group.Subcommands {
		if group.Subcommands[i].Name == "chaos" {
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(chaos)
			return group, nil
		}
	}
	return cliapp.SubcommandGroup{}, fmt.Errorf("recovery: manifest is missing chaos command")
}

func register(core *cliapp.ScenarioApp, manifest []byte, groupName string) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"CleanupService.ListProviders":    cliapp.ProtoList(h.listProvidersCall, h.listProvidersReport),
		"CleanupService.GetPolicy":        cliapp.ProtoList(h.getPolicyCall, h.getPolicyReport),
		"CleanupService.SetPolicyProfile": cliapp.ProtoMutation(h.setPolicyProfileCall, h.setPolicyProfileReport),
		"CleanupService.CreatePlan":       cliapp.ProtoOperational(h.createPlanCall, h.createPlanReport),
		"CleanupService.ApplyPlan":        cliapp.ProtoMutation(h.applyPlanCall, h.applyPlanReport),
		"CleanupService.ReportPressure":   cliapp.ProtoMutation(h.reportPressureCall, h.reportPressureReport),
		"CleanupService.StartRecovery":    cliapp.ProtoMutation(h.startRecoveryCall, h.recoveryReport),
		"CleanupService.WaitRecovery":     cliapp.ProtoOperational(h.waitRecoveryCall, h.recoveryRunReport),
		"CleanupService.ListRecovery":     cliapp.ProtoList(h.listRecoveryCall, h.listRecoveryReport),
		"CleanupService.ListAudit":        cliapp.ProtoList(h.listAuditCall, h.listAuditReport),
		"tiers":                           cliapp.Action(h.tiersCall, tiersReport),
		"roots":                           cliapp.Action(h.rootsCall, rootsReport),
	}
	if groupName == "recovery" {
		bindings["chaos"] = cliapp.Action(h.chaosCall, chaosReport)
		for method := range bindings {
			if method != "CleanupService.StartRecovery" && method != "CleanupService.WaitRecovery" && method != "CleanupService.ListRecovery" && method != "chaos" {
				delete(bindings, method)
			}
		}
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, groupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", groupName, err)
	}
	return group, nil
}
