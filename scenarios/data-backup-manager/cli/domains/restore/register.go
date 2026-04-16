package restore

import (
	"fmt"
	"os"
	"strings"

	"data-backup-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes the `restore` subcommand group for creating and inspecting
// restore operations.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "restore",
		Description: "Create and inspect restore operations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a restore operation", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "status", Description: "Show restore operation status by id", Run: func(args []string) error { return runStatus(core, args) }},
		},
	}
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("restore create")
	backupID := fs.String("backup-id", "", "Backup job ID to restore from")
	restorePoint := fs.String("restore-point", "", "Restore point ID to restore from")
	targets := fs.String("targets", "", "Comma-separated targets to restore")
	verify := fs.Bool("verify", false, "Verify backup integrity before restore")
	destination := fs.String("destination", "", "Custom restore destination path")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if strings.TrimSpace(*backupID) == "" && strings.TrimSpace(*restorePoint) == "" {
		return fmt.Errorf("either --backup-id or --restore-point is required")
	}

	targetList := support.SplitCSV(*targets)
	payload := map[string]interface{}{
		"backup_job_id":         *backupID,
		"restore_point_id":      *restorePoint,
		"targets":               targetList,
		"verify_before_restore": *verify,
		"destination":           *destination,
	}

	body, err := core.Request("POST", "/restore/create", nil, payload)
	if err != nil {
		return err
	}
	var resp support.RestoreCreateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Restore ID: %s", resp.RestoreID),
	}
	if resp.Status != "" {
		changes = append(changes, fmt.Sprintf("Status: %s", resp.Status))
	}
	if resp.EstimatedDuration != "" {
		changes = append(changes, fmt.Sprintf("Estimated duration: %s", resp.EstimatedDuration))
	}
	if len(targetList) > 0 {
		changes = append(changes, fmt.Sprintf("Targets: %s", strings.Join(targetList, ",")))
	}
	if *verify {
		changes = append(changes, "Verify before restore: true")
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created restore operation %s", resp.RestoreID)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s restore status %s", support.CLIName, resp.RestoreID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("restore status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: restore status <restore-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/restore/status/"+id, nil)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Restore %s", id)},
		ResultsHeading: "Details",
		Results:        support.MapRows(payload),
		RetrievalHints: []string{fmt.Sprintf("%s backup list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
