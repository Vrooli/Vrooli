package scenarios

import (
	"fmt"
	"os"
	"strconv"
	"tidiness-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "tidiness-manager"

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "scenarios",
		Description: "Inspect scenario-level tidiness summaries",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List scenario summaries", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one scenario detail", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenarios list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/agent/scenarios", nil)
	if err != nil {
		return err
	}

	var response support.ScenariosResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenarios: %d", response.Count),
		},
		ResultsHeading: "Scenarios",
		Results:        scenarioRows(response.Scenarios),
		RetrievalHints: []string{fmt.Sprintf("%s scenarios get <scenario>", cliName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("scenarios get")
	limit := fs.Int("limit", 10, "Maximum files to show")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenarios get <scenario> [--limit N]")
	}
	scenario := fs.Arg(0)

	body, err := core.Get("/agent/scenarios/"+scenario, nil)
	if err != nil {
		return err
	}

	var response support.ScenarioDetail
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", response.Scenario),
			fmt.Sprintf("Light issues: %d", response.LightIssues),
			fmt.Sprintf("AI issues: %d", response.AIIssues),
			fmt.Sprintf("Long files: %d", response.LongFiles),
		},
		ResultsHeading: "Files",
		Results:        scenarioFileRows(response.Files, *limit),
		RetrievalHints: []string{
			fmt.Sprintf("%s issues %s --limit 20", cliName, response.Scenario),
			fmt.Sprintf("%s score %s", cliName, response.Scenario),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func scenarioRows(scenarios []support.ScenarioSummary) []string {
	if len(scenarios) == 0 {
		return []string{"No scenarios returned"}
	}
	rows := make([]string, 0, len(scenarios))
	for _, scenario := range scenarios {
		rows = append(rows, fmt.Sprintf("%s | total %d | lint %d | type %d | long files %d",
			scenario.Scenario, scenario.Total, scenario.Lint, scenario.Type, scenario.LongFiles))
	}
	return rows
}

func scenarioFileRows(files []support.ScenarioFile, limit int) []string {
	if len(files) == 0 {
		return []string{"No file metrics found"}
	}
	if limit <= 0 || limit > len(files) {
		limit = len(files)
	}
	rows := make([]string, 0, limit)
	for _, file := range files[:limit] {
		rows = append(rows, file.Path+" | issues "+strconv.Itoa(file.TotalIssues)+" | lines "+strconv.Itoa(file.Lines)+" | visits "+strconv.Itoa(file.VisitCount))
	}
	return rows
}
