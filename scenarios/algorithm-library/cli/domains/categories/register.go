package categories

import (
	"fmt"
	"os"

	"algorithm-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `algorithm-library categories` as a flat list command. The
// API returns the category summary as a top-level JSON array, so we expose it
// without a sub-verb.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Categories",
		Commands: []cliapp.Command{
			{
				Name:        "categories",
				Description: "List algorithm categories and counts",
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

	body, err := core.Get("/algorithms/categories", nil)
	if err != nil {
		return err
	}

	var categories []support.Category
	if err := support.Decode(body, &categories); err != nil {
		return err
	}

	results := make([]string, 0, len(categories))
	for _, c := range categories {
		results = append(results, fmt.Sprintf("%s: %d algorithms", c.Name, c.Count))
	}
	if len(results) == 0 {
		results = []string{"(no categories defined)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Categories: %d", len(categories))},
		ResultsHeading: "Categories",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s algorithm search <query> --category <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
