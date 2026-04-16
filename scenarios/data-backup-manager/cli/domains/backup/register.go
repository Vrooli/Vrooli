package backup

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"data-backup-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `backup` subcommand group covering the scenario-specific
// backup lifecycle: create/list/status/verify. The root `/health` probe is
// handled by cli-core's built-in `status` command; `backup status` wraps the
// richer /api/v1/backup/status endpoint.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "backup",
		Description: "Create and inspect backup jobs",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "status", Description: "Show backup system status", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "create", Description: "Create a backup job", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "list", Aliases: []string{"ls"}, Description: "List backups", Run: func(args []string) error { return runList(core, args) }},
			{Name: "verify", Description: "Verify backup integrity by id", Run: func(args []string) error { return runVerify(core, args) }},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("backup status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/backup/status", nil)
	if err != nil {
		return err
	}
	var status support.BackupStatusResponse
	if err := support.Decode(body, &status); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("System status: %s", status.SystemStatus),
		fmt.Sprintf("Last successful backup: %s", support.PtrTimeString(status.LastSuccessfulBackup)),
		fmt.Sprintf("Storage used: %.2f GB", status.StorageUsage.UsedGB),
		fmt.Sprintf("Storage available: %.2f GB", status.StorageUsage.AvailableGB),
		fmt.Sprintf("Compression ratio: %.2f", status.StorageUsage.CompressionRatio),
		fmt.Sprintf("Active jobs: %d", len(status.ActiveJobs)),
	}
	if len(status.ResourceHealth) > 0 {
		results = append(results, "Resource health:")
		for name, rh := range status.ResourceHealth {
			msg := rh.Status
			if rh.Message != "" {
				msg = fmt.Sprintf("%s (%s)", rh.Status, rh.Message)
			}
			results = append(results, fmt.Sprintf("  %s: %s", name, msg))
		}
	}
	for _, job := range status.ActiveJobs {
		results = append(results, fmt.Sprintf("  active: %s | %s | %s | target=%s",
			support.ShortID(job.ID), job.Type, job.Status, job.Target))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Backup system: %s", status.SystemStatus)},
		ResultsHeading: "Status",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s backup list", support.CLIName),
			fmt.Sprintf("%s schedule list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("backup create")
	targets := fs.String("targets", "", "Comma-separated targets (postgres,files,scenarios,minio) (required)")
	backupType := fs.String("type", "full", "Backup type: full|incremental|differential")
	retentionDays := fs.Int("retention-days", 7, "Retention period in days")
	description := fs.String("description", "", "Human-readable backup description")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	// Accept positional target list for compatibility with the old bash CLI
	// (`backup postgres,files`).
	if strings.TrimSpace(*targets) == "" && fs.NArg() >= 1 {
		*targets = fs.Arg(0)
	}
	targetList := support.SplitCSV(*targets)
	if len(targetList) == 0 {
		return fmt.Errorf("--targets is required (e.g. --targets postgres,files)")
	}

	payload := map[string]interface{}{
		"type":           *backupType,
		"targets":        targetList,
		"retention_days": *retentionDays,
	}
	if strings.TrimSpace(*description) != "" {
		payload["description"] = *description
	}

	body, err := core.Request("POST", "/backup/create", nil, payload)
	if err != nil {
		return err
	}
	var resp support.BackupCreateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Job ID: %s", resp.JobID),
		fmt.Sprintf("Type: %s", *backupType),
		fmt.Sprintf("Targets: %s", strings.Join(targetList, ",")),
	}
	if resp.Status != "" {
		changes = append(changes, fmt.Sprintf("Status: %s", resp.Status))
	}
	if resp.EstimatedDuration != "" {
		changes = append(changes, fmt.Sprintf("Estimated duration: %s", resp.EstimatedDuration))
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created backup job %s", resp.JobID)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s backup status", support.CLIName),
			fmt.Sprintf("%s backup verify %s", support.CLIName, resp.JobID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("backup list")
	backupType := fs.String("type", "", "Filter by backup type (full|incremental|differential)")
	target := fs.String("target", "", "Filter by backup target")
	since := fs.String("since", "", "Show backups since date (YYYY-MM-DD)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"type":   *backupType,
		"target": *target,
		"since":  *since,
	})
	body, err := core.Get("/backup/list", query)
	if err != nil {
		return err
	}
	var resp support.BackupListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.Backups))
	for _, b := range resp.Backups {
		rows = append(rows, fmt.Sprintf("%s | %s | %s | %s | started=%s | size=%d",
			support.ShortID(b.ID), b.Type, b.Target, b.Status,
			support.PtrTimeString(b.StartedAt), b.SizeBytes))
	}
	if len(rows) == 0 {
		rows = []string{"(no backups returned)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Backups: %d", resp.Total)},
		ResultsHeading: "Backups",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s backup verify <backup-id>", support.CLIName),
			fmt.Sprintf("%s restore create --backup-id <backup-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runVerify(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("backup verify")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: backup verify <backup-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/backup/verify/"+id, nil, nil)
	if err != nil {
		return err
	}
	var resp support.BackupVerifyResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Backup ID: %s", resp.BackupID),
		fmt.Sprintf("Verified: %s", strconv.FormatBool(resp.Verified)),
		fmt.Sprintf("Checksum match: %s", strconv.FormatBool(resp.ChecksumMatch)),
		fmt.Sprintf("Size match: %s", strconv.FormatBool(resp.SizeMatch)),
	}
	if resp.VerifiedAt != nil {
		changes = append(changes, fmt.Sprintf("Verified at: %s", support.FormatTimeValue(*resp.VerifiedAt)))
	}
	for _, issue := range resp.Issues {
		changes = append(changes, fmt.Sprintf("Issue: %s", issue))
	}

	result := fmt.Sprintf("Verification %s for %s", verifyStatus(resp.Verified), id)
	report := cliapp.MutationReport{
		Result:      []string{result},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s backup list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func verifyStatus(ok bool) string {
	if ok {
		return "passed"
	}
	return "failed"
}
