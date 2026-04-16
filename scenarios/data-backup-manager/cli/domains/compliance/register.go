package compliance

import (
	"fmt"
	"os"

	"data-backup-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `compliance` for compliance report/scan/fix operations.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "compliance",
		Description: "Inspect and remediate backup compliance issues",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "report", Description: "Show the latest compliance report", Run: func(args []string) error { return runReport(core, args) }},
			{Name: "scan", Description: "Trigger a compliance scan", Run: func(args []string) error { return runScan(core, args) }},
			{Name: "fix", Description: "Apply the automated fix for a compliance issue", Run: func(args []string) error { return runFix(core, args) }},
		},
	}
}

func runReport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("compliance report")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/compliance/report", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Compliance report"},
		ResultsHeading: "Details",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{
			fmt.Sprintf("%s compliance scan", support.CLIName),
			fmt.Sprintf("%s compliance fix <issue-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runScan(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("compliance scan")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/compliance/scan", nil, nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	scanID, _ := payload["scan_id"].(string)
	result := "Started compliance scan"
	if scanID != "" {
		result = fmt.Sprintf("Started compliance scan %s", scanID)
	}

	mutation := cliapp.MutationReport{
		Result:      []string{result},
		Changes:     support.MapRows(payload),
		NextCommand: []string{fmt.Sprintf("%s compliance report", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, mutation)
	}
	return cliapp.RenderMutationReport(os.Stdout, mutation)
}

func runFix(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("compliance fix")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: compliance fix <issue-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/compliance/issue/"+id+"/fix", nil, nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	mutation := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Applied fix for issue %s", id)},
		Changes:     support.MapRows(payload),
		NextCommand: []string{fmt.Sprintf("%s compliance report", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, mutation)
	}
	return cliapp.RenderMutationReport(os.Stdout, mutation)
}
