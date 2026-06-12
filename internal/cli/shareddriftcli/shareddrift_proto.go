package shareddriftcli

import (
	"io"

	shareddrift "github.com/vrooli/vrooli/internal/app/shareddrift"
	"github.com/vrooli/vrooli/internal/cliout"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// sharedDriftReport maps the internal shared-drift report onto the vrooli.cli.v1
// wire contract.
func sharedDriftReport(report shareddrift.Report) *cliv1.SharedDriftReport {
	out := &cliv1.SharedDriftReport{
		Clean:                report.Clean,
		Root:                 report.Root,
		TouchedPackages:      report.TouchedPackages,
		OnlyTouched:          report.OnlyTouchedUsed,
		BuildChecked:         report.BuildChecked,
		FixApplied:           report.FixApplied,
		ModifiedTrackedFiles: report.ModifiedTrackedOK,
		ElapsedMs:            int32(report.ElapsedMs),
	}
	for _, scenario := range report.Scenarios {
		out.Scenarios = append(out.Scenarios, &cliv1.SharedDriftScenario{
			Path:       scenario.Path,
			ApiDir:     scenario.APIDir,
			Status:     string(scenario.Status),
			DiffPaths:  scenario.DiffPaths,
			BuildError: scenario.BuildError,
			Error:      scenario.Error,
			Replaces:   scenario.Replaces,
		})
	}
	return out
}

func writeSharedDriftJSON(w io.Writer, report shareddrift.Report) error {
	return cliout.WriteProtoJSON(w, sharedDriftReport(report))
}
