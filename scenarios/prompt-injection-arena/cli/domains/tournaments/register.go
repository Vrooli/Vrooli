package tournaments

import (
	"fmt"
	"os"

	"prompt-injection-arena/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `tournaments` subcommand group covering the
// /api/v1/tournaments* surface: listing, creating, running, and retrieving
// results of tournaments.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "tournaments",
		Description: "Schedule and inspect injection tournaments",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List tournaments", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a new tournament", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "run", Description: "Run a tournament by ID", Run: func(args []string) error { return runRun(core, args) }},
			{Name: "results", Description: "Show tournament results by ID", Run: func(args []string) error { return runResults(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tournaments list")
	status := fs.String("status", "", "Filter by tournament status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{"status": *status})
	body, err := core.Get("/tournaments", query)
	if err != nil {
		return err
	}

	var resp support.TournamentListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tournaments: %d", resp.Count)},
		ResultsHeading: "Tournaments",
		Results:        tournamentRows(resp.Tournaments),
		RetrievalHints: []string{
			fmt.Sprintf("%s tournaments results <id>", support.CLIName),
			fmt.Sprintf("%s tournaments run <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tournaments create")
	name := fs.String("name", "", "Tournament name (required)")
	description := fs.String("description", "", "Tournament description")
	scheduledAt := fs.String("scheduled-at", "", "Scheduled time in RFC3339 format")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
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
		if *name == "" {
			return fmt.Errorf("usage: tournaments create --name NAME [--description TEXT] [--scheduled-at RFC3339] [--body-file PATH]")
		}
		req := map[string]interface{}{
			"name":        *name,
			"description": *description,
		}
		if *scheduledAt != "" {
			req["scheduled_at"] = *scheduledAt
		}
		payload = req
	}

	body, err := core.Request("POST", "/tournaments", nil, payload)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Tournament created"
	}

	var created struct {
		Tournament support.Tournament `json:"tournament"`
	}
	_ = support.Decode(body, &created)

	changes := []string{}
	if created.Tournament.ID != "" {
		changes = append(changes, fmt.Sprintf("Tournament ID: %s", created.Tournament.ID))
	}
	if created.Tournament.Name != "" {
		changes = append(changes, fmt.Sprintf("Name: %s", created.Tournament.Name))
	}
	if created.Tournament.Status != "" {
		changes = append(changes, fmt.Sprintf("Status: %s", created.Tournament.Status))
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s tournaments run %s", support.CLIName, created.Tournament.ID),
			fmt.Sprintf("%s tournaments results %s", support.CLIName, created.Tournament.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runRun(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tournaments run")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tournaments run <tournament-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/tournaments/"+id+"/run", nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Tournament %s started", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Tournament %s: run requested", id)},
		NextCommand: []string{fmt.Sprintf("%s tournaments results %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runResults(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tournaments results")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tournaments results <tournament-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/tournaments/"+id+"/results", nil)
	if err != nil {
		return err
	}

	var resp support.TournamentResultsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Tournament: %s (%s)", resp.Name, resp.Status),
		fmt.Sprintf("Results: %d", resp.Count),
	}
	if resp.CompletedAt != nil {
		summary = append(summary, fmt.Sprintf("Completed: %s", support.FormatTimeValue(*resp.CompletedAt)))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Results",
		Results:        resultRows(resp.Results),
		RetrievalHints: []string{
			fmt.Sprintf("%s tournaments list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func tournamentRows(items []support.Tournament) []string {
	if len(items) == 0 {
		return []string{"(no tournaments)"}
	}
	rows := make([]string, 0, len(items))
	for _, t := range items {
		scheduled := "unscheduled"
		if t.ScheduledAt != nil {
			scheduled = support.FormatTimeValue(*t.ScheduledAt)
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | status=%s | scheduled=%s",
			t.Name, support.ShortID(t.ID), t.Status, scheduled))
	}
	return rows
}

func resultRows(items []support.TournamentResult) []string {
	if len(items) == 0 {
		return []string{"(no results yet)"}
	}
	rows := make([]string, 0, len(items))
	for _, r := range items {
		rows = append(rows, fmt.Sprintf("agent=%s | injection=%s | success=%t | score=%.2f | %dms | %s",
			support.ShortID(r.AgentID), support.ShortID(r.InjectionID), r.Success, r.Score,
			r.ExecutionTimeMS, support.FormatTimeValue(r.TestedAt)))
	}
	return rows
}
