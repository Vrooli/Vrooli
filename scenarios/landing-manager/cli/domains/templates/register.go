package templates

import (
	"fmt"
	"os"
	"strings"

	"landing-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `template` subcommand group for the template registry:
// `GET /api/v1/templates` and `GET /api/v1/templates/{id}`.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "template",
		Description: "Browse available landing-page templates",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List available templates", Run: func(args []string) error { return runList(core, args) }},
			{Name: "show", Description: "Show detailed template metadata", Run: func(args []string) error { return runShow(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("template list")
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
		Summary:        []string{fmt.Sprintf("Templates: %d", len(templates))},
		ResultsHeading: "Templates",
		Results:        templateRows(templates),
		RetrievalHints: []string{
			fmt.Sprintf("%s template show <template-id>", support.CLIName),
			fmt.Sprintf("%s generate <template-id> --name <name> --slug <slug>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("template show")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: template show <template-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/templates/"+id, nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Template: %s", id)},
		ResultsHeading: "Details",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{
			fmt.Sprintf("%s generate %s --name <name> --slug <slug>", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func templateRows(templates []support.Template) []string {
	if len(templates) == 0 {
		return []string{"No templates available"}
	}
	rows := make([]string, 0, len(templates))
	for _, t := range templates {
		line := fmt.Sprintf("%s | %s", t.ID, t.Name)
		if t.Description != "" {
			line += " | " + t.Description
		}
		if len(t.Tags) > 0 {
			line += " | tags=" + strings.Join(t.Tags, ",")
		}
		rows = append(rows, line)
	}
	return rows
}
