package rules

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "rules"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"ClassificationRulesService.ListRules":           cliapp.ProtoList(h.listCall, h.listReport),
		"ClassificationRulesService.CreateRule":          cliapp.ProtoMutation(h.createCall, h.createReport),
		"ClassificationRulesService.DeleteRule":          cliapp.ProtoMutation(h.deleteCall, h.deleteReport),
		"ClassificationRulesService.DryRunRule":          cliapp.ProtoList(h.dryRunCall, h.dryRunReport),
		"ClassificationRulesService.EnableRule":          cliapp.ProtoMutation(h.enableCall, h.enableReport),
		"ClassificationRulesService.RevertRule":          cliapp.ProtoMutation(h.revertCall, h.revertReport),
		"ClassificationRulesService.RefacetCorpus":       cliapp.ProtoMutation(h.refacetCall, h.refacetReport),
		"ClassificationRulesService.MeasureDistribution": cliapp.ProtoList(h.measureCall, h.measureReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("rules: load manifest: %w", err)
	}
	return group, nil
}
