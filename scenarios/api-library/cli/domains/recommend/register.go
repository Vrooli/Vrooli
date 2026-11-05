package recommend

import (
	"fmt"
	"os"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `api-library recommend` as a flat command that GETs
// /recommendations with optional capability/max_price filters.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Recommendations",
		Commands: []cliapp.Command{
			{
				Name:        "recommend",
				Description: "Recommend APIs for a capability/budget",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runRecommend(core, args) },
			},
		},
	}
}

func runRecommend(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("recommend")
	capability := fs.String("capability", "", "Filter by capability substring")
	maxPrice := fs.String("max-price", "", "Maximum price per request")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"capability": *capability,
		"max_price":  *maxPrice,
	})
	raw, err := core.Get("/recommendations", query)
	if err != nil {
		return err
	}

	var resp map[string]interface{}
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}

	results := support.MapRows(resp)
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recommendations for capability=%q max_price=%q", *capability, *maxPrice)},
		ResultsHeading: "Recommendations",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
