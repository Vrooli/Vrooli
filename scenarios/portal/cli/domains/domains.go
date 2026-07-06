package domains

import (
	"bytes"
	"encoding/json"
	"fmt"

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
	health, err := cliapp.LoadFromManifest(manifest, "health", map[string]func(cliapp.RunContext) error{
		"HealthService.Status": runHealthStatus,
	})
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{health}, nil
}

func runHealthStatus(ctx cliapp.RunContext) error {
	body, err := ctx.Core().GetRoot("/health", nil)
	if err != nil {
		return err
	}
	if ctx.JSON() {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", "  "); err == nil {
			_, err = fmt.Fprintln(ctx.Stdout(), pretty.String())
			return err
		}
	}
	_, err = fmt.Fprintln(ctx.Stdout(), string(body))
	return err
}
