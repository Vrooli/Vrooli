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
		"MachineService.RepairMachine":            h.repair,
		"MachineService.MergeMachines":            h.merge,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("machines: load from manifest: %w", err)
	}
	return group, nil
}

// RegisterConfiguration exposes the document-oriented aliases while keeping
// the legacy machines group stable for existing scripts.
func RegisterConfiguration(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"MachineService.GetMachineConfiguration":    h.get,
		"MachineService.ApplyMachineConfiguration": h.applyPolicy,
		"MachineService.GetMachineDrift":            h.get,
	}
	group, err := cliapp.LoadFromManifest(manifest, "configuration", bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("configuration: load from manifest: %w", err)
	}
	return group, nil
}
