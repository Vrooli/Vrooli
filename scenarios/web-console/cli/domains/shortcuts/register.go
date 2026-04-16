package shortcuts

import (
	"fmt"
	"os"

	"web-console/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `shortcuts` subcommand group covering the effective
// shortcut view and per-profile CRUD under /shortcuts/profiles.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "shortcuts",
		Description: "Inspect and manage keyboard shortcut profiles",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "effective", Aliases: []string{"list"}, Description: "Show effective shortcut bindings for this client", Run: func(args []string) error { return runEffective(core, args) }},
			{Name: "profiles", Description: "List saved shortcut profiles", Run: func(args []string) error { return runProfiles(core, args) }},
			{Name: "upsert", Description: "Create or update a shortcut profile (--body-file PATH)", Run: func(args []string) error { return runUpsert(core, args) }},
			{Name: "delete", Description: "Delete a shortcut profile", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runEffective(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("shortcuts effective")
	osFilter := fs.String("os", "", "OS filter (darwin|linux|windows)")
	layout := fs.String("layout", "", "Keyboard layout filter")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"os":     *osFilter,
		"layout": *layout,
	})
	body, err := core.Get("/shortcuts", query)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Effective shortcuts"},
		ResultsHeading: "Bindings",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{fmt.Sprintf("%s shortcuts profiles", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runProfiles(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("shortcuts profiles")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/shortcuts/profiles", nil)
	if err != nil {
		return err
	}
	var profiles []support.ShortcutProfile
	if err := support.Decode(body, &profiles); err != nil {
		return err
	}

	rows := make([]string, 0, len(profiles))
	for _, p := range profiles {
		rows = append(rows, fmt.Sprintf("%s | %s | os=%s | layout=%s | enabled=%t",
			support.ShortID(p.ID), p.Name, p.OS, p.Layout, p.Enabled))
	}
	if len(rows) == 0 {
		rows = []string{"(no profiles)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Shortcut profiles: %d", len(profiles))},
		ResultsHeading: "Profiles",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s shortcuts upsert --body-file profile.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpsert(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("shortcuts upsert")
	bodyFile := fs.String("body-file", "", "Path to a JSON profile body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request("PUT", "/shortcuts/profiles", nil, payload)
	if err != nil {
		return err
	}
	var profile support.ShortcutProfile
	_ = support.Decode(body, &profile)

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Upserted shortcut profile %s", profile.Name)},
		Changes: []string{
			fmt.Sprintf("ID: %s", profile.ID),
			fmt.Sprintf("OS: %s", profile.OS),
			fmt.Sprintf("Layout: %s", profile.Layout),
		},
		NextCommand: []string{fmt.Sprintf("%s shortcuts profiles", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("shortcuts delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: shortcuts delete <profile-id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/shortcuts/profiles/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted shortcut profile %s", id)},
		NextCommand: []string{fmt.Sprintf("%s shortcuts profiles", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
