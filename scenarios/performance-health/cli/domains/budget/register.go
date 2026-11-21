// Package budget is the CLI's budget-domain command surface. It mirrors the
// API's BudgetService: read, write, and check per-scenario performance budgets.
package budget

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "budget"

// Register builds the budget subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"BudgetService.GetBudget":   h.get,
		"BudgetService.SetBudget":   h.set,
		"BudgetService.CheckBudget": h.check,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("budget: load from manifest: %w", err)
	}
	return group, nil
}
