// Package analysis is the CLI's analysis-domain command surface. It mirrors the
// API's AnalysisService: analyze a captured perf trace and compare two traces.
package analysis

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "analysis"

// Register builds the analysis subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AnalysisService.AnalyzeTrace":  h.analyze,
		"AnalysisService.CompareTraces": h.compare,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("analysis: load from manifest: %w", err)
	}
	return group, nil
}
