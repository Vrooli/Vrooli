package admin

import (
	"fmt"
	"os"

	"prompt-injection-arena/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `admin` subcommand group covering /api/v1/admin/*
// operational endpoints.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "admin",
		Description: "Administrative operations for arena maintenance",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "cleanup-test-data", Description: "Delete test injections and their test results", Run: func(args []string) error { return runCleanup(core, args) }},
		},
	}
}

func runCleanup(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("admin cleanup-test-data")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/admin/cleanup-test-data", nil, nil)
	if err != nil {
		return err
	}

	var resp support.AdminCleanupResponse
	_ = support.Decode(body, &resp)

	message := resp.Message
	if message == "" {
		message = support.EnvelopeMessage(body)
	}
	if message == "" {
		message = "Cleanup complete"
	}

	report := cliapp.MutationReport{
		Result: []string{message},
		Changes: []string{
			fmt.Sprintf("Test results deleted: %d", resp.TestResultsDeleted),
			fmt.Sprintf("Test injections deleted: %d", resp.TestInjectionsDeleted),
		},
		NextCommand: []string{
			fmt.Sprintf("%s statistics", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
