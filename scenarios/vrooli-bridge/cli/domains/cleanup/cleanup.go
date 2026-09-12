// Package cleanup is the operator-facing Bridge cleanup command group. The
// manifest owns governance; these handlers only bind typed Connect requests and
// render the durable operation.
package cleanup

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "cleanup"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CleanupService.ProvisionBreakGlass": h.provisionBreakGlass,
		"CleanupService.ResetBreakGlass":     h.resetBreakGlass,
		"CleanupService.StartCleanup":        h.start,
		"CleanupService.GetCleanup":          h.get,
		"CleanupService.PlanCleanup":         h.plan,
		"CleanupService.ConfirmCleanup":      h.confirm,
		"CleanupService.ApplyCleanup":        h.apply,
		"CleanupService.VerifyCleanup":       h.verify,
		"CleanupService.CancelCleanup":       h.cancel,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("cleanup: load from manifest: %w", err)
	}
	return group, nil
}
