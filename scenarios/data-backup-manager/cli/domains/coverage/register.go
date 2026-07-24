// Package coverage is the CLI's coverage-domain command surface. Mirrors the
// API's Connect-RPC CoverageService. Operators run `data-backup-manager coverage
// report` to see first-real-backup readiness and `... coverage accept-defaults`
// to bulk-register the recommended non-sensitive durable targets.
//
// The manifest (cli/manifest.json) is the single source of truth for the command
// shape (flags, governance, RPC bindings); this package only wires bindings to
// handlers in handlers.go.
package coverage

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this package owns.
const GroupName = "coverage"

// Register builds the coverage subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CoverageService.GetCoverageReport":    h.report,
		"CoverageService.AcceptDefaultTargets": h.acceptDefaults,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("coverage: load from manifest: %w", err)
	}
	return group, nil
}
