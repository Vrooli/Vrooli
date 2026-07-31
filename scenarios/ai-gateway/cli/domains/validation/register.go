package validation

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "validation"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.ValidateScenario": h.validate,
		"ScenarioValidationService.DescribeProvider": h.describeProvider,
		"ScenarioValidationService.PreviewFix":       h.previewFix,
		"ScenarioValidationService.ApplyFix":         h.applyFix,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validation: load from manifest: %w", err)
	}
	return group, nil
}
