package categories

import (
	"fmt"
	"os"

	"bookmark-intelligence-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `bookmark-intelligence-hub categories` as a flat command
// because the API surface is a single read-only endpoint
// (`GET /api/v1/categories`). Category CRUD endpoints exist server-side
// (POST/PUT/DELETE) but currently return 501 Not Implemented, so we don't
// wrap them yet.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Categories",
		Commands: []cliapp.Command{
			{
				Name:        "categories",
				Description: "List bookmark categories with counts",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("categories")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/categories", nil)
	if err != nil {
		return err
	}
	var categories []support.Category
	if err := support.Decode(body, &categories); err != nil {
		return err
	}

	rows := make([]string, 0, len(categories))
	for _, c := range categories {
		rows = append(rows, fmt.Sprintf("%s: %d bookmarks", c.Name, c.Count))
	}
	if len(rows) == 0 {
		rows = []string{"(no categories)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Categories: %d", len(categories))},
		ResultsHeading: "Categories",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s bookmark query --category <name>", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
