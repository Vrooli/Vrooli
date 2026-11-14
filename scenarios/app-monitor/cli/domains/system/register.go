package system

import (
	"fmt"
	"os"

	"app-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `system` subcommand group for host-level status and metrics.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "system",
		Description: "Inspect host system status and metrics",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "status", Description: "System status overview", Run: func(args []string) error { return runMap(core, args, "system status", "/system/status") }},
			{Name: "metrics", Description: "System metrics snapshot", Run: func(args []string) error { return runMap(core, args, "system metrics", "/system/metrics") }},
		},
	}
}

func runMap(core *cliapp.ScenarioApp, args []string, label, path string) error {
	fs := support.NewFlagSet(label)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{label},
		ResultsHeading: "Fields",
		Results:        support.MapRows(data),
		RetrievalHints: []string{fmt.Sprintf("%s %s --json", support.CLIName, label)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
