package lifecycle

import (
	"fmt"
	"os"

	"landing-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `lifecycle` subcommand group for generated-scenario lifecycle
// controls exposed at `/api/v1/lifecycle/{scenario_id}/...`.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "lifecycle",
		Description: "Control lifecycle of generated landing-page scenarios",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "start", Description: "Start a generated scenario", Run: func(args []string) error { return runAction(core, args, "start", "Start") }},
			{Name: "stop", Description: "Stop a generated scenario", Run: func(args []string) error { return runAction(core, args, "stop", "Stop") }},
			{Name: "restart", Description: "Restart a generated scenario", Run: func(args []string) error { return runAction(core, args, "restart", "Restart") }},
			{Name: "promote", Description: "Promote a generated scenario", Run: func(args []string) error { return runAction(core, args, "promote", "Promote") }},
			{Name: "status", Description: "Show lifecycle status for a generated scenario", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "logs", Description: "Show lifecycle logs for a generated scenario", Run: func(args []string) error { return runLogs(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a generated scenario", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runAction(core *cliapp.ScenarioApp, args []string, verb, display string) error {
	fs := support.NewFlagSet("lifecycle " + verb)
	bodyFile := fs.String("body-file", "", "Optional JSON request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: lifecycle %s <scenario-id> [--body-file PATH]", verb)
	}
	id := fs.Arg(0)

	raw, err := support.ReadJSONFile(*bodyFile, false)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/lifecycle/"+id+"/"+verb, nil, raw)
	if err != nil {
		return err
	}

	var decoded map[string]interface{}
	if err := support.Decode(body, &decoded); err != nil {
		decoded = map[string]interface{}{}
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		if v, ok := decoded["message"].(string); ok && v != "" {
			message = v
		} else {
			message = fmt.Sprintf("%s issued for %s", display, id)
		}
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Scenario %s: %s", id, verb)},
		NextCommand: []string{
			fmt.Sprintf("%s lifecycle status %s", support.CLIName, id),
			fmt.Sprintf("%s lifecycle logs %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("lifecycle status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: lifecycle status <scenario-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/lifecycle/"+id+"/status", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Lifecycle status for " + id},
		ResultsHeading: "Status",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{
			fmt.Sprintf("%s lifecycle logs %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runLogs(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("lifecycle logs")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: lifecycle logs <scenario-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/lifecycle/"+id+"/logs", nil)
	if err != nil {
		return err
	}

	var payload interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	results := []string{}
	switch v := payload.(type) {
	case []interface{}:
		if len(v) == 0 {
			results = []string{"(no log lines returned)"}
		} else {
			for _, item := range v {
				results = append(results, support.RenderValue(item))
			}
		}
	case map[string]interface{}:
		results = support.MapRows(v)
	default:
		results = []string{support.RenderValue(v)}
	}

	report := cliapp.ListReport{
		Summary:        []string{"Lifecycle logs for " + id},
		ResultsHeading: "Logs",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s lifecycle status %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("lifecycle delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: lifecycle delete <scenario-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/lifecycle/"+id, nil, nil)
	if err != nil {
		return err
	}

	var decoded map[string]interface{}
	if err := support.Decode(body, &decoded); err != nil {
		decoded = map[string]interface{}{}
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		if v, ok := decoded["message"].(string); ok && v != "" {
			message = v
		} else {
			message = "Deleted " + id
		}
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Scenario %s: deleted", id)},
		NextCommand: []string{
			fmt.Sprintf("%s generated", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
