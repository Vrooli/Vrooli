// Package machines exposes the operator-facing fleet surface: which machines
// this installation has linked, which are asking to join, and what each one is
// allowed to do. It is the terminal-side view of the same surface the Web
// Console renders, so an operator never has to open the control plane's own
// interface to link a machine.
package machines

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "machine"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"MachineService.List":      h.list,
		"MachineService.IssueCode": h.issueCode,
		"MachineService.Decide":    h.decide,
		"MachineService.SetGrant":  h.setGrant,
		"MachineService.Forget":    h.forget,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("machine: load from manifest: %w", err)
	}
	return group, nil
}
