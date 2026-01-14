// Package redeploy provides the convenience redeploy workflow command.
package redeploy

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/deployment"
)

// Run executes the redeploy workflow: create/update → execute → report.
func Run(client *deployment.Client, args []string) error {
	fs := flag.NewFlagSet("redeploy", flag.ContinueOnError)
	name := fs.String("name", "", "Optional deployment name")
	preflight := fs.Bool("preflight", false, "Run VPS preflight checks")
	forceBuild := fs.Bool("force-bundle", false, "Force rebuild of bundle")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return printUsage()
	}

	manifestPath := fs.Arg(0)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	// Step 1: Create or update the deployment
	if !*jsonOutput {
		fmt.Println("Creating/updating deployment...")
	}

	createReq := deployment.CreateRequest{
		Name:     *name,
		Manifest: json.RawMessage(manifestBytes),
	}

	createBody, createResp, err := client.Create(createReq)
	if err != nil {
		return fmt.Errorf("create deployment: %w", err)
	}

	dep := createResp.Deployment
	if !*jsonOutput {
		action := "Created"
		if createResp.Updated {
			action = "Updated"
		}
		fmt.Printf("%s deployment: %s (%s)\n", action, dep.ID, dep.Name)
	}

	// Step 2: Execute the deployment
	if !*jsonOutput {
		fmt.Println("Starting deployment execution...")
	}

	execReq := deployment.ExecuteRequest{
		RunPreflight:     *preflight,
		ForceBundleBuild: *forceBuild,
	}

	execBody, execResp, err := client.Execute(dep.ID, execReq)
	if err != nil {
		return fmt.Errorf("execute deployment: %w", err)
	}

	if !*jsonOutput {
		fmt.Printf("Execution started (run_id: %s)\n", execResp.RunID)
	}

	// If JSON output, return a combined response
	if *jsonOutput {
		combined := map[string]interface{}{
			"create":    json.RawMessage(createBody),
			"execute":   json.RawMessage(execBody),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		out, _ := json.MarshalIndent(combined, "", "  ")
		cliutil.PrintJSON(out)
		return nil
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
  --json              Output raw JSON

Examples:
  scenario-to-cloud redeploy cloud-manifest.json
  scenario-to-cloud redeploy cloud-manifest.json --preflight
  scenario-to-cloud redeploy cloud-manifest.json --name "Production Deploy"`)
	return nil
}
