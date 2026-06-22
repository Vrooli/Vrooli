// Package advisor is the CLI's advisor-domain command surface. It mirrors the
// API's AdvisorService: migration-hygiene grading and Postgres→SQLite
// engine-fitness ranking.
package advisor

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "advisor"

// Register builds the advisor subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AdvisorService.AnalyzeMigrations": h.migrations,
		"AdvisorService.AdviseEngines":     h.engines,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("advisor: load from manifest: %w", err)
	}
	return group, nil
}
