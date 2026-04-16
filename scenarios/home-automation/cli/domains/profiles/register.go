package profiles

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"home-automation/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "profiles",
		Description: "Inspect home profiles and their permissions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List profiles", Run: func(args []string) error { return runList(core, args) }},
			{Name: "permissions", Description: "Show one profile permission document", Run: func(args []string) error { return runPermissions(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/profiles", nil)
	if err != nil {
		return err
	}

	var response support.ProfilesResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	sort.SliceStable(response.Profiles, func(i, j int) bool {
		return response.Profiles[i].Name < response.Profiles[j].Name
	})

	results := make([]string, 0, len(response.Profiles))
	for _, profile := range response.Profiles {
		results = append(results, fmt.Sprintf("%s | %s | type=%s | permissions=%s", profile.ID, firstNonEmpty(profile.Name, profile.ID), firstNonEmpty(profile.Type, "unknown"), support.FormatMapInline(profile.Permissions)))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Profiles returned: %d", len(response.Profiles)), fmt.Sprintf("Server count: %d", response.Count)},
		ResultsHeading: "Profiles",
		Results:        results,
		RetrievalHints: []string{"home-automation profiles permissions <profile-id>", "home-automation devices control <device-id> turn_on --profile <profile-id>"},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runPermissions(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles permissions")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: home-automation profiles permissions <profile-id> [--json]")
	}

	profileID := strings.TrimSpace(fs.Arg(0))
	body, err := core.Get("/profiles/"+profileID+"/permissions", nil)
	if err != nil {
		return err
	}

	var response map[string]interface{}
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	results := make([]string, 0, len(response))
	keys := make([]string, 0, len(response))
	for key := range response {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		results = append(results, fmt.Sprintf("%s=%v", key, response[key]))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Profile: %s", profileID)},
		ResultsHeading: "Permissions",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("home-automation devices control <device-id> turn_on --profile %s", profileID)},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
