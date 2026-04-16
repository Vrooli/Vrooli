package analytics

import (
	"fmt"
	"os"
	"sort"

	"bookmark-intelligence-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `analytics` subcommand group. The API currently exposes
// a single endpoint, `/analytics/metrics`, returning a free-form map (total
// bookmarks, processing accuracy, action counts, per-platform breakdown).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "analytics",
		Description: "View processing metrics and insights",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "metrics", Description: "Show processing metrics and insights", Run: func(args []string) error { return runMetrics(core, args) }},
		},
	}
}

func runMetrics(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analytics metrics")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/analytics/metrics", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	results := []string{}
	if v, ok := data["total_bookmarks"].(float64); ok {
		results = append(results, fmt.Sprintf("Total bookmarks: %d", int(v)))
	}
	if v, ok := data["processing_accuracy"].(float64); ok {
		results = append(results, fmt.Sprintf("Processing accuracy: %.2f%%", v))
	}
	if v, ok := data["actions_executed"].(float64); ok {
		results = append(results, fmt.Sprintf("Actions executed: %d", int(v)))
	}
	if breakdown, ok := data["platform_breakdown"].(map[string]interface{}); ok && len(breakdown) > 0 {
		results = append(results, "Platform breakdown:")
		keys := make([]string, 0, len(breakdown))
		for k := range breakdown {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			count := 0
			if v, ok := breakdown[k].(float64); ok {
				count = int(v)
			}
			results = append(results, fmt.Sprintf("  %s: %d", k, count))
		}
	}
	if len(results) == 0 {
		results = support.MapRows(data)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Bookmark intelligence analytics"},
		ResultsHeading: "Metrics",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s analytics metrics --json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
