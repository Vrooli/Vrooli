package escalations

import (
	"fmt"
	"os"

	"ai-chatbot-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wires `ai-chatbot-manager escalations ...` covering list/update.
// list hits /chatbots/{id}/escalations; update hits /escalations/{id}.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "escalations",
		Description: "Inspect and update chatbot escalations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List pending escalations for a chatbot", Run: func(args []string) error { return runList(core, args) }},
			{Name: "update", Description: "Update an escalation's status", Run: func(args []string) error { return runUpdate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("escalations list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: escalations list <chatbot-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/chatbots/"+id+"/escalations", nil)
	if err != nil {
		return err
	}
	var items []support.Escalation
	if err := support.Decode(body, &items); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Escalations for %s: %d", id, len(items))},
		ResultsHeading: "Escalations",
		Results:        escalationRows(items),
		RetrievalHints: []string{
			fmt.Sprintf("%s escalations update <escalation-id> --status resolved", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("escalations update")
	status := fs.String("status", "", "New status: pending|in_progress|resolved|dismissed (required)")
	notes := fs.String("notes", "", "Resolution notes")
	bodyFile := fs.String("body-file", "", "Path to update JSON (overrides --status/--notes)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: escalations update <escalation-id> --status STATUS [--notes TEXT] | --body-file PATH")
	}
	id := fs.Arg(0)

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *status == "" {
			return fmt.Errorf("--status STATUS is required (or supply --body-file PATH)")
		}
		payload = map[string]interface{}{
			"status": *status,
			"notes":  *notes,
		}
	}

	body, err := core.Request("PATCH", "/escalations/"+id, nil, payload)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Escalation %s updated", id)
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Escalation %s -> status=%s", id, *status)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func escalationRows(items []support.Escalation) []string {
	if len(items) == 0 {
		return []string{"(no escalations)"}
	}
	rows := make([]string, 0, len(items))
	for _, e := range items {
		rows = append(rows, fmt.Sprintf("%s | status=%s | type=%s | conf=%.2f | at=%s | reason=%s",
			support.ShortID(e.ID), e.Status, e.EscalationType, e.ConfidenceScore,
			support.FormatTimeValue(e.EscalatedAt), e.Reason))
	}
	return rows
}
