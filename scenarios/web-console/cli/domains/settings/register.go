package settings

import (
	"fmt"
	"os"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `settings` subcommand group for scenario-wide
// configuration (currently: session defaults).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Inspect and update web-console settings",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "session-defaults-get", Aliases: []string{"session-defaults"}, Description: "Show default values applied to new sessions", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "session-defaults-set", Description: "Update default session settings (--body-file PATH)", Run: func(args []string) error { return runSet(core, args) }},
		},
	}
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings session-defaults-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/settings/session-defaults", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Session defaults"},
		ResultsHeading: "Values",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{fmt.Sprintf("%s settings session-defaults-set --body-file defaults.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings session-defaults-set")
	bodyFile := fs.String("body-file", "", "Path to a JSON body with session defaults (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("PUT", "/settings/session-defaults", nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Updated session defaults"},
		NextCommand: []string{fmt.Sprintf("%s settings session-defaults-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
