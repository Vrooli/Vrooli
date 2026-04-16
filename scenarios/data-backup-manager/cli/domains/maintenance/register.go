package maintenance

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"data-backup-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `maintenance` for the maintenance orchestrator integration.
// Endpoints are distinct from the root /health probe (which cli-core handles),
// so this domain does not duplicate the built-in status command.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "maintenance",
		Description: "Inspect and invoke the maintenance orchestrator",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "status", Description: "Show maintenance orchestrator status", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "task", Description: "Dispatch a maintenance task (payload from --body-file)", Run: func(args []string) error { return runTask(core, args) }},
			{Name: "toggle", Description: "Enable or disable the maintenance agent", Run: func(args []string) error { return runToggle(core, args) }},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("maintenance status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/maintenance/status", nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Maintenance orchestrator status"},
		ResultsHeading: "Details",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{
			fmt.Sprintf("%s maintenance task --body-file ./task.json", support.CLIName),
			fmt.Sprintf("%s maintenance toggle --enabled true", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runTask(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("maintenance task")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the task request (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/maintenance/task", nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	taskID, _ := resp["task_id"].(string)
	result := "Dispatched maintenance task"
	if taskID != "" {
		result = fmt.Sprintf("Dispatched maintenance task %s", taskID)
	}

	mutation := cliapp.MutationReport{
		Result:      []string{result},
		Changes:     support.MapRows(resp),
		NextCommand: []string{fmt.Sprintf("%s maintenance status", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, mutation)
	}
	return cliapp.RenderMutationReport(os.Stdout, mutation)
}

func runToggle(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("maintenance toggle")
	enabled := fs.String("enabled", "", "Enabled state (true|false) (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*enabled) == "" {
		return fmt.Errorf("--enabled is required (true|false)")
	}
	b, err := strconv.ParseBool(*enabled)
	if err != nil {
		return fmt.Errorf("--enabled must be true or false: %w", err)
	}

	body, err := core.Request("POST", "/maintenance/agent/toggle", nil, map[string]interface{}{"enabled": b})
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	verb := "disabled"
	if b {
		verb = "enabled"
	}
	mutation := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Maintenance agent %s", verb)},
		Changes:     support.MapRows(resp),
		NextCommand: []string{fmt.Sprintf("%s maintenance status", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, mutation)
	}
	return cliapp.RenderMutationReport(os.Stdout, mutation)
}
