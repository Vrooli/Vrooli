package template

import (
	"fmt"
	"os"
	"strings"

	"notes/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `template` subcommand group covering /api/templates endpoints.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "template",
		Description: "List and create note templates",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List templates", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Aliases: []string{"new", "add"}, Description: "Create a template", Run: func(args []string) error { return runCreate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("template list")
	userID := fs.String("user-id", "", "Filter by user ID (server-side)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"user_id": *userID,
	})
	body, err := core.Get("/templates", query)
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
			fmt.Sprintf("%s template create --name \"...\" --content \"...\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("template create")
	name := fs.String("name", "", "Template name (required)")
	description := fs.String("description", "", "Description")
	content := fs.String("content", "", "Template content")
	category := fs.String("category", "", "Category")
	userID := fs.String("user-id", "", "Owner user ID")
	public := fs.Bool("public", false, "Mark as public (shared across users)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the full create payload (overrides flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*name) == "" {
			return fmt.Errorf("--name is required (or use --body-file)")
		}
		body := map[string]interface{}{
			"name":      *name,
			"is_public": *public,
		}
		if strings.TrimSpace(*description) != "" {
			body["description"] = *description
		}
		if strings.TrimSpace(*content) != "" {
			body["content"] = *content
		}
		if strings.TrimSpace(*category) != "" {
			body["category"] = *category
		}
		if strings.TrimSpace(*userID) != "" {
			body["user_id"] = *userID
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/templates", nil, payload)
	if err != nil {
		return err
	}
	var created support.Template
	if err := support.Decode(respBody, &created); err != nil {
		return err
	}

	changes := []string{}
	if created.Name != "" {
		changes = append(changes, fmt.Sprintf("Name: %s", created.Name))
	}
	if created.ID != "" {
		changes = append(changes, fmt.Sprintf("ID: %s", created.ID))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created template %q", created.Name)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s template list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func templateRows(templates []support.Template) []string {
	if len(templates) == 0 {
		return []string{"No templates found"}
	}
	rows := make([]string, 0, len(templates))
	for _, t := range templates {
		desc := t.Description
		if desc == "" {
			desc = "(no description)"
		}
		public := ""
		if t.IsPublic {
			public = " [public]"
		}
		rows = append(rows, fmt.Sprintf("%s | %s - %s [%s]%s",
			support.ShortID(t.ID), t.Name, desc, t.Category, public))
	}
	return rows
}
