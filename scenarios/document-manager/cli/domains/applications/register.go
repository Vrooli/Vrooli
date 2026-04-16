package applications

import (
	"fmt"
	"os"

	"document-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `applications` subcommand group wrapping /api/applications.
// The API is the source of truth for validation; this package only formats
// requests and responses through the standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "applications",
		Description: "List and manage applications tracked by document-manager",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List applications", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create an application (flags or --body-file)", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete an application by ID", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("applications list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/applications", nil)
	if err != nil {
		return err
	}
	var apps []support.Application
	if err := support.Decode(body, &apps); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Applications: %d", len(apps))},
		ResultsHeading: "Applications",
		Results:        appRows(apps),
		RetrievalHints: []string{
			fmt.Sprintf("%s applications create --name <name> --repository <url>", support.CLIName),
			fmt.Sprintf("%s agents list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("applications create")
	name := fs.String("name", "", "Application name")
	repo := fs.String("repository", "", "Repository URL")
	docsPath := fs.String("docs-path", "/docs", "Documentation path within the repository")
	bodyFile := fs.String("body-file", "", "Optional JSON request body (overrides flag inputs); use '-' for stdin")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *name == "" || *repo == "" {
			return fmt.Errorf("usage: applications create --name <name> --repository <url> [--docs-path /docs] | --body-file <path>")
		}
		payload = map[string]interface{}{
			"name":               *name,
			"repository_url":     *repo,
			"documentation_path": *docsPath,
		}
	}

	body, err := core.Request("POST", "/applications", nil, payload)
	if err != nil {
		return err
	}
	var created support.Application
	if err := support.Decode(body, &created); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Application %q created", created.Name)
	}
	changes := []string{}
	if created.ID != "" {
		changes = append(changes, fmt.Sprintf("ID: %s", created.ID))
	}
	if created.Name != "" {
		changes = append(changes, fmt.Sprintf("Name: %s", created.Name))
	}
	if created.RepositoryURL != "" {
		changes = append(changes, fmt.Sprintf("Repository: %s", created.RepositoryURL))
	}
	if created.DocumentationPath != "" {
		changes = append(changes, fmt.Sprintf("Docs path: %s", created.DocumentationPath))
	}
	changes = append(changes, fmt.Sprintf("Created: %s", support.FormatTimeValue(created.CreatedAt)))

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s applications list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("applications delete")
	id := fs.String("id", "", "Application ID (or pass as positional argument)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	appID := *id
	if appID == "" && fs.NArg() >= 1 {
		appID = fs.Arg(0)
	}
	if appID == "" {
		return fmt.Errorf("usage: applications delete <id> | --id <id>")
	}

	query := support.BuildQuery(map[string]string{"id": appID})
	body, err := core.Request("DELETE", "/applications", query, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Application %s deleted", appID)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Deleted application %s (and cascaded agents/queue items)", appID)},
		NextCommand: []string{fmt.Sprintf("%s applications list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func appRows(apps []support.Application) []string {
	if len(apps) == 0 {
		return []string{"No applications registered"}
	}
	rows := make([]string, 0, len(apps))
	for _, a := range apps {
		status := a.Status
		if status == "" {
			if a.Active {
				status = "active"
			} else {
				status = "inactive"
			}
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | health=%.2f | agents=%d | repo=%s",
			a.Name, support.ShortID(a.ID), status, a.HealthScore, a.AgentCount, a.RepositoryURL))
	}
	return rows
}
