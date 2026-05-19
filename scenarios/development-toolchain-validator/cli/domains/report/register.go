// Package report is the CLI's report command surface. Mirrors the API's
// Connect-RPC ReportService. Command surface loads from cli/manifest.json
// via cliapp.LoadFromManifest.
package report

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "report"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ReportService.GetGoldenSummary": h.goldenSummary,
		"ReportService.GetTupleHistory":  h.tupleHistory,
		"ReportService.GetCoverage":      h.coverage,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("report: load from manifest: %w", err)
	}
	return group, nil
}
