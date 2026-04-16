package user

import (
	"os"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wraps /api/v1/user/stats and /api/v1/user/preferences. The
// preferences payload is a free-form map, so updates use --body-file.
// /api/v1/user/accounts is surfaced as the flat `accounts` command.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "user",
		Description: "Inspect and update the authenticated user",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "stats", Description: "Show user stats and usage", Run: func(args []string) error { return runStats(core, args) }},
			{Name: "preferences", Description: "Update user preferences (--body-file PATH)", Run: func(args []string) error { return runPreferences(core, args) }},
		},
	}
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("user stats")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/user/stats", nil)
	if err != nil {
		return err
	}
	var generic map[string]interface{}
	_ = support.Decode(body, &generic)

	report := cliapp.ListReport{
		Summary:        []string{"User stats"},
		ResultsHeading: "Stats",
		Results:        support.MapRows(generic),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runPreferences(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("user preferences")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the preferences payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("PUT", "/user/preferences", nil, payload)
	if err != nil {
		return err
	}
	msg := support.EnvelopeMessage(body)
	if msg == "" {
		msg = "Updated user preferences"
	}
	report := cliapp.MutationReport{
		Result:  []string{msg},
		Changes: []string{"PUT /user/preferences"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
