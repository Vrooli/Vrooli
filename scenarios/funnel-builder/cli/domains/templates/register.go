package templates

import (
	"encoding/json"
	"fmt"
	"os"

	"funnel-builder/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "funnel-builder"

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "templates",
		Description: "Inspect funnel templates",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List available templates", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one template", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("templates list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/templates", nil)
	if err != nil {
		return err
	}
	var templates []support.Template
	if err := support.Decode(body, &templates); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Templates: %d", len(templates)),
		},
		ResultsHeading: "Templates",
		Results:        templateRows(templates),
		RetrievalHints: []string{fmt.Sprintf("%s templates get <template-slug>", cliName)},
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
		return fmt.Errorf("usage: templates get <slug>")
	}
	slug := fs.Arg(0)

	body, err := core.Get("/templates/"+slug, nil)
	if err != nil {
		return err
	}
	var tpl support.Template
	if err := support.Decode(body, &tpl); err != nil {
		return err
	}

	stepsCount := 0
	var metrics map[string]interface{}
	_ = json.Unmarshal(tpl.Metrics, &metrics)
	var templateData map[string]interface{}
	if err := json.Unmarshal(tpl.TemplateData, &templateData); err == nil {
		if rawSteps, ok := templateData["steps"].([]interface{}); ok {
			stepsCount = len(rawSteps)
		}
	}

	results := []string{
		fmt.Sprintf("Category: %s", tpl.Category),
		fmt.Sprintf("Description: %s", tpl.Description),
		fmt.Sprintf("Template steps: %d", stepsCount),
	}
	if len(metrics) > 0 {
		for key, value := range metrics {
			results = append(results, fmt.Sprintf("Metric %s: %v", key, value))
		}
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Template: %s", tpl.Name),
			fmt.Sprintf("Slug: %s", tpl.Slug),
		},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s funnels create --name \"New Funnel\" --project <project-id> --template %s", cliName, tpl.Slug)},
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
	for _, tpl := range templates {
		rows = append(rows, fmt.Sprintf("%s | %s | category=%s", tpl.Slug, tpl.Name, tpl.Category))
	}
	return rows
}
