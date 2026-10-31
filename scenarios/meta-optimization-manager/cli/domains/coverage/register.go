// Package coverage is the CLI's coverage-domain command surface. Mirrors the
// API's Connect-RPC CoverageService and the UI's coverage client. The manifest
// (cli/manifest.json) carries the declarative command shape; handlers.go builds
// each typed request and renders the response.
package coverage

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "coverage"

// Register builds the coverage subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CoverageService.GetStatus":        h.status,
		"CoverageService.ListCells":        h.listCells,
		"CoverageService.ExplainCell":      h.explainCell,
		"CoverageService.ValidateBaseDocs": h.validateDocs,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("coverage: load from manifest: %w", err)
	}
	return group, nil
}
