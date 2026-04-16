package catalog

import (
	"fmt"
	"os"
	"strings"

	"tech-tree-designer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "catalog",
		Description: "Inspect and curate the scenario catalog used by the tech tree",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List catalog scenarios", Run: func(args []string) error { return runList(deps, args) }},
			{Name: "refresh", Description: "Refresh the scenario catalog", Run: func(args []string) error { return runRefresh(deps, args) }},
			{Name: "hide", Description: "Hide a scenario from the catalog", Run: func(args []string) error { return runVisibility(deps, args, true) }},
			{Name: "show", Description: "Unhide a scenario in the catalog", Run: func(args []string) error { return runVisibility(deps, args, false) }},
		},
	}
}

func runList(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("catalog list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := deps.Core.Get("/tech-tree/scenario-catalog", nil)
	if err != nil {
		return err
	}
	var response support.ScenarioCatalogSnapshot
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenarios: %d", len(response.Scenarios)),
			fmt.Sprintf("Edges: %d", len(response.Edges)),
			fmt.Sprintf("Hidden: %d", len(response.Hidden)),
			"Last synced: " + support.FormatDateTime(response.LastSynced),
		},
		ResultsHeading: "Catalog scenarios",
		Results:        scenarioRows(response.Scenarios),
		RetrievalHints: []string{
			"tech-tree-designer catalog refresh",
			"tech-tree-designer catalog hide <scenario-id>",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRefresh(deps support.Dependencies, args []string) error {
	fs := support.NewFlagSet("catalog refresh")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := deps.Core.Request("POST", "/tech-tree/scenario-catalog/refresh", nil, nil)
	if err != nil {
		return err
	}
	var response support.ScenarioCatalogSnapshot
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{"Refreshed the scenario catalog."},
		Changes: []string{
			"Last synced: " + support.FormatDateTime(response.LastSynced),
		},
		NextCommand: []string{
			"tech-tree-designer catalog list",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runVisibility(deps support.Dependencies, args []string, hidden bool) error {
	commandName := "catalog show"
	if hidden {
		commandName = "catalog hide"
	}
	fs := support.NewFlagSet(commandName)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: %s <scenario-id>", commandName)
	}
	scenario := strings.TrimSpace(fs.Arg(0))
	body, err := deps.Core.Request("POST", "/tech-tree/scenario-catalog/visibility", nil, map[string]interface{}{
		"scenario": scenario,
		"hidden":   hidden,
	})
	if err != nil {
		return err
	}
	var response support.ScenarioCatalogSnapshot
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	action := "visible"
	if hidden {
		action = "hidden"
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Marked %s as %s.", scenario, action)},
		Changes: []string{
			fmt.Sprintf("Hidden scenarios: %d", len(response.Hidden)),
		},
		NextCommand: []string{
			"tech-tree-designer catalog list",
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, response)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func scenarioRows(items []support.ScenarioCatalogEntry) []string {
	if len(items) == 0 {
		return []string{"No catalog entries found."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s | %s | %s", item.Name, item.Category, item.Description))
	}
	return rows
}
