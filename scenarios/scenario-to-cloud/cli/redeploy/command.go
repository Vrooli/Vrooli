// Package redeploy provides the convenience redeploy workflow command.
package redeploy

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/internal/flagutil"
)

// Run executes the redeploy workflow: create/update → execute → report.
func Run(client *deployment.Client, args []string) error {
	fs := flag.NewFlagSet("redeploy", flag.ContinueOnError)
	name := fs.String("name", "", "Optional deployment name")
	preflight := fs.Bool("preflight", false, "Run VPS preflight checks")
	forceBuild := fs.Bool("force-bundle", false, "Force rebuild of bundle")
	ifNeeded := fs.Bool("if-needed", false, "Deploy only when missing, unhealthy, or outdated")
	wait := fs.Bool("wait", false, "Wait for completion and print stage durations")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *wait && *jsonOutput {
		return fmt.Errorf("--wait cannot be combined with --json")
	}
	if *ifNeeded && *jsonOutput {
		return fmt.Errorf("--if-needed cannot be combined with --json")
	}

	if fs.NArg() != 1 {
		return printUsage()
	}

	manifestPath := fs.Arg(0)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	if *ifNeeded {
		return runIfNeeded(client, manifestPath, manifestBytes, *name, *preflight, *forceBuild, *wait)
	}

	return createAndExecute(client, manifestBytes, *name, *preflight, *forceBuild, *wait, *jsonOutput)
}

func runIfNeeded(client *deployment.Client, manifestPath string, manifestBytes []byte, name string, preflight bool, forceBuild bool, wait bool) error {
	selector, err := deployment.ReadSelectorFromManifest(manifestPath)
	if err != nil {
		return err
	}

	existing, err := deployment.ResolveLatestBySelector(client, selector)
	if err != nil {
		return fmt.Errorf("resolve deployment: %w", err)
	}

	if existing == nil {
		fmt.Println("No matching deployment found for manifest target; creating a new deployment.")
		return createAndExecute(client, manifestBytes, name, preflight, forceBuild, wait, false)
	}

	fmt.Printf("Found existing deployment: %s (%s)\n", existing.ID, existing.Status)
	_, health, err := client.Health(existing.ID)
	if err != nil {
		return fmt.Errorf("deployment health: %w", err)
	}

	healthState := strings.ToLower(strings.TrimSpace(health.Health))
	freshnessState := ""
	if health.Freshness != nil {
		freshnessState = strings.ToLower(strings.TrimSpace(health.Freshness.Status))
	}

	if healthState == "healthy" && freshnessState == "current" && !forceBuild {
		fmt.Println("Deployment is healthy and current. No redeploy needed.")
		fmt.Printf("  Verify: scenario-to-cloud deployment health %s\n", existing.ID)
		return nil
	}

	if healthState == "stopped" && freshnessState == "current" && !forceBuild {
		fmt.Println("Deployment is stopped but current. Starting deployment...")
		_, startResp, err := client.Start(existing.ID, deployment.ExecuteRequest{})
		if err != nil {
			return fmt.Errorf("start deployment: %w", err)
		}
		fmt.Printf("Start initiated (run_id: %s)\n", startResp.RunID)
		if wait {
			return deployment.WaitForDeploymentCompletion(client, existing.ID)
		}
		fmt.Printf("  Check status:  scenario-to-cloud deployment get %s\n", existing.ID)
		return nil
	}

	derivedForceBuild := forceBuild || freshnessState == "outdated"
	if freshnessState == "outdated" && !forceBuild {
		fmt.Println("Deployment is outdated relative to local scenario state; forcing bundle rebuild.")
	}

	fmt.Println("Redeploy required; updating deployment record and executing...")
	return createAndExecute(client, manifestBytes, name, preflight, derivedForceBuild, wait, false)
}

func createAndExecute(client *deployment.Client, manifestBytes []byte, name string, preflight bool, forceBuild bool, wait bool, jsonOutput bool) error {
	// Step 1: Create or update the deployment
	if !jsonOutput {
		fmt.Println("Creating/updating deployment...")
	}

	createReq := deployment.CreateRequest{
		Name:     name,
		Manifest: json.RawMessage(manifestBytes),
	}

	createBody, createResp, err := client.Create(createReq)
	if err != nil {
		return fmt.Errorf("create deployment: %w", err)
	}

	dep := createResp.Deployment
	if !jsonOutput {
		action := "Created"
		if createResp.Updated {
			action = "Updated"
		}
		fmt.Printf("%s deployment: %s (%s)\n", action, dep.ID, dep.Name)
	}

	// Step 2: Execute the deployment
	if !jsonOutput {
		fmt.Println("Starting deployment execution...")
	}

	execReq := deployment.ExecuteRequest{
		RunPreflight:     preflight,
		ForceBundleBuild: forceBuild,
	}

	execBody, execResp, err := client.Execute(dep.ID, execReq)
	if err != nil {
		return fmt.Errorf("execute deployment: %w", err)
	}

	if !jsonOutput {
		fmt.Printf("Execution started (run_id: %s)\n", execResp.RunID)
	}

	// If JSON output, return a combined response
	if jsonOutput {
		combined := map[string]interface{}{
			"create":    json.RawMessage(createBody),
			"execute":   json.RawMessage(execBody),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		out, _ := json.MarshalIndent(combined, "", "  ")
		cliutil.PrintJSON(out)
		return nil
	}

	if wait {
		return deployment.WaitForDeploymentCompletion(client, dep.ID)
	}

	// Print helpful next steps
	fmt.Println()
	fmt.Println("Deployment is running in the background.")
	fmt.Printf("  Check status:  scenario-to-cloud deployment get %s\n", dep.ID)
	fmt.Printf("  View history:  scenario-to-cloud deployment history %s\n", dep.ID)
	fmt.Printf("  Stop:          scenario-to-cloud deployment stop %s\n", dep.ID)

	return nil
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud redeploy <manifest.json> [options]

Convenience command that creates/updates a deployment and executes it.

	Options:
	  --name <name>       Optional deployment name
	  --preflight         Run VPS preflight checks before deployment
	  --force-bundle      Force rebuild of bundle even if one exists
	  --if-needed         Only execute when deployment is missing, unhealthy, stopped, or outdated
	  --wait              Wait for completion and print stage durations
	  --json              Output raw JSON

	Examples:
	  scenario-to-cloud redeploy cloud-manifest.json
	  scenario-to-cloud redeploy cloud-manifest.json --preflight
	  scenario-to-cloud redeploy cloud-manifest.json --if-needed --preflight --wait
	  scenario-to-cloud redeploy cloud-manifest.json --preflight --wait
	  scenario-to-cloud redeploy cloud-manifest.json --name "Production Deploy"`)
	return nil
}
