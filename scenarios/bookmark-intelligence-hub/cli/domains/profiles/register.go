package profiles

import (
	"fmt"
	"os"

	"bookmark-intelligence-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `profile` subcommand group covering list/get/stats/create.
// The API is the source of truth for profile state; this package is a thin
// wrapper that formats responses through the standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "profile",
		Description: "Manage user profiles and settings",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List profiles", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"view", "show"}, Description: "Show one profile", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "stats", Description: "Show profile statistics", Run: func(args []string) error { return runStats(core, args) }},
			{Name: "create", Description: "Create a profile (request body via --body-file)", Run: func(args []string) error { return runCreate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profile list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/profiles", nil)
	if err != nil {
		return err
	}
	var profiles []support.Profile
	if err := support.Decode(body, &profiles); err != nil {
		return err
	}

	rows := make([]string, 0, len(profiles))
	for _, p := range profiles {
		rows = append(rows, fmt.Sprintf("%s | %s | %s", support.ShortID(p.ID), p.Name, p.Description))
	}
	if len(rows) == 0 {
		rows = []string{"(no profiles)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Profiles: %d", len(profiles))},
		ResultsHeading: "Profiles",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s profile get <profile-id>", support.CLIName),
			fmt.Sprintf("%s profile stats <profile-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profile get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: profile get <profile-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/profiles/"+id, nil)
	if err != nil {
		return err
	}
	var profile support.Profile
	if err := support.Decode(body, &profile); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", profile.ID),
		fmt.Sprintf("Name: %s", profile.Name),
	}
	if profile.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", profile.Description))
	}
	if profile.CreatedAt != nil {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTimeValue(*profile.CreatedAt)))
	}
	if len(profile.Settings) > 0 {
		results = append(results, "Settings:")
		for _, row := range support.MapRows(profile.Settings) {
			results = append(results, "  "+row)
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Profile: %s", profile.Name)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s profile stats %s", support.CLIName, profile.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profile stats")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: profile stats <profile-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/profiles/"+id+"/stats", nil)
	if err != nil {
		return err
	}
	var stats support.ProfileStats
	if err := support.Decode(body, &stats); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Total bookmarks: %d", stats.TotalBookmarks),
		fmt.Sprintf("Categories: %d", stats.CategoriesCount),
		fmt.Sprintf("Pending actions: %d", stats.PendingActions),
		fmt.Sprintf("Accuracy rate: %.2f%%", stats.AccuracyRate),
	}
	if stats.LastSyncAt != nil {
		results = append(results, fmt.Sprintf("Last sync: %s", support.FormatTimeValue(*stats.LastSyncAt)))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Statistics for profile %s", id)},
		ResultsHeading: "Metrics",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s profile get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profile create")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the request body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/profiles", nil, payload)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Profile creation submitted"
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		NextCommand: []string{fmt.Sprintf("%s profile list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
