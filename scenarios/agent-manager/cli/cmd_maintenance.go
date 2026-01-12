package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// =============================================================================
// Maintenance Command Dispatcher
// =============================================================================

func (a *App) cmdMaintenance(args []string) error {
	if len(args) == 0 {
		return a.maintenanceHelp()
	}

	switch args[0] {
	case "purge":
		return a.maintenancePurge(args[1:])
	case "help", "-h", "--help":
		return a.maintenanceHelp()
	default:
		return fmt.Errorf("unknown maintenance subcommand: %s\n\nRun 'agent-manager maintenance help' for usage", args[0])
	}
}

func (a *App) maintenanceHelp() error {
	fmt.Println(`Usage: agent-manager maintenance <subcommand> [options]

Subcommands:
  purge             Delete profiles, tasks, or runs matching a regex pattern

Options:
  --pattern         Regex pattern to match against names/IDs (required)
  --targets         Comma-separated targets: profiles,tasks,runs (required)
  --dry-run         Preview what would be deleted without actually deleting

Examples:
  agent-manager maintenance purge --pattern "^test-.*" --targets profiles,tasks,runs --dry-run
  agent-manager maintenance purge --pattern "^old-" --targets runs`)
	return nil
}

// =============================================================================
// Maintenance Purge
// =============================================================================

func (a *App) maintenancePurge(args []string) error {
	fs := flag.NewFlagSet("maintenance purge", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	pattern := fs.String("pattern", "", "Regex pattern to match (required)")
	targetsStr := fs.String("targets", "", "Comma-separated targets: profiles,tasks,runs (required)")
	dryRun := fs.Bool("dry-run", false, "Preview without deleting")
	force := fs.Bool("force", false, "Skip confirmation (for non-dry-run)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *pattern == "" {
		return fmt.Errorf("--pattern is required")
	}

	if *targetsStr == "" {
		return fmt.Errorf("--targets is required (comma-separated: profiles,tasks,runs)")
	}

	// Parse targets
	targetStrs := strings.Split(*targetsStr, ",")
	targets := make([]apipb.PurgeTarget, 0, len(targetStrs))
	for _, t := range targetStrs {
		switch strings.TrimSpace(strings.ToLower(t)) {
		case "profiles":
			targets = append(targets, apipb.PurgeTarget_PURGE_TARGET_PROFILES)
		case "tasks":
			targets = append(targets, apipb.PurgeTarget_PURGE_TARGET_TASKS)
		case "runs":
			targets = append(targets, apipb.PurgeTarget_PURGE_TARGET_RUNS)
		default:
			return fmt.Errorf("invalid target: %s (valid: profiles, tasks, runs)", t)
		}
	}

	// Confirm if not dry-run and not forced
	if !*dryRun && !*force {
		fmt.Printf("Purge items matching pattern '%s' from %s? [y/N]: ", *pattern, *targetsStr)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	req := &apipb.PurgeDataRequest{
		Pattern: *pattern,
		Targets: targets,
		DryRun:  *dryRun,
	}

	body, resp, err := a.services.Maintenance.Purge(req)
	if err != nil {
		return err
	}

	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	action := "Deleted"
	if resp.DryRun {
		action = "Would delete"
	}

	fmt.Printf("\nPurge Results (pattern: %s)\n", *pattern)
	fmt.Printf("%-10s  %s\n", "Target", "Matched → "+action)
	fmt.Printf("%-10s  %s\n", strings.Repeat("-", 10), strings.Repeat("-", 20))

	if resp.Matched != nil && resp.Deleted != nil {
		if resp.Matched.Profiles > 0 || resp.Deleted.Profiles > 0 {
			fmt.Printf("%-10s  %d → %d\n", "Profiles", resp.Matched.Profiles, resp.Deleted.Profiles)
		}
		if resp.Matched.Tasks > 0 || resp.Deleted.Tasks > 0 {
			fmt.Printf("%-10s  %d → %d\n", "Tasks", resp.Matched.Tasks, resp.Deleted.Tasks)
		}
		if resp.Matched.Runs > 0 || resp.Deleted.Runs > 0 {
			fmt.Printf("%-10s  %d → %d\n", "Runs", resp.Matched.Runs, resp.Deleted.Runs)
		}
	}

	if resp.DryRun {
		fmt.Println("\n(Dry run - no items were deleted)")
	}

	return nil
}
