package domains

import (
	"ui-health/cli/domains/fix"
	"ui-health/cli/domains/reindex"
	"ui-health/cli/domains/search"
	"ui-health/cli/domains/validate"
	"ui-health/cli/domains/visual"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	validateGroup, err := validate.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	searchGroup, err := search.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	reindexGroup, err := reindex.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	fixGroup, err := fix.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	visualGroup, err := visual.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{validateGroup, searchGroup, reindexGroup, fixGroup, visualGroup}, nil
}
