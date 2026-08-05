package rules

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "rules"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"ClassificationRulesService.ListRules":   cliapp.ProtoList(h.listCall, h.listReport),
		"ClassificationRulesService.CreateRule":   cliapp.ProtoMutation(h.createCall, h.createReport),
		"ClassificationRulesService.DryRunRule":   cliapp.ProtoList(h.dryRunCall, h.dryRunReport),
		"ClassificationRulesService.EnableRule":  cliapp.ProtoMutation(h.enableCall, h.enableReport),
		"ClassificationRulesService.RevertRule": cliapp.ProtoMutation(h.revertCall, h.revertReport),
		"ClassificationRulesService.RefacetCorpus": cliapp.ProtoMutation(h.refacetCall, h.refacetReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("rules: load manifest: %w", err)
	}
	return g, nil
}
