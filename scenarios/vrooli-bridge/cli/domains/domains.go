package domains

import (
	"vrooli-bridge/cli/domains/artifacts"
	"vrooli-bridge/cli/domains/attached"
	"vrooli-bridge/cli/domains/audit"
	"vrooli-bridge/cli/domains/auth"
	"vrooli-bridge/cli/domains/dispatch"
	"vrooli-bridge/cli/domains/fleet"
	"vrooli-bridge/cli/domains/gate"
	"vrooli-bridge/cli/domains/machines"
	"vrooli-bridge/cli/domains/nodes"
	"vrooli-bridge/cli/domains/onboard"
	"vrooli-bridge/cli/domains/pairing"
	"vrooli-bridge/cli/domains/provision"
	"vrooli-bridge/cli/domains/queue"
	"vrooli-bridge/cli/domains/readiness"
	"vrooli-bridge/cli/domains/relay"
	"vrooli-bridge/cli/domains/runs"
	"vrooli-bridge/cli/domains/session"

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
	// One registrar per domain. Adding a domain is one line here; the loop keeps
	// this aggregator flat (no per-domain error-handling branch to grow).
	registrars := []func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error){
		auth.Register,
		attached.Register,
		nodes.Register,
		machines.Register,
		pairing.Register,
		dispatch.Register,
		fleet.Register,
		gate.Register,
		provision.Register,
		queue.Register,
		relay.Register,
		runs.Register,
		audit.Register,
		artifacts.Register,
		onboard.Register,
		readiness.Register,
		session.Register,
	}
	groups := make([]cliapp.SubcommandGroup, 0, len(registrars))
	for _, register := range registrars {
		group, err := register(core, manifest)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}
