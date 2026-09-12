package docs

import (
	"fmt"
	"os"

	"data-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `data-tools docs` as a flat command. The API serves `/docs`
// at the root (outside `/api/v1`).
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Docs",
		Commands: []cliapp.Command{
			{
				Name:        "docs",
				Description: "Show API documentation",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runDocs(core, args) },
			},
		},
	}
}

func runDocs(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("docs")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.GetRoot("/docs", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Data Tools API documentation"},
		ResultsHeading: "Fields",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{fmt.Sprintf("%s docs --json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
