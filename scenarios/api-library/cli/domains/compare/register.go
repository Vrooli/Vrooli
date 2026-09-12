package compare

import (
	"fmt"
	"os"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `api-library compare <id1> <id2>...` as a flat command.
// The API (POST /compare) requires at least two API IDs.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Comparison",
		Commands: []cliapp.Command{
			{
				Name:        "compare",
				Description: "Compare two or more APIs by selected attributes",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runCompare(core, args) },
			},
		},
	}
}

func runCompare(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("compare")
	attributes := fs.String("attributes", "", "Comma-separated attribute names (default: pricing,rate_limits,auth_type,regions,features,support)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the full comparison request body")
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
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: compare <api-id-1> <api-id-2> [<api-id-n>...] [--attributes csv]")
		}
		payload := map[string]interface{}{
			"api_ids": fs.Args(),
		}
		if attrs := splitCSV(*attributes); len(attrs) > 0 {
			payload["attributes"] = attrs
		}
		body = payload
	}

	raw, err := core.Request("POST", "/compare", nil, body)
	if err != nil {
		return err
	}

	var resp map[string]interface{}
	if err := support.Decode(raw, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"API comparison"},
		ResultsHeading: "Matrix",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
