package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func (a *App) cmdDeclarations(args []string) error {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "reconcile-scenario":
		return a.declarationsReconcile(args[1:], "/api/v1/declarations/reconcile-scenario", false)
	case "plan":
		return a.declarationsReconcile(args[1:], "/api/v1/declarations/plan", true)
	case "help", "-h", "--help":
		return nil
	default:
		return fmt.Errorf("unknown declarations subcommand: %s\n\nRun 'agent-manager declarations help' for usage", args[0])
	}
}

func (a *App) declarationsReconcile(args []string, path string, forceDry bool) error {
	fs := flag.NewFlagSet("declarations reconcile", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	scenario := fs.String("scenario", "", "Owning scenario slug")
	dry := fs.Bool("dry-run", false, "Validate without writes")
	validateOnly := fs.Bool("validate-only", false, "Only validate sources")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" {
		return fmt.Errorf("--scenario is required")
	}
	body, resp, err := a.services.Declarations.Reconcile(path, &apipb.ReconcileScenarioDeclarationsRequest{
		Scenario:     strings.TrimSpace(*scenario),
		DryRun:       *dry || forceDry,
		ValidateOnly: *validateOnly,
	})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Scenario: %s\n", resp.Scenario)
	fmt.Printf("Profiles  created=%d updated=%d unchanged=%d skipped=%d conflicted=%d failed=%d\n",
		resp.ProfilesCreated, resp.ProfilesUpdated, resp.ProfilesUnchanged, resp.ProfilesSkipped, resp.ProfilesConflicted, resp.ProfilesFailed)
	fmt.Printf("Workflows created=%d activated=%d unchanged=%d skipped=%d failed=%d\n",
		resp.WorkflowsCreated, resp.WorkflowsActivated, resp.WorkflowsUnchanged, resp.WorkflowsSkipped, resp.WorkflowsFailed)
	for _, item := range resp.ProfileResults {
		fmt.Printf("- profile %s %s (%s)\n", item.ProfileKey, item.Status.String(), item.SourcePath)
		if item.Message != "" {
			fmt.Printf("  %s\n", item.Message)
		}
	}
	for _, item := range resp.WorkflowResults {
		fmt.Printf("- workflow %s@%s %s (%s) %s\n", item.WorkflowKey, item.Version, item.Status.String(), item.SourcePath, item.Message)
	}
	if resp.ProfilesFailed > 0 || resp.WorkflowsFailed > 0 {
		return fmt.Errorf("declaration reconciliation failed validation")
	}
	return nil
}
