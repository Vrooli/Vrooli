package templates

import (
	"fmt"
	"os"
	"strings"

	"data-structurer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes the read-only /api/v1/schema-templates surface. Schema
// creation-from-template lives under the `schemas` group since it's the one
// endpoint that mutates /schemas/*.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "templates",
		Description: "Browse public schema templates",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List schema templates", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one template", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("templates list")
	category := fs.String("category", "", "Filter templates by category")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"category": *category,
	})
	body, err := core.Get("/schema-templates", query)
	if err != nil {
		return err
	}
	var resp support.SchemaTemplateListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Available templates: %d", resp.Count)}
	if *category != "" {
		summary = append(summary, fmt.Sprintf("Category: %s", *category))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Templates",
		Results:        templateRows(resp.Templates),
		RetrievalHints: []string{
			fmt.Sprintf("%s templates get <template-id>", support.CLIName),
			fmt.Sprintf("%s schemas create-from-template <template-id> <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("templates get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: templates get <template-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/schema-templates/"+id, nil)
	if err != nil {
		return err
	}
	var template support.SchemaTemplate
	if err := support.Decode(body, &template); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", template.ID),
		fmt.Sprintf("Name: %s", template.Name),
	}
	if template.Category != "" {
		results = append(results, fmt.Sprintf("Category: %s", template.Category))
	}
	if template.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", template.Description))
	}
	results = append(results, fmt.Sprintf("Usage count: %d", template.UsageCount))
	if len(template.Tags) > 0 {
		results = append(results, fmt.Sprintf("Tags: %s", strings.Join(template.Tags, ", ")))
	}
	if template.CreatedAt != nil {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTimePtr(template.CreatedAt)))
	}
	if len(template.SchemaDefinition) > 0 {
		results = append(results, "Schema definition:")
		results = append(results, support.MapRows(template.SchemaDefinition)...)
	}
	if len(template.ExampleData) > 0 {
		results = append(results, "Example data:")
		results = append(results, support.MapRows(template.ExampleData)...)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Template: %s (%s)", template.Name, template.Category)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s schemas create-from-template %s <name>", support.CLIName, template.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func templateRows(templates []support.SchemaTemplate) []string {
	if len(templates) == 0 {
		return []string{"No templates available"}
	}
	rows := make([]string, 0, len(templates))
	for _, t := range templates {
		category := t.Category
		if category == "" {
			category = "-"
		}
		description := t.Description
		if description == "" {
			description = "(no description)"
		}
		rows = append(rows, fmt.Sprintf("%s | %s | %s | %d uses | %s",
			support.ShortID(t.ID), t.Name, category, t.UsageCount, description))
	}
	return rows
}
