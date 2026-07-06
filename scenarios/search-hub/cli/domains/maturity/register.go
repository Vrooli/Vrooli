// Package maturity owns target search-maturity scan commands.
package maturity

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "maturity"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.ValidateScenario": h.scan,
		"ScenarioValidationService.PreviewFix":       h.fix,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("maturity: load from manifest: %w", err)
	}
	// The handler reports provider liveness explicitly; a generic API preflight
	// would collapse provider availability into command startup failure.
	group.NeedsAPI = false
	return group, nil
}
