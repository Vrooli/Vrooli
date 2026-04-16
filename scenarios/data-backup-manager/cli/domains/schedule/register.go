package schedule

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"data-backup-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `schedule` for backup schedule CRUD and enable/disable.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "schedule",
		Description: "Manage backup schedules",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List backup schedules", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a backup schedule", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update a backup schedule", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "enable", Description: "Enable a backup schedule", Run: func(args []string) error { return runToggle(core, args, true, "enable") }},
			{Name: "disable", Description: "Disable a backup schedule", Run: func(args []string) error { return runToggle(core, args, false, "disable") }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a backup schedule", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schedule list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/schedules", nil)
	if err != nil {
		return err
	}
	var resp support.ScheduleListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.Schedules))
	for _, s := range resp.Schedules {
		rows = append(rows, fmt.Sprintf("%s | %s | cron=%q | type=%s | enabled=%t | next=%s",
			support.ShortID(s.ID), s.Name, s.CronExpression, s.BackupType, s.Enabled,
			support.PtrTimeString(s.NextRun)))
	}
	if len(rows) == 0 {
		rows = []string{"(no schedules)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Schedules: %d", resp.Total)},
		ResultsHeading: "Schedules",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s schedule enable <id>", support.CLIName),
			fmt.Sprintf("%s schedule disable <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schedule create")
	name := fs.String("name", "", "Schedule name (required)")
	cron := fs.String("cron", "", "Cron expression (required)")
	targets := fs.String("targets", "", "Comma-separated targets (required)")
	retention := fs.Int("retention", 7, "Retention period in days")
	backupType := fs.String("type", "full", "Backup type: full|incremental|differential")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*cron) == "" || strings.TrimSpace(*targets) == "" {
		return fmt.Errorf("--name, --cron, and --targets are required")
	}
	targetList := support.SplitCSV(*targets)
	if len(targetList) == 0 {
		return fmt.Errorf("--targets must contain at least one value")
	}

	payload := map[string]interface{}{
		"name":            *name,
		"cron_expression": *cron,
		"backup_type":     *backupType,
		"targets":         targetList,
		"retention_days":  *retention,
	}

	body, err := core.Request("POST", "/schedules", nil, payload)
	if err != nil {
		return err
	}
	var created map[string]interface{}
	if err := support.Decode(body, &created); err != nil {
		return err
	}
	id, _ := created["id"].(string)

	changes := []string{
		fmt.Sprintf("ID: %s", id),
		fmt.Sprintf("Name: %s", *name),
		fmt.Sprintf("Cron: %s", *cron),
		fmt.Sprintf("Type: %s", *backupType),
		fmt.Sprintf("Targets: %s", strings.Join(targetList, ",")),
		fmt.Sprintf("Retention days: %d", *retention),
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created schedule %s", *name)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s schedule list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schedule update")
	name := fs.String("name", "", "New schedule name")
	cron := fs.String("cron", "", "New cron expression")
	targets := fs.String("targets", "", "Comma-separated targets")
	retention := fs.Int("retention", -1, "Retention period in days")
	backupType := fs.String("type", "", "Backup type: full|incremental|differential")
	enabled := fs.String("enabled", "", "Set enabled state (true|false)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full update payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: schedule update <id> [flags]")
	}
	id := fs.Arg(0)

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		updates := map[string]interface{}{}
		if strings.TrimSpace(*name) != "" {
			updates["name"] = *name
		}
		if strings.TrimSpace(*cron) != "" {
			updates["cron_expression"] = *cron
		}
		if strings.TrimSpace(*targets) != "" {
			updates["targets"] = support.SplitCSV(*targets)
		}
		if *retention >= 0 {
			updates["retention_days"] = *retention
		}
		if strings.TrimSpace(*backupType) != "" {
			updates["backup_type"] = *backupType
		}
		if strings.TrimSpace(*enabled) != "" {
			b, err := strconv.ParseBool(*enabled)
			if err != nil {
				return fmt.Errorf("--enabled must be true or false: %w", err)
			}
			updates["enabled"] = b
		}
		if len(updates) == 0 {
			return fmt.Errorf("no fields to update — provide flags or --body-file")
		}
		payload = updates
	}

	if _, err := core.Request("PUT", "/schedules/"+id, nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated schedule %s", id)},
		NextCommand: []string{fmt.Sprintf("%s schedule list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runToggle(core *cliapp.ScenarioApp, args []string, enabled bool, verb string) error {
	fs := support.NewFlagSet("schedule " + verb)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: schedule %s <id>", verb)
	}
	id := fs.Arg(0)

	payload := map[string]interface{}{"enabled": enabled}
	if _, err := core.Request("PUT", "/schedules/"+id, nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Schedule %s %sd", id, verb)},
		Changes:     []string{fmt.Sprintf("enabled: %t", enabled)},
		NextCommand: []string{fmt.Sprintf("%s schedule list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("schedule delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: schedule delete <id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/schedules/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted schedule %s", id)},
		NextCommand: []string{fmt.Sprintf("%s schedule list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
