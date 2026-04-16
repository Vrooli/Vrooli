package export

import (
	"fmt"
	"os"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `api-library export` as a flat command that downloads the
// full API library as JSON or CSV. The response is written to --output or
// stdout; no pretty-printing is applied so CSV round-trips correctly.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Export",
		Commands: []cliapp.Command{
			{
				Name:        "export",
				Description: "Export the full API library as JSON or CSV",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runExport(core, args) },
			},
		},
	}
}

func runExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("export")
	format := fs.String("format", "json", "Export format: json or csv")
	output := fs.String("output", "", "Write output to this path instead of stdout")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	f := strings.TrimSpace(strings.ToLower(*format))
	if f != "json" && f != "csv" {
		return fmt.Errorf("--format must be json or csv, got %q", *format)
	}

	query := support.BuildQuery(map[string]string{"format": f})
	raw, err := core.Get("/export", query)
	if err != nil {
		return err
	}

	if *jsonOutput {
		// Emit an operational report summarising the export; the raw payload
		// remains available via --output.
		report := cliapp.OperationalReport{
			Status: []string{
				fmt.Sprintf("Exported %d bytes in %s format", len(raw), f),
			},
		}
		if strings.TrimSpace(*output) != "" {
			if err := support.WriteOutput(*output, raw); err != nil {
				return err
			}
			report.Status = append(report.Status, fmt.Sprintf("Saved to %s", *output))
		}
		return cliapp.PrintReportJSON(os.Stdout, report)
	}

	return support.WriteOutput(*output, raw)
}
