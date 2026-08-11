package domains

import (
	"vrooli-onboarding/cli/domains/control"
	"vrooli-onboarding/cli/domains/credentials"
	"vrooli-onboarding/cli/domains/glossary"
	"vrooli-onboarding/cli/domains/operator"
	"vrooli-onboarding/cli/domains/readiness"
	"vrooli-onboarding/cli/domains/resources"
	"vrooli-onboarding/cli/domains/wizard"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb domains like
// `setup-order` and `glossary` live here so the invocation stays
// `vrooli-onboarding glossary ...` instead of
// `vrooli-onboarding glossary list ...`.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return append([]cliapp.CommandGroup{glossary.Register(core), readiness.Register(core)}, control.CommandGroups(core)...)
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return append([]cliapp.SubcommandGroup{
		resources.Register(core),
		credentials.Register(core),
		operator.Register(core),
		wizard.Register(core),
	}, control.SubcommandGroups(core)...)
}
