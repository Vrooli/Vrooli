package workspace

import (
	"fmt"
	"os"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `workspace` subcommand group covering the shared pane
// layout and group/pane mutations exposed under /workspace.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "workspace",
		Description: "Manage pane-based workspace layout and groups",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "layout-get", Aliases: []string{"layout"}, Description: "Show the current workspace layout", Run: func(args []string) error { return runLayoutGet(core, args) }},
			{Name: "layout-save", Description: "Save a workspace layout (--body-file PATH)", Run: func(args []string) error { return runLayoutSave(core, args) }},
			{Name: "pane-update", Description: "Update a pane assignment (--body-file PATH)", Run: func(args []string) error { return runPaneUpdate(core, args) }},
			{Name: "pane-delete", Description: "Remove a pane assignment", Run: func(args []string) error { return runPaneDelete(core, args) }},
			{Name: "group-create", Description: "Create a workspace group (--body-file PATH)", Run: func(args []string) error { return runGroupCreate(core, args) }},
			{Name: "group-update", Description: "Update a workspace group (--body-file PATH)", Run: func(args []string) error { return runGroupUpdate(core, args) }},
			{Name: "group-delete", Description: "Delete a workspace group", Run: func(args []string) error { return runGroupDelete(core, args) }},
		},
	}
}

func runLayoutGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workspace layout-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/workspace/layout", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Workspace layout"},
		ResultsHeading: "Layout",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{fmt.Sprintf("%s workspace layout-save --body-file layout.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runLayoutSave(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workspace layout-save")
	bodyFile := fs.String("body-file", "", "Path to a JSON layout body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("PUT", "/workspace/layout", nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Workspace layout saved"},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runPaneUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workspace pane-update")
	bodyFile := fs.String("body-file", "", "Path to a JSON pane body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: workspace pane-update <session-id> --body-file PATH")
	}
	sessionID := fs.Arg(0)

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("PUT", "/workspace/panes/"+sessionID, nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated pane for session %s", sessionID)},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runPaneDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workspace pane-delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: workspace pane-delete <session-id>")
	}
	sessionID := fs.Arg(0)

	if _, err := core.Request("DELETE", "/workspace/panes/"+sessionID, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Removed pane for session %s", sessionID)},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runGroupCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workspace group-create")
	bodyFile := fs.String("body-file", "", "Path to a JSON group body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", "/workspace/groups", nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	_ = support.Decode(body, &resp)

	report := cliapp.MutationReport{
		Result:      []string{"Created workspace group"},
		Changes:     support.MapRows(resp),
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runGroupUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workspace group-update")
	bodyFile := fs.String("body-file", "", "Path to a JSON group body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: workspace group-update <group-id> --body-file PATH")
	}
	id := fs.Arg(0)

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("PUT", "/workspace/groups/"+id, nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated workspace group %s", id)},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runGroupDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workspace group-delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: workspace group-delete <group-id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/workspace/groups/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted workspace group %s", id)},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
