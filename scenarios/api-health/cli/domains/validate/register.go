package validate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "validate"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ScenarioValidationService.ValidateScenario": cliapp.ProtoListOutcome(h.validateScenarioCall, h.validateScenarioReport, h.validateScenarioOutcome),
		"ScenarioValidationService.PreviewFix":       cliapp.ProtoList(h.previewFixCall, h.fixReport(false)),
		"ScenarioValidationService.ApplyFix":         cliapp.ProtoList(h.applyFixCall, h.fixReport(true)),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("validate: load from manifest: %w", err)
	}
	return group, nil
}
