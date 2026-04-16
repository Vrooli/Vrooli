package application

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"scenario-authenticator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `application` subcommand group covering the CRUD surface
// plus integration-code generation. All mutating endpoints require an admin
// token on the API side.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "application",
		Description: "Register and manage downstream scenarios/apps",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List registered applications", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Get one application by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "register", Description: "Register a new application (admin; requires --body-file)", Run: func(args []string) error { return runRegister(core, args) }},
			{Name: "update", Description: "Update an application (admin; requires --body-file)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Deactivate an application (admin)", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "integration-code", Description: "Generate integration code for an application", Run: func(args []string) error { return runIntegrationCode(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("application list")
	stats := fs.Bool("stats", false, "Return usage statistics instead of the full record")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{})
	if *stats {
		query.Set("stats", "true")
	}

	body, err := core.Get("/applications", query)
	if err != nil {
		return err
	}

	var envelope support.ApplicationsListResponse
	if err := support.Decode(body, &envelope); err != nil {
		return err
	}

	var results []string
	if *stats {
		var apps []support.ApplicationStats
		if err := json.Unmarshal(envelope.Applications, &apps); err != nil {
			return fmt.Errorf("parse stats response: %w", err)
		}
		results = statsRows(apps)
	} else {
		var apps []support.Application
		if err := json.Unmarshal(envelope.Applications, &apps); err != nil {
			return fmt.Errorf("parse applications response: %w", err)
		}
		results = applicationRows(apps)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Applications: %d", envelope.Total)},
		ResultsHeading: "Applications",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s application get <id>", support.CLIName),
			fmt.Sprintf("%s application list --stats", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("application get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: application get <application-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/applications/"+id, nil)
	if err != nil {
		return err
	}

	var app support.Application
	if err := support.Decode(body, &app); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", app.ID),
		fmt.Sprintf("Name: %s", app.Name),
		fmt.Sprintf("Display name: %s", app.DisplayName),
		fmt.Sprintf("Type: %s", app.ScenarioType),
		fmt.Sprintf("Active: %t", app.IsActive),
		fmt.Sprintf("Allowed origins: %s", support.JoinStrings(app.AllowedOrigins, "-")),
		fmt.Sprintf("Redirect URIs: %s", support.JoinStrings(app.RedirectURIs, "-")),
		fmt.Sprintf("Permissions: %s", support.JoinStrings(app.Permissions, "-")),
		fmt.Sprintf("Rate limit: %d/min", app.RateLimit),
		fmt.Sprintf("Total users: %d", app.TotalUsers),
		fmt.Sprintf("Total authentications: %d", app.TotalAuths),
		fmt.Sprintf("Last accessed: %s", support.FormatTimePtr(app.LastAccessed)),
	}
	if app.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", app.Description))
	}
	if app.MaxUsers != nil {
		results = append(results, fmt.Sprintf("Max users: %d", *app.MaxUsers))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Application: %s", app.DisplayName)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s application integration-code %s --type api", support.CLIName, app.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRegister(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("application register")
	bodyFile := fs.String("body-file", "", "Path to JSON file with RegisterAppRequest payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/applications", nil, raw)
	if err != nil {
		return err
	}

	var creds support.AppCredentials
	if err := support.Decode(body, &creds); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Application ID: %s", creds.ApplicationID),
			fmt.Sprintf("API key: %s", creds.APIKey),
			fmt.Sprintf("API secret: %s", creds.APISecret),
			"These credentials are only displayed once — store them securely.",
		},
		Changes: []string{fmt.Sprintf("Application %s registered", creds.ApplicationID)},
		NextCommand: []string{
			fmt.Sprintf("%s application get %s", support.CLIName, creds.ApplicationID),
			fmt.Sprintf("%s application integration-code %s", support.CLIName, creds.ApplicationID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("application update")
	bodyFile := fs.String("body-file", "", "Path to JSON file with update payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: application update <application-id> --body-file <path>")
	}
	id := fs.Arg(0)

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("PUT", "/applications/"+id, nil, raw)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Application %s updated", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Application %s updated", id)},
		NextCommand: []string{fmt.Sprintf("%s application get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("application delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: application delete <application-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/applications/"+id, nil, nil)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Application %s deactivated", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Application %s deactivated", id)},
		NextCommand: []string{fmt.Sprintf("%s application list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runIntegrationCode(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("application integration-code")
	integrationType := fs.String("type", "api", "Integration type: api|ui|workflow")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: application integration-code <application-id> [--type api|ui|workflow]")
	}
	id := fs.Arg(0)

	query := support.BuildQuery(map[string]string{"type": strings.TrimSpace(*integrationType)})
	body, err := core.Get("/applications/"+id+"/integration-code", query)
	if err != nil {
		return err
	}

	var resp support.IntegrationCode
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Application: %s", resp.ApplicationName),
			fmt.Sprintf("Integration type: %s", resp.IntegrationType),
		},
		ResultsHeading: "Code",
		Results:        strings.Split(strings.TrimRight(resp.Code, "\n"), "\n"),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func applicationRows(apps []support.Application) []string {
	if len(apps) == 0 {
		return []string{"No applications registered"}
	}
	rows := make([]string, 0, len(apps))
	for _, a := range apps {
		rows = append(rows, fmt.Sprintf("%s (%s) | type=%s | active=%t | users=%d",
			a.Name, support.ShortID(a.ID), a.ScenarioType, a.IsActive, a.TotalUsers))
	}
	return rows
}

func statsRows(apps []support.ApplicationStats) []string {
	if len(apps) == 0 {
		return []string{"No applications registered"}
	}
	rows := make([]string, 0, len(apps))
	for _, a := range apps {
		rows = append(rows, fmt.Sprintf("%s (%s) | active=%t | users=%d | sessions=%d | events_today=%d",
			a.Name, support.ShortID(a.ID), a.IsActive, a.TotalUsers, a.ActiveSessions, a.TotalEventsToday))
	}
	return rows
}
