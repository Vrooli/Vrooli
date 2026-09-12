package cost

import (
	"fmt"
	"os"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `api-library cost` as a flat command that POSTs to
// /calculate-cost. Inputs can be provided via flags or --body-file.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Cost",
		Commands: []cliapp.Command{
			{
				Name:        "cost",
				Description: "Calculate estimated monthly cost for an API",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runCost(core, args) },
			},
		},
	}
}

func runCost(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("cost")
	apiID := fs.String("api-id", "", "API ID (required unless --body-file is used)")
	requests := fs.Int("requests", 0, "Requests per month")
	dataPerReq := fs.Float64("data-per-request-mb", 0, "Average data per request (MB)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the full cost request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var body interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		body = raw
	} else {
		if strings.TrimSpace(*apiID) == "" {
			return fmt.Errorf("usage: cost --api-id <id> [--requests N] [--data-per-request-mb F] | cost --body-file PATH")
		}
		body = map[string]interface{}{
			"api_id":              *apiID,
			"requests_per_month":  *requests,
			"data_per_request_mb": *dataPerReq,
		}
	}

	raw, err := core.Request("POST", "/calculate-cost", nil, body)
	if err != nil {
		return err
	}

	var resp map[string]interface{}
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Cost estimate"},
		ResultsHeading: "Details",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
