package agents

import (
	"fmt"
	"os"
	"strconv"

	"document-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `agents` subcommand group wrapping /api/agents.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "agents",
		Description: "List and manage documentation agents",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List agents", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create an agent (flags or --body-file)", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete an agent by ID", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agents list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/agents", nil)
	if err != nil {
		return err
	}
	var list []support.Agent
	if err := support.Decode(body, &list); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Agents: %d", len(list))},
		ResultsHeading: "Agents",
		Results:        agentRows(list),
		RetrievalHints: []string{
			fmt.Sprintf("%s agents create --name <name> --type <type> --application <id>", support.CLIName),
			fmt.Sprintf("%s queue list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agents create")
	name := fs.String("name", "", "Agent name")
	agentType := fs.String("type", "", "Agent type (e.g., link_validator)")
	appID := fs.String("application", "", "Application ID the agent belongs to")
	schedule := fs.String("schedule", "0 */6 * * *", "Cron schedule for agent runs")
	threshold := fs.Float64("threshold", 0.8, "Auto-apply confidence threshold (0.0-1.0)")
	configuration := fs.String("configuration", "{}", "Agent configuration JSON blob")
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
		if *name == "" || *agentType == "" || *appID == "" {
			return fmt.Errorf("usage: agents create --name <name> --type <type> --application <id> [--schedule cron] [--threshold 0.8] [--configuration '{}'] | --body-file <path>")
		}
		payload = map[string]interface{}{
			"name":                 *name,
			"type":                 *agentType,
			"application_id":       *appID,
			"schedule_cron":        *schedule,
			"auto_apply_threshold": *threshold,
			"configuration":        *configuration,
		}
	}

	body, err := core.Request("POST", "/agents", nil, payload)
	if err != nil {
		return err
	}
	var created support.Agent
	if err := support.Decode(body, &created); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Agent %q created", created.Name)
	}
	changes := []string{}
	if created.ID != "" {
		changes = append(changes, fmt.Sprintf("ID: %s", created.ID))
	}
	if created.Name != "" {
		changes = append(changes, fmt.Sprintf("Name: %s", created.Name))
	}
	if created.Type != "" {
		changes = append(changes, fmt.Sprintf("Type: %s", created.Type))
	}
	if created.ApplicationID != "" {
		changes = append(changes, fmt.Sprintf("Application: %s", created.ApplicationID))
	}
	if created.ScheduleCron != "" {
		changes = append(changes, fmt.Sprintf("Schedule: %s", created.ScheduleCron))
	}
	changes = append(changes, fmt.Sprintf("Threshold: %s", strconv.FormatFloat(created.AutoApplyThreshold, 'f', -1, 64)))
	changes = append(changes, fmt.Sprintf("Created: %s", support.FormatTimeValue(created.CreatedAt)))

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s agents list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("agents delete")
	id := fs.String("id", "", "Agent ID (or pass as positional argument)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	agentID := *id
	if agentID == "" && fs.NArg() >= 1 {
		agentID = fs.Arg(0)
	}
	if agentID == "" {
		return fmt.Errorf("usage: agents delete <id> | --id <id>")
	}

	query := support.BuildQuery(map[string]string{"id": agentID})
	body, err := core.Request("DELETE", "/agents", query, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Agent %s deleted", agentID)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Deleted agent %s (and cascaded queue items)", agentID)},
		NextCommand: []string{fmt.Sprintf("%s agents list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func agentRows(agents []support.Agent) []string {
	if len(agents) == 0 {
		return []string{"No agents registered"}
	}
	rows := make([]string, 0, len(agents))
	for _, a := range agents {
		status := a.Status
		if status == "" {
			if a.Enabled {
				status = "enabled"
			} else {
				status = "disabled"
			}
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | type=%s | app=%s | schedule=%s",
			a.Name, support.ShortID(a.ID), status, a.Type, a.ApplicationName, a.ScheduleCron))
	}
	return rows
}
