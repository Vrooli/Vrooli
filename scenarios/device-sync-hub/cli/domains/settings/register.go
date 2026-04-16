package settings

import (
	"os"

	"device-sync-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `device-sync-hub settings` as a flat command. The API
// is the source of truth for configured limits and user statistics; this
// wrapper renders whatever shape the API returns.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Settings",
		Commands: []cliapp.Command{
			{
				Name:        "settings",
				Description: "Show server settings, limits, and user stats",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runShow(core, args) },
			},
		},
	}
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/sync/settings", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Device Sync Hub settings"},
		ResultsHeading: "Settings",
		Results:        support.MapRows(payload),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
