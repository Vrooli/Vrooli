package machines

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "machines"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"MachineService.CreateMachine":            h.create,
		"MachineService.GetMachine":               h.get,
		"MachineService.ListMachines":             h.list,
		"MachineService.ArchiveMachine":           h.archive,
		"MachineService.RemoveMachine":            h.remove,
		"MachineService.GetMachineTrust":          h.getTrust,
		"MachineService.ReviewMachineHostKey":     h.reviewHostKey,
		"MachineService.RequestMachineSSHCleanup": h.requestSSHCleanup,
		"MachineService.UpdateMachineCleanup":     h.updateCleanup,
		"MachineService.ApplyMachinePolicy":       h.applyPolicy,
		"MachineService.RevokeMachineNode":        h.revokeNode,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("machines: load from manifest: %w", err)
	}
	return group, nil
}
