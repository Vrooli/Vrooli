package analytics

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"ai-chatbot-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `ai-chatbot-manager analytics <chatbot-id>` — a single-verb
// read endpoint with a `--days` filter, mirroring the bash CLI.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Analytics",
		Commands: []cliapp.Command{
			{
				Name:        "analytics",
				Description: "Show chatbot analytics and metrics",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runAnalytics(core, args) },
			},
		},
	}
}

func runAnalytics(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analytics")
	days := fs.Int("days", 7, "Number of days to analyze")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: analytics <chatbot-id> [--days N]")
	}
	id := fs.Arg(0)

	query := support.BuildQuery(map[string]string{"days": strconv.Itoa(*days)})
	body, err := core.Get("/analytics/"+id, query)
	if err != nil {
		return err
	}

	// Analytics is a large, shape-variable payload; decode into a generic map so
	// new fields from the API flow through unfiltered.
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Analytics for %s over last %d day(s)", id, *days)}
	results := flattenAnalytics(data)

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Metrics",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s analytics %s --days 30", support.CLIName, id),
			fmt.Sprintf("%s chatbot get %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func flattenAnalytics(data map[string]interface{}) []string {
	if len(data) == 0 {
		return []string{"(no analytics data)"}
	}
	// Use MapRows for the top level so output is stable and sorted; serialize
	// nested objects/arrays as JSON for transparency.
	rows := support.MapRows(data)
	// Promote a common nested summary if present.
	if top, ok := data["top_intents"].([]interface{}); ok && len(top) > 0 {
		rows = append(rows, "Top intents:")
		for _, item := range top {
			if raw, err := json.Marshal(item); err == nil {
				rows = append(rows, "  "+string(raw))
			}
		}
	}
	return rows
}
