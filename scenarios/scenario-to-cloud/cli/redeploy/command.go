// Package redeploy is the manifest-first convenience wrapper: create or
// update the deployment record from a manifest, then run the same
// plan → review → apply → wait path as `deployment execute`. It carries no
// policy of its own: whether anything must change is the compiled plan's
// outcome (no_op, apply, needs_input), never a client-side health guess.
package redeploy

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/internal/apierr"
)

// Run executes the redeploy workflow.
func Run(client *deployment.Client, args []string) error {
	fs := flag.NewFlagSet("redeploy", flag.ContinueOnError)
	name := fs.String("name", "", "Optional deployment name")
	yes := fs.Bool("yes", false, "Skip printing the plan review before applying")
	noWait := fs.Bool("no-wait", false, "Return after admission (exit 3) instead of waiting once")
	requestKey := fs.String("request-key", "", "Idempotency key for the admitted operation")
	forceBundle := fs.Bool("force-bundle", false, "Rebuild the release bundle before compiling the plan")
	preflight := fs.Bool("preflight", false, "Run the target preflight checks as part of the operation")
	timeout := fs.Duration("timeout", 0, "Observer bound for the wait (default 300s)")
	showCommands := fs.Bool("show-commands", false, "Also print the derived shell preview")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return printUsage()
	}
	manifestBytes, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	body, created, err := client.Create(deployment.CreateRequest{Name: *name, Manifest: json.RawMessage(manifestBytes)})
	if err != nil {
		return err
	}
	if created.Deployment == nil || strings.TrimSpace(created.Deployment.ID) == "" {
		return apierr.Failed("create returned no deployment id: %s", string(body))
	}
	if !*jsonOutput {
		action := "Created"
		if created.Updated {
			action = "Updated"
		}
		fmt.Printf("%s deployment record: %s\n", action, created.Deployment.ID)
	}
	executeArgs := []string{"--deployment", created.Deployment.ID}
	if *yes {
		executeArgs = append(executeArgs, "--yes")
	}
	if *noWait {
		executeArgs = append(executeArgs, "--no-wait")
	}
	if *jsonOutput {
		executeArgs = append(executeArgs, "--json")
	}
	if *showCommands {
		executeArgs = append(executeArgs, "--show-commands")
	}
	if *forceBundle {
		executeArgs = append(executeArgs, "--force-bundle")
	}
	if *preflight {
		executeArgs = append(executeArgs, "--preflight")
	}
	if strings.TrimSpace(*requestKey) != "" {
		executeArgs = append(executeArgs, "--request-key", *requestKey)
	}
	if *timeout > 0 {
		executeArgs = append(executeArgs, "--timeout", timeout.String())
	}
	return deployment.Run(client, append([]string{"execute"}, executeArgs...))
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud redeploy <manifest.json> [options]

Creates or updates the deployment record from the manifest, then compiles
the executable plan, prints the review, applies the exact digest shown and
waits once on the admitted operation (the same path as 'deployment execute').
A satisfied desired state is reported as "no change" (exit 0); missing
input prints the onboarding handoff (exit 3).

Options:
  --name <name>        Optional deployment name
  --yes                Skip printing the plan review before applying
  --no-wait            Return after admission (exit 3) instead of waiting once
  --request-key <key>  Idempotency key for the admitted operation
  --force-bundle       Rebuild the release bundle before compiling the plan
  --preflight          Run the target preflight checks as part of the operation
  --timeout <dur>      Observer bound for the wait (default 300s)
  --show-commands      Also print the derived shell preview
  --json               Output proto JSON

Examples:
  scenario-to-cloud redeploy cloud-manifest.json
  scenario-to-cloud redeploy cloud-manifest.json --yes --timeout 15m
  scenario-to-cloud redeploy cloud-manifest.json --no-wait --json`)
	return nil
}
