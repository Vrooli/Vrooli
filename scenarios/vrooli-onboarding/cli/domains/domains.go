package domains

import (
	"vrooli-onboarding/cli/domains/glossary"
	"vrooli-onboarding/cli/domains/operator"
	"vrooli-onboarding/cli/domains/resources"
	"vrooli-onboarding/cli/domains/setuporder"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like
// `setup-order` and `glossary` live here so the invocation stays
// `vrooli-onboarding glossary ...` instead of
// `vrooli-onboarding glossary list ...`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		setuporder.Register(core),
		glossary.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		resources.Register(core),
		operator.Register(core),
	}
}
