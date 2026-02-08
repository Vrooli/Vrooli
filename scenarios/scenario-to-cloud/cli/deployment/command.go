package deployment

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/internal/flagutil"
	internalmanifest "scenario-to-cloud/cli/internal/manifest"
	"scenario-to-cloud/cli/internal/streaming"
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
	case "resolve":
		return runResolve(client, args[1:])
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
	case "health":
		return runHealth(client, args[1:])
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
  resolve                   Resolve deployment ID by manifest or selector (read-only)
  list                      List all deployments
  get <id>                  Get deployment details
  delete <id>               Delete a deployment
  execute <id>              Execute the deployment pipeline
  start <id>                Start/resume a stopped deployment
  stop <id>                 Stop a running deployment
  history <id>              Show deployment history
  health                    Unified health check by deployment ID or selector

Run 'scenario-to-cloud deployment <command> -h' for command-specific options.`)
	return nil
}

func runPlan(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment plan", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
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
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
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
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
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
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
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

func runResolve(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment resolve", flag.ContinueOnError)
	selectorArgs := registerSelectorFlags(fs)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	selectorFlagsUsed := selectorArgs.anySet()
	if fs.NArg() > 1 || (fs.NArg() == 1 && selectorFlagsUsed) || (fs.NArg() == 0 && !selectorFlagsUsed) {
		return fmt.Errorf("usage: scenario-to-cloud deployment resolve <manifest.json> OR scenario-to-cloud deployment resolve [--host <host> | --domain <domain> | --target <domain-or-host>] [--scenario <id>]")
	}

	var (
		selector ManifestSelector
		err      error
	)
	if fs.NArg() == 1 {
		selector, err = ReadSelectorFromManifest(fs.Arg(0))
		if err != nil {
			return err
		}
	} else {
		selector, err = selectorArgs.toSelector()
		if err != nil {
			return err
		}
	}

	dep, err := ResolveLatestBySelector(client, selector)
	if err != nil {
		return err
	}

	response := struct {
		Found      bool               `json:"found"`
		Selector   ManifestSelector   `json:"selector"`
		Deployment *DeploymentSummary `json:"deployment,omitempty"`
		Timestamp  string             `json:"timestamp"`
	}{
		Found:      dep != nil,
		Selector:   selector,
		Deployment: dep,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	if *jsonOutput {
		body, _ := json.MarshalIndent(response, "", "  ")
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Selector: scenario=%s host=%s", selector.ScenarioID, displayOrNA(selector.Host))
	if selector.Domain != "" {
		fmt.Printf(" domain=%s", selector.Domain)
	}
	if selector.Target != "" {
		fmt.Printf(" target=%s", selector.Target)
	}
	fmt.Println()

	if dep == nil {
		fmt.Println("No existing deployment matches this selector.")
		if fs.NArg() == 1 {
			fmt.Printf("Next step: scenario-to-cloud redeploy %s --if-needed --preflight --wait\n", fs.Arg(0))
		} else {
			fmt.Println("Next step: create/validate a manifest, then run:")
			fmt.Println("  scenario-to-cloud redeploy <manifest.json> --if-needed --preflight --wait")
		}
		return nil
	}

	fmt.Printf("Resolved deployment: %s\n", dep.ID)
	fmt.Printf("  Name:          %s\n", dep.Name)
	fmt.Printf("  Status:        %s\n", dep.Status)
	fmt.Printf("  Domain:        %s\n", displayValue(dep.Domain))
	fmt.Printf("  Created:       %s\n", dep.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Printf("  Last deployed: %s\n", formatOptionalTime(dep.LastDeployedAt))
	return nil
}

func runDelete(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment delete", flag.ContinueOnError)
	stop := fs.Bool("stop", false, "Stop the deployment on VPS before deleting")
	cleanup := fs.Bool("cleanup", false, "Clean up bundle files")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
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
	wait := fs.Bool("wait", false, "Wait for completion and print stage durations")
	stream := fs.Bool("stream", true, "Stream progress updates (enabled by default)")
	noStream := fs.Bool("no-stream", false, "Disable streaming, return immediately after starting")
	jsonOutput := fs.Bool("json", false, "Output raw JSON (implies --no-stream)")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment execute <id> [--preflight] [--force-bundle] [--wait|--stream|--no-stream]")
	}

	deploymentID := fs.Arg(0)
	if *wait && *jsonOutput {
		return fmt.Errorf("--wait cannot be combined with --json")
	}

	// JSON output implies no streaming
	useStreaming := *stream && !*noStream && !*jsonOutput && !*wait

	req := ExecuteRequest{
		RunPreflight:     *preflight,
		ForceBundleBuild: *forceBuild,
	}

	// Start the execution
	body, resp, err := client.Execute(deploymentID, req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Deployment execution started (Run ID: %s)\n", resp.RunID)
	if *wait {
		return WaitForDeploymentCompletion(client, deploymentID)
	}

	if !useStreaming {
		fmt.Println("\nUse 'deployment get <id>' to check status.")
		return nil
	}

	// Set up context with signal handling for graceful cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, stopping...")
		cancel()
	}()

	// Stream progress with animated display
	fmt.Println()
	var lastStep string
	startTime := time.Now()

	err = client.StreamProgress(ctx, deploymentID, func(event streaming.ProgressEvent) error {
		// Clear previous line and print new status
		if lastStep != event.Step {
			if lastStep != "" {
				fmt.Printf("\r\033[K")
			}
			lastStep = event.Step
		}

		elapsed := time.Since(startTime).Round(time.Second)
		progressBar := renderProgressBar(event.Percent, 30)

		fmt.Printf("\r\033[K%s %5.1f%% | %s | %s",
			progressBar,
			event.Percent,
			event.Step,
			elapsed)

		if event.Message != "" && event.Message != event.Step {
			fmt.Printf(" - %s", event.Message)
		}

		return nil
	})

	fmt.Println() // New line after progress

	if err != nil {
		if streaming.IsSuccess(err) {
			elapsed := time.Since(startTime).Round(time.Second)
			fmt.Printf("\nDeployment completed successfully in %s\n", elapsed)
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("streaming cancelled: %w", ctx.Err())
		}
		// Streaming failed, fall back to showing final status
		fmt.Printf("\nStreaming interrupted: %v\n", err)
		fmt.Println("Checking final status...")
	}

	// Get final deployment status
	_, getResp, err := client.Get(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get final status: %w", err)
	}

	d := getResp.Deployment
	fmt.Printf("\nFinal Status: %s\n", d.Status)
	if d.ErrorMessage != nil {
		fmt.Printf("Error: %s\n", *d.ErrorMessage)
	}

	return nil
}

// renderProgressBar creates an ASCII progress bar.
func renderProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat(" ", width-filled) + "]"
}

func runStart(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment start", flag.ContinueOnError)
	wait := fs.Bool("wait", false, "Wait for completion and print stage durations")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment start <id> [--wait]")
	}
	if *wait && *jsonOutput {
		return fmt.Errorf("--wait cannot be combined with --json")
	}

	deploymentID := fs.Arg(0)
	body, resp, err := client.Start(deploymentID, ExecuteRequest{})
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
	if *wait {
		return WaitForDeploymentCompletion(client, deploymentID)
	}
	return nil
}

func runStop(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment stop", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
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
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
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

func displayValue(v string) string {
	if strings.TrimSpace(v) == "" {
		return "n/a"
	}
	return v
}

// WaitForDeploymentCompletion polls deployment status until it reaches a terminal state.
// It prints each stage transition and the stage duration.
func WaitForDeploymentCompletion(client *Client, deploymentID string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, stopping...")
		cancel()
	}()

	fmt.Println("Waiting for deployment to complete...")

	startedAt := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	currentStage := ""
	stageStartedAt := time.Time{}
	stageInitialized := false

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait cancelled: %w", ctx.Err())
		default:
		}

		_, getResp, err := client.Get(deploymentID)
		if err != nil {
			return fmt.Errorf("failed to poll deployment status: %w", err)
		}
		d := getResp.Deployment

		stage := stageFromDeployment(d)
		now := time.Now()

		if !stageInitialized {
			stageInitialized = true
			currentStage = stage
			stageStartedAt = now
			fmt.Printf("▶ Stage started: %s\n", currentStage)
		} else if stage != currentStage {
			fmt.Printf("✓ Stage complete: %s (%s)\n", currentStage, now.Sub(stageStartedAt).Round(time.Second))
			currentStage = stage
			stageStartedAt = now
			fmt.Printf("▶ Stage started: %s\n", currentStage)
		}

		if isTerminalDeploymentStatus(d.Status) {
			fmt.Printf("✓ Stage complete: %s (%s)\n", currentStage, now.Sub(stageStartedAt).Round(time.Second))
			total := now.Sub(startedAt).Round(time.Second)
			fmt.Printf("\nFinal Status: %s\n", d.Status)
			fmt.Printf("Total Duration: %s\n", total)
			if d.ErrorMessage != nil && strings.TrimSpace(*d.ErrorMessage) != "" {
				fmt.Printf("Error: %s\n", *d.ErrorMessage)
			}
			if d.Status == StatusDeployed {
				return nil
			}
			return fmt.Errorf("deployment ended with status %s", d.Status)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait cancelled: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func stageFromDeployment(d *Deployment) string {
	if d != nil && d.ProgressStep != nil && strings.TrimSpace(*d.ProgressStep) != "" {
		return strings.TrimSpace(*d.ProgressStep)
	}
	if d == nil {
		return "unknown"
	}
	return string(d.Status)
}

func isTerminalDeploymentStatus(status DeploymentStatus) bool {
	switch status {
	case StatusDeployed, StatusFailed, StatusStopped, StatusSetupComplete:
		return true
	default:
		return false
	}
}
