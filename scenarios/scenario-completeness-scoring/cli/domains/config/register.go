package config

import (
	"fmt"
	"os"

	"scenario-completeness-scoring/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `config` subcommand group for server-side scoring
// configuration (weights, thresholds, schema). Note: this is distinct from the
// built-in `configure` command (CLI local settings like api_base).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "config",
		Description: "View and update server-side scoring configuration",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "show", Aliases: []string{"get"}, Description: "Show current scoring configuration", Run: func(args []string) error { return runShow(core, args) }},
			{Name: "update", Description: "Replace the scoring configuration (requires --body-file)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "schema", Description: "Show the scoring configuration JSON schema", Run: func(args []string) error { return runSchema(core, args) }},
			{Name: "reset", Description: "Reset scoring configuration to defaults", Run: func(args []string) error { return runReset(core, args) }},
			{Name: "thresholds", Description: "Show configured classification thresholds (optionally for a category)", Run: func(args []string) error { return runThresholds(core, args) }},
		},
	}
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config show")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/config", nil)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Scoring configuration loaded"},
		ResultsHeading: "Configuration",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s config schema", support.CLIName),
			fmt.Sprintf("%s config thresholds", support.CLIName),
			fmt.Sprintf("%s config update --body-file config.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config update")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the new configuration payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	respBody, err := core.Request("PUT", "/config", nil, raw)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(respBody)
	if message == "" {
		message = "Scoring configuration updated"
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{"Server-side scoring weights and components were updated.", "Source: " + *bodyFile},
		NextCommand: []string{
			fmt.Sprintf("%s config show", support.CLIName),
			fmt.Sprintf("%s score list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSchema(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config schema")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/config/schema", nil)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Scoring configuration schema"},
		ResultsHeading: "Schema",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s config show", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runReset(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config reset")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/config/reset", nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Scoring configuration reset to defaults"
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{"All scoring weights and thresholds restored to their defaults."},
		NextCommand: []string{
			fmt.Sprintf("%s config show", support.CLIName),
			fmt.Sprintf("%s score list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runThresholds(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config thresholds")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	path := "/config/thresholds"
	label := "All categories"
	if fs.NArg() >= 1 {
		category := fs.Arg(0)
		path = "/config/thresholds/" + category
		label = "Category: " + category
	}

	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Scoring thresholds", label},
		ResultsHeading: "Thresholds",
		Results:        support.JSONLines(body),
		RetrievalHints: []string{
			fmt.Sprintf("%s config show", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
