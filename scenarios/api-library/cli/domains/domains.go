package domains

import (
	"api-library/cli/domains/apis"
	"api-library/cli/domains/catalog"
	"api-library/cli/domains/codegen"
	"api-library/cli/domains/compare"
	"api-library/cli/domains/cost"
	"api-library/cli/domains/export"
	"api-library/cli/domains/recipes"
	"api-library/cli/domains/recommend"
	"api-library/cli/domains/research"
	"api-library/cli/domains/search"
	"api-library/cli/domains/snippets"
	"api-library/cli/domains/webhooks"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains live here
// so the invocation stays `api-library <verb>` instead of `api-library foo verb`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		search.Register(core),
		research.Register(core),
		cost.Register(core),
		compare.Register(core),
		recommend.Register(core),
		export.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		apis.Register(core),
		catalog.Register(core),
		snippets.Register(core),
		recipes.Register(core),
		webhooks.Register(core),
		codegen.Register(core),
	}
}
