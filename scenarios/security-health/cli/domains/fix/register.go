// Package fix exposes Security Health's deterministic shared Fix RPCs.
package fix

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "fix"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.PreviewFix": h.preview,
		"ScenarioValidationService.ApplyFix":   h.apply,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fix: load from manifest: %w", err)
	}
	return group, nil
}
