package lighthouse

import (
	"fmt"
	"os"
	"strings"

	"app-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `lighthouse` subcommand group covering Lighthouse audit operations.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "lighthouse",
		Description: "Run and inspect Lighthouse audits",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "missing-configs", Description: "List scenarios without Lighthouse config", Run: func(args []string) error { return runMissing(core, args) }},
			{Name: "run", Description: "Run Lighthouse for a scenario", Run: func(args []string) error { return runAudit(core, args) }},
			{Name: "history", Description: "Show audit history for a scenario", Run: func(args []string) error { return runHistory(core, args) }},
			{Name: "report", Description: "Fetch a specific audit report", Run: func(args []string) error { return runReport(core, args) }},
		},
	}
}

func runMissing(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("lighthouse missing-configs")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/lighthouse/missing-configs", nil)
	if err != nil {
		return err
	}
	var payload struct {
		Missing []map[string]string `json:"missing"`
		Count   int                 `json:"count"`
		Total   int                 `json:"total"`
	}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	rows := make([]string, 0, len(payload.Missing))
	for _, entry := range payload.Missing {
		rows = append(rows, fmt.Sprintf("%s | expected=%s", entry["scenario"], entry["expected_path"]))
	}
	if len(rows) == 0 {
		rows = []string{"(no missing configs)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenarios without lighthouse.json: %d of %d", payload.Count, payload.Total)},
		ResultsHeading: "Missing configs",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s lighthouse run <scenario>", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runAudit(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("lighthouse run")
	pages := fs.String("pages", "", "Comma-separated pages to audit")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: lighthouse run <scenario> [--pages /,/about]")
	}
	scenario := fs.Arg(0)

	var requestBody interface{}
	if strings.TrimSpace(*pages) != "" {
		pageList := make([]string, 0)
		for _, p := range strings.Split(*pages, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				pageList = append(pageList, trimmed)
			}
		}
		requestBody = map[string]interface{}{"pages": pageList}
	}

	body, err := core.Request("POST", "/scenarios/"+scenario+"/lighthouse/run", nil, requestBody)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Lighthouse run for %s: %s", scenario, support.RenderValue(data["status"]))},
		Changes:     support.MapRows(data),
		NextCommand: []string{fmt.Sprintf("%s lighthouse history %s", support.CLIName, scenario)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("lighthouse history")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: lighthouse history <scenario>")
	}
	scenario := fs.Arg(0)

	body, err := core.Get("/scenarios/"+scenario+"/lighthouse/history", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	reports, _ := data["reports"].([]interface{})
	rows := make([]string, 0, len(reports))
	for _, raw := range reports {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		rows = append(rows, fmt.Sprintf("%s | page=%s | performance=%s",
			support.RenderValue(entry["report_id"]),
			support.RenderValue(entry["page_id"]),
			support.RenderValue(entry["performance"]),
		))
	}
	if len(rows) == 0 {
		rows = []string{"(no reports)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Lighthouse history for %s", scenario)},
		ResultsHeading: "Reports",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s lighthouse report %s <report-id>", support.CLIName, scenario)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runReport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("lighthouse report")
	format := fs.String("format", "json", "Report format: html|json")
	output := fs.String("output", "", "Write report to file (default: stdout)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: lighthouse report <scenario> <report-id> [--format html|json] [--output path]")
	}
	scenario := fs.Arg(0)
	reportID := fs.Arg(1)

	query := support.BuildQuery(map[string]string{"format": *format})
	body, err := core.Get("/scenarios/"+scenario+"/lighthouse/report/"+reportID, query)
	if err != nil {
		return err
	}

	if *format == "html" || strings.TrimSpace(*output) != "" {
		if err := support.WriteOutput(*output, body); err != nil {
			return err
		}
		if *jsonOutput {
			return cliapp.PrintReportJSON(os.Stdout, cliapp.MutationReport{
				Result:  []string{fmt.Sprintf("Wrote report %s", reportID)},
				Changes: []string{fmt.Sprintf("Output: %s", outputOrStdout(*output))},
			})
		}
		if strings.TrimSpace(*output) != "" {
			fmt.Fprintf(os.Stdout, "Wrote report %s to %s\n", reportID, *output)
		}
		return nil
	}

	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Report %s for %s", reportID, scenario)},
		ResultsHeading: "Fields",
		Results:        support.MapRows(data),
		RetrievalHints: []string{fmt.Sprintf("%s lighthouse history %s", support.CLIName, scenario)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func outputOrStdout(path string) string {
	if strings.TrimSpace(path) == "" {
		return "stdout"
	}
	return path
}
