package accounts

import (
	"fmt"
	"os"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `social-media-scheduler accounts` as a flat command because
// /api/v1/user/accounts is a single read-only surface. OAuth linking lives
// under `oauth connect`/`oauth disconnect`.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Accounts",
		Commands: []cliapp.Command{
			{
				Name:        "accounts",
				Description: "List connected social media accounts",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("accounts")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/user/accounts", nil)
	if err != nil {
		return err
	}
	var accounts []support.SocialAccount
	if err := support.Decode(body, &accounts); err != nil {
		return err
	}

	rows := make([]string, 0, len(accounts))
	for _, a := range accounts {
		active := "inactive"
		if a.IsActive {
			active = "active"
		}
		rows = append(rows, fmt.Sprintf("%s | @%s | %s | %s", a.Platform, a.Username, a.DisplayName, active))
	}
	if len(rows) == 0 {
		rows = []string{"No accounts connected"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Connected accounts: %d", len(accounts))},
		ResultsHeading: "Accounts",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s oauth connect <platform>", support.CLIName),
			fmt.Sprintf("%s oauth disconnect <platform>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
