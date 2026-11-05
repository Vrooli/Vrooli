package performance

import (
	"fmt"
	"os"
	"strconv"

	"algorithm-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wires the `performance` subcommand group. All three endpoints
// (history/trends/record) return variable-shape payloads, so we decode
// generically rather than binding typed structs.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "performance",
		Description: "Inspect and record algorithm performance metrics",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "history", Description: "Show performance history for an algorithm", Run: func(args []string) error { return runHistory(core, args) }},
			{Name: "trends", Description: "Show performance trends for an algorithm", Run: func(args []string) error { return runTrends(core, args) }},
			{Name: "record", Description: "Record a new performance data point", Run: func(args []string) error { return runRecord(core, args) }},
		},
	}
}

func runHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("performance history")
	limit := fs.Int("limit", 0, "Optional max entries to retrieve")
	language := fs.String("language", "", "Filter by language")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: performance history <algorithm-id> [--limit N] [--language L]")
	}
	id := fs.Arg(0)

	params := map[string]string{"language": *language}
	if *limit > 0 {
		params["limit"] = strconv.Itoa(*limit)
	}
	body, err := core.Get("/algorithms/"+id+"/performance-history", support.BuildQuery(params))
	if err != nil {
		return err
	}
	return renderList(body, fmt.Sprintf("Performance history for %s", id), "Entries", *jsonOutput)
}

func runTrends(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("performance trends")
	window := fs.String("window", "", "Aggregation window (e.g. 7d, 30d)")
	language := fs.String("language", "", "Filter by language")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: performance trends <algorithm-id> [--window W] [--language L]")
	}
	id := fs.Arg(0)

	params := support.BuildQuery(map[string]string{
		"window":   *window,
		"language": *language,
	})
	body, err := core.Get("/algorithms/"+id+"/performance-trends", params)
	if err != nil {
		return err
	}
	return renderList(body, fmt.Sprintf("Performance trends for %s", id), "Trends", *jsonOutput)
}

func runRecord(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("performance record")
	bodyFile := fs.String("body-file", "", "Path to performance record JSON (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/performance/record", nil, payload)
	if err != nil {
		return err
	}
	return renderMutation(body, "Recorded performance data point", *jsonOutput)
}

func renderMutation(body []byte, title string, jsonOut bool) error {
	var data interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{title},
		Changes: renderGeneric(data),
	}
	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderList(body []byte, summary, heading string, jsonOut bool) error {
	var data interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: heading,
		Results:        renderGeneric(data),
	}
	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderGeneric(value interface{}) []string {
	switch v := value.(type) {
	case map[string]interface{}:
		return support.MapRows(v)
	case []interface{}:
		if len(v) == 0 {
			return []string{"(empty list)"}
		}
		rows := make([]string, 0, len(v))
		for i, item := range v {
			rows = append(rows, fmt.Sprintf("%d: %s", i, support.RenderValue(item)))
		}
		return rows
	case nil:
		return []string{"(empty payload)"}
	default:
		return []string{support.RenderValue(v)}
	}
}
