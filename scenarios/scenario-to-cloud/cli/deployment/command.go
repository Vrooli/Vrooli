package deployment

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/vrooli/cli-core/cliutil"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// Run executes deployment subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "plan":
		return runPlan(client, args[1:])
	case "create":
		return runCreate(client, args[1:])
	case "list":
		return runList(client, args[1:])
	case "get":
		return runGet(client, args[1:])
	case "delete":
		return runDelete(client, args[1:])
	case "execute":
		return runExecute(client, args[1:])
	case "start":
		return runStart(client, args[1:])
	case "stop":
		return runStop(client, args[1:])
	case "history":
		return runHistory(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud deployment help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud deployment <command> [arguments]

Commands:
  plan <manifest.json>      Generate a deployment plan from a cloud manifest
  create <manifest.json>    Create a deployment from a manifest
  list                      List all deployments
  get <id>                  Get deployment details
  delete <id>               Delete a deployment
  execute <id>              Execute the deployment pipeline
  start <id>                Start/resume a stopped deployment
  stop <id>                 Stop a running deployment
  history <id>              Show deployment history

Run 'scenario-to-cloud deployment <command> -h' for command-specific options.`)
	return nil
}

func runPlan(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment plan", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment plan <manifest.json>")
	}

	manifest, err := internalmanifest.ReadJSONFile(fs.Arg(0))
	if err != nil {
		return err
	}

	body, resp, err := client.Plan(manifest)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print the plan
	fmt.Printf("Deployment Plan (%s)\n", resp.Timestamp)
	fmt.Println(strings.Repeat("-", 60))
	for i, step := range resp.Plan {
		fmt.Printf("%d. %s\n", i+1, step.Title)
		if step.Description != "" {
			fmt.Printf("   %s\n", step.Description)
		}
	}
	return nil
}

func runCreate(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment create", flag.ContinueOnError)
	name := fs.String("name", "", "Optional deployment name")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment create <manifest.json> [--name <name>]")
	}

	manifestPath := fs.Arg(0)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	req := CreateRequest{
		Name:     *name,
		Manifest: json.RawMessage(manifestBytes),
	}

	body, resp, err := client.Create(req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	action := "Created"
	if resp.Updated {
		action = "Updated"
	}
	fmt.Printf("%s deployment: %s\n", action, resp.Deployment.ID)
	fmt.Printf("  Name:     %s\n", resp.Deployment.Name)
	fmt.Printf("  Scenario: %s\n", resp.Deployment.ScenarioID)
	fmt.Printf("  Status:   %s\n", resp.Deployment.Status)
	return nil
}

func runList(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment list", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by status (pending, deploying, deployed, failed, stopped)")
	scenario := fs.String("scenario", "", "Filter by scenario ID")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := ListOptions{
		Status:     *status,
		ScenarioID: *scenario,
	}

	body, resp, err := client.List(opts)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(resp.Deployments) == 0 {
		fmt.Println("No deployments found.")
		return nil
	}

	// Pretty print as table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSCENARIO\tSTATUS\tDOMAIN\tCREATED")
	for _, d := range resp.Deployments {
		created := d.CreatedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncate(d.ID, 12),
			truncate(d.Name, 30),
			truncate(d.ScenarioID, 20),
			d.Status,
			truncate(d.Domain, 25),
			created,
		)
	}
	w.Flush()
	return nil
}

func runGet(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment get", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment get <id>")
	}

	body, resp, err := client.Get(fs.Arg(0))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	d := resp.Deployment
	fmt.Printf("Deployment: %s\n", d.ID)
	fmt.Printf("  Name:       %s\n", d.Name)
	fmt.Printf("  Scenario:   %s\n", d.ScenarioID)
	fmt.Printf("  Status:     %s\n", d.Status)
	if d.ProgressStep != nil {
		fmt.Printf("  Progress:   %s (%.0f%%)\n", *d.ProgressStep, d.ProgressPercent)
	}
	if d.ErrorMessage != nil {
		fmt.Printf("  Error:      %s\n", *d.ErrorMessage)
	}
	fmt.Printf("  Created:    %s\n", d.CreatedAt.Format(time.RFC3339))
	if d.LastDeployedAt != nil {
		fmt.Printf("  Deployed:   %s\n", d.LastDeployedAt.Format(time.RFC3339))
	}
	return nil
}

func runDelete(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment delete", flag.ContinueOnError)
	stop := fs.Bool("stop", false, "Stop the deployment on VPS before deleting")
	cleanup := fs.Bool("cleanup", false, "Clean up bundle files")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment delete <id> [--stop] [--cleanup]")
	}

	opts := DeleteOptions{
		Stop:    *stop,
		Cleanup: *cleanup,
	}

	body, _, err := client.Delete(fs.Arg(0), opts)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Deployment %s deleted.\n", fs.Arg(0))
	return nil
}

func runExecute(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment execute", flag.ContinueOnError)
	preflight := fs.Bool("preflight", false, "Run VPS preflight checks before deployment")
	forceBuild := fs.Bool("force-bundle", false, "Force rebuild of bundle even if one exists")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment execute <id> [--preflight] [--force-bundle]")
	}

	req := ExecuteRequest{
		RunPreflight:     *preflight,
		ForceBundleBuild: *forceBuild,
	}

	body, resp, err := client.Execute(fs.Arg(0), req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Deployment execution started.\n")
	fmt.Printf("  Run ID:  %s\n", resp.RunID)
	fmt.Printf("  Message: %s\n", resp.Message)
	fmt.Println("\nUse 'deployment get <id>' to check status.")
	return nil
}

func runStart(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment start", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment start <id>")
	}

	body, resp, err := client.Start(fs.Arg(0), ExecuteRequest{})
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Deployment start initiated.\n")
	fmt.Printf("  Run ID:  %s\n", resp.RunID)
	fmt.Printf("  Message: %s\n", resp.Message)
	return nil
}

func runStop(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment stop", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment stop <id>")
	}

	body, resp, err := client.Stop(fs.Arg(0))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("Deployment stopped successfully.\n")
	} else {
		fmt.Printf("Deployment stop failed: %s\n", resp.Error)
	}
	return nil
}

func runHistory(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment history", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment history <id>")
	}

	body, resp, err := client.History(fs.Arg(0))
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(resp.Events) == 0 {
		fmt.Println("No history events found.")
		return nil
	}

	fmt.Printf("History for deployment %s:\n", resp.DeploymentID)
	fmt.Println(strings.Repeat("-", 60))
	for _, e := range resp.Events {
		status := ""
		if e.Success != nil {
			if *e.Success {
				status = " [OK]"
			} else {
				status = " [FAILED]"
			}
		}
		fmt.Printf("%s  %s%s\n", e.Timestamp.Format("2006-01-02 15:04:05"), e.Message, status)
		if e.Details != "" {
			fmt.Printf("           %s\n", e.Details)
		}
	}
	return nil
}

// truncate shortens a string to max length with ellipsis.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
