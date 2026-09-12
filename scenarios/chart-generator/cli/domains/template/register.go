package template

import (
	"fmt"
	"os"

	"chart-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `template` subcommand group wrapping /api/v1/templates
// and /api/v1/templates/{id}. The API supports server-side filtering via
// category/industry query params; the CLI passes both through unchanged.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "template",
		Description: "List and inspect industry chart templates",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List chart templates (filterable by --category/--industry)", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one template", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("template list")
	category := fs.String("category", "", "Filter by category (business, financial, healthcare, technology, ...)")
	industry := fs.String("industry", "", "Filter by industry (retail, saas, finance, ...)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"category": *category,
		"industry": *industry,
	})
	body, err := core.Get("/templates", query)
	if err != nil {
		return err
	}
	var resp support.TemplateListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Templates: %d", resp.Count)}
	if *category != "" {
		summary = append(summary, fmt.Sprintf("Category: %s", *category))
	}
	if *industry != "" {
		summary = append(summary, fmt.Sprintf("Industry: %s", *industry))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Templates",
		Results:        templateRows(resp.Templates),
		RetrievalHints: []string{
			fmt.Sprintf("%s template get <template-id>", support.CLIName),
			fmt.Sprintf("%s template list --category business", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("template get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: template get <template-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/templates/"+id, nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Template: %s", id)},
		ResultsHeading: "Details",
		Results:        support.MapRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s template list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func templateRows(templates []support.Template) []string {
	if len(templates) == 0 {
		return []string{"No templates found"}
	}
	rows := make([]string, 0, len(templates))
	for _, t := range templates {
		row := fmt.Sprintf("%s (%s)", t.Name, t.ID)
		if t.ChartType != "" {
			row += fmt.Sprintf(" | type=%s", t.ChartType)
		}
		if t.Category != "" {
			row += fmt.Sprintf(" | category=%s", t.Category)
		}
		if t.Industry != "" {
			row += fmt.Sprintf(" | industry=%s", t.Industry)
		}
		if t.Description != "" {
			row += " — " + t.Description
		}
		rows = append(rows, row)
	}
	return rows
}
