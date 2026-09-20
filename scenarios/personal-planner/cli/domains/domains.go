package domains

import (
	"personal-planner/cli/domains/calendar"
	"personal-planner/cli/domains/commitments"
	"personal-planner/cli/domains/forecasts"
	"personal-planner/cli/domains/focus"
	"personal-planner/cli/domains/goals"
	"personal-planner/cli/domains/integrations"
	"personal-planner/cli/domains/review"
	"personal-planner/cli/domains/work"
	"personal-planner/cli/domains/workspace"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here. For greenfield scenarios, domain packages are the
// default architecture; do not treat flat command files as the long-term plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Each domain package owns a Register(core, manifest) function returning a
// SubcommandGroup built from the scenario's cli/manifest.json. The aggregator
// passes the embedded manifest bytes through unchanged; per-domain Register
// implementations call cliapp.LoadFromManifest with the relevant group name.
//
// This is the CLI side of the domain-module pattern; the API side uses
// the same one-liner-per-domain shape via server.New(deps, modules...).
// See docs/concepts/ARCHITECTURE.md "Domain modules" for the canonical
// pattern when swapping the example domain for your scenario's first
// domain.
//
// For API-backed commands the manifest carries the declarative surface
// (governance, flags, positionals, RPC binding). Handlers stay in
// handlers.go and are wired via the bindings map; refer to
// templates/scenarios/react-vite/docs/internal/SEAMS.md (manifest ↔
// handlers bindings seam) for the contract.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	groups := []cliapp.SubcommandGroup{}
	calendarGroup, err := calendar.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, calendarGroup)
	commitmentsGroup, err := commitments.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, commitmentsGroup)
	forecastsGroup, err := forecasts.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, forecastsGroup)
	workspaceGroup, err := workspace.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, workspaceGroup)
	goalsGroup, err := goals.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, goalsGroup)
	integrationsGroup, err := integrations.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, integrationsGroup)
	reviewGroup, err := review.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, reviewGroup)
	focusGroup, err := focus.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, focusGroup)
	workGroup, err := work.Register(core, manifest)
	if err != nil {
		return nil, err
	}
	groups = append(groups, workGroup)
	return groups, nil
}
