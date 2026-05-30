package domains

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"

	"architecture-cartographer/cli/domains/analytics"
	"architecture-cartographer/cli/domains/apply"
	"architecture-cartographer/cli/domains/audit"
	"architecture-cartographer/cli/domains/campaign"
	"architecture-cartographer/cli/domains/conflicts"
	domainsdomain "architecture-cartographer/cli/domains/domains"
	"architecture-cartographer/cli/domains/graph"
	"architecture-cartographer/cli/domains/signals"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/conflicts and append their registrations
// here. For greenfield scenarios, domain packages are the default
// architecture; do not treat flat command files as the long-term plan.
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
//
// As each product domain (domains, signals, apply, analytics) ships its
// CLI surface, add a Register call here.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	out := make([]cliapp.SubcommandGroup, 0, 8)

	domainsGroup, err := domainsdomain.Register(core, manifest)
	if err != nil {
		return nil, fmt.Errorf("register domains: %w", err)
	}
	out = append(out, domainsGroup)

	graphGroup, err := graph.Register(core, manifest)
	if err != nil {
		return nil, fmt.Errorf("register graph: %w", err)
	}
	out = append(out, graphGroup)

	conflictsGroup, err := conflicts.Register(core, manifest)
	if err != nil {
		return nil, fmt.Errorf("register conflicts: %w", err)
	}
	out = append(out, conflictsGroup)

	signalsGroup, err := signals.Register(core, manifest)
	if err != nil {
		return nil, fmt.Errorf("register signals: %w", err)
	}
	out = append(out, signalsGroup)

	analyticsGroup, err := analytics.Register(core, manifest)
	if err != nil {
		return nil, fmt.Errorf("register analytics: %w", err)
	}
	out = append(out, analyticsGroup)

	applyGroup, err := apply.Register(core, manifest)
	if err != nil {
		return nil, fmt.Errorf("register apply: %w", err)
	}
	out = append(out, applyGroup)

	auditGroup, err := audit.Register(core, manifest)
	if err != nil {
		return nil, fmt.Errorf("register audit: %w", err)
	}
	out = append(out, auditGroup)

	campaignGroup, err := campaign.Register(core, manifest)
	if err != nil {
		return nil, fmt.Errorf("register campaign: %w", err)
	}
	out = append(out, campaignGroup)

	return out, nil
}
