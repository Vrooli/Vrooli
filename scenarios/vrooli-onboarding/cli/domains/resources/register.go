package resources

import (
	"fmt"
	"os"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `resources` subcommand group covering list/get/health.
// The API is the source of truth for resource discovery (it delegates to
// `vrooli resource status --json`); this package is a thin wrapper.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "resources",
		Description: "List Vrooli resources and inspect their onboarding health",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List available resources", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one resource by name", Run: func(args []string) error { return getResource(core, args) }},
			{Name: "health", Description: "Show onboarding health for all resources", Run: func(args []string) error { return runHealth(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resources list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/resources", nil)
	if err != nil {
		return err
	}
	var resp support.ResourceList
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Resources: %d", resp.Count)},
		ResultsHeading: "Resources",
		Results:        resourceRows(resp.Resources),
		RetrievalHints: []string{
			fmt.Sprintf("%s resources get <name>", support.CLIName),
			fmt.Sprintf("%s resources health", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func getResource(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resources get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: resources get <name>")
	}
	name := fs.Arg(0)

	body, err := core.Get("/resources/"+name, nil)
	if err != nil {
		return err
	}
	var res support.Resource
	if err := support.Decode(body, &res); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Name: %s", res.Name),
		fmt.Sprintf("Status: %s", res.Status),
		fmt.Sprintf("Category: %s", res.Category),
		fmt.Sprintf("Installed: %t", res.Installed),
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Resource: %s (%s)", res.Name, res.Status)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s resources health", support.CLIName),
			fmt.Sprintf("%s setup-order", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runHealth(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resources health")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/resources/health", nil)
	if err != nil {
		return err
	}
	var resp support.ResourceHealthResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Resource health: %d/%d healthy", resp.HealthyCount, resp.Total),
			fmt.Sprintf("Checked at: %s", support.FormatTime(resp.CheckedAt)),
		},
		ResultsHeading: "Statuses",
		Results:        healthRows(resp.Resources),
		RetrievalHints: []string{
			fmt.Sprintf("%s resources list", support.CLIName),
			fmt.Sprintf("%s status", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func resourceRows(resources []support.Resource) []string {
	if len(resources) == 0 {
		return []string{"No resources available"}
	}
	rows := make([]string, 0, len(resources))
	for _, r := range resources {
		rows = append(rows, fmt.Sprintf("%s | status=%s | category=%s | installed=%t",
			r.Name, r.Status, r.Category, r.Installed))
	}
	return rows
}

func healthRows(statuses []support.ResourceHealthStatus) []string {
	if len(statuses) == 0 {
		return []string{"No resources reporting health"}
	}
	rows := make([]string, 0, len(statuses))
	for _, s := range statuses {
		rows = append(rows, fmt.Sprintf("%s | status=%s | category=%s | available=%t",
			s.Name, s.Status, s.Category, s.Available))
	}
	return rows
}
