package actions

import (
	"fmt"
	"os"

	"bookmark-intelligence-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `action` subcommand group for list/approve/reject.
// Approve and reject POST to `/actions/approve` and `/actions/reject` with
// `{ "action_ids": [...] }`. A positional <action-id> covers the common
// single-action case; `--body-file` lets callers submit batches without
// hand-building JSON in the CLI.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "action",
		Description: "Manage pending automation actions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List pending actions", Run: func(args []string) error { return runList(core, args) }},
			{Name: "approve", Description: "Approve one action (or --body-file for batches)", Run: func(args []string) error { return runDecision(core, args, "approve", "approved") }},
			{Name: "reject", Description: "Reject one action (or --body-file for batches)", Run: func(args []string) error { return runDecision(core, args, "reject", "rejected") }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("action list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/actions", nil)
	if err != nil {
		return err
	}

	var actions []map[string]interface{}
	if err := support.Decode(body, &actions); err != nil {
		return err
	}

	rows := make([]string, 0, len(actions))
	for _, a := range actions {
		id, _ := a["id"].(string)
		title, _ := a["title"].(string)
		status, _ := a["status"].(string)
		rows = append(rows, fmt.Sprintf("%s | %s | status=%s", support.ShortID(id), title, status))
	}
	if len(rows) == 0 {
		rows = []string{"(no pending actions)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Pending actions: %d", len(actions))},
		ResultsHeading: "Actions",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s action approve <action-id>", support.CLIName),
			fmt.Sprintf("%s action reject <action-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runDecision(core *cliapp.ScenarioApp, args []string, verb, pastTense string) error {
	fs := support.NewFlagSet("action " + verb)
	bodyFile := fs.String("body-file", "", "Optional JSON file with the request body (overrides positional <action-id>)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	switch {
	case *bodyFile != "":
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	case fs.NArg() >= 1:
		payload = map[string]interface{}{"action_ids": []string{fs.Arg(0)}}
	default:
		return fmt.Errorf("usage: action %s <action-id> | action %s --body-file payload.json", verb, verb)
	}

	body, err := core.Request("POST", "/actions/"+verb, nil, payload)
	if err != nil {
		return err
	}

	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := fmt.Sprintf("Actions %s", pastTense)
	if msg, ok := resp["message"].(string); ok && msg != "" {
		result = msg
	}

	changes := []string{}
	countKey := pastTense + "_count"
	if v, ok := resp[countKey].(float64); ok {
		changes = append(changes, fmt.Sprintf("%s: %d", countKey, int(v)))
	}

	report := cliapp.MutationReport{
		Result:      []string{result},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s action list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
