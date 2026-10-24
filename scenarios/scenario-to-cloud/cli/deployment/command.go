package deployment

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	deploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/identity"

	"scenario-to-cloud/cli/internal/protoout"
	"scenario-to-cloud/cli/internal/selector"
)

// Run executes deployment subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
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
	case "plan":
		return runPlan(client, args[1:])
	case "apply":
		return runApply(client, args[1:])
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
	case "recovery-points":
		return runRecoveryPoints(client, args[1:])
	case "rollback":
		return runRollback(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud deployment help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud deployment <command> [arguments]

Every command that targets a deployment takes one selector:
  ` + selector.Usage + `
An ambiguous selector is refused (exit 2) with the candidate ids listed.

Commands:
  create <manifest.json>          Create or update a deployment record from a manifest
  resolve <selector>              Resolve a selector to its stable identity
  list                            List deployments (--scenario, --environment, --status)
  get <selector>                  Read a deployment record
  delete <selector>               Delete a deployment record (--stop, --cleanup)
  plan <selector>                 Preview the executable plan (--scope, --force-bundle, --show-commands)
  apply <selector> --plan-digest  Admit a reviewed plan as a durable operation (--preflight, --no-wait)
  execute <selector>              Plan, print the review, apply and wait once (--yes, --force-bundle, --preflight, --no-wait)
  start <selector>                Plan and apply the start scope (--yes, --no-wait)
  stop <selector>                 Stop the workload
  history <selector>              Show deployment history
  health <selector>               Typed health observation (status, freshness, checks)
  recovery-points <verb>          list | capture | verify | restore (--dry-run)
  rollback <selector>             Governed rollback (--dry-run | --confirm --preview-ref)

Governed publication lives under 'scenario-to-cloud publication <request|apply|status>'.

Exit codes: 0 ok / no change, 1 failed, 2 refused (ambiguous selector,
stale plan, authority), 3 pending (operation admitted, input required),
124 observer timeout (operation unchanged; reattach command printed).

Run 'scenario-to-cloud deployment <command> -h' for command-specific options.`)
	return nil
}

// resolveArgs parses the selector flags plus a positional id and resolves
// the deployment.
func resolveArgs(client *Client, fs *flag.FlagSet, sel *selector.Flags) (*identityv1.DeploymentRef, error) {
	chosen, err := sel.Selector(fs.Args())
	if err != nil {
		return nil, err
	}
	return client.Resolve(context.Background(), chosen)
}

func runCreate(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment create", flag.ContinueOnError)
	name := fs.String("name", "", "Optional deployment name")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud deployment create <manifest.json> [--name <name>]")
	}
	manifestBytes, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	body, resp, err := client.Create(CreateRequest{Name: *name, Manifest: json.RawMessage(manifestBytes)})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	printCreated(resp)
	return nil
}

func printCreated(resp CreateResponse) {
	action := "Created"
	if resp.Updated {
		action = "Updated"
	}
	d := resp.Deployment
	if d == nil {
		fmt.Printf("%s deployment.\n", action)
		return
	}
	fmt.Printf("%s deployment: %s\n", action, d.ID)
	fmt.Printf("  name:        %s\n", d.Name)
	fmt.Printf("  scenario:    %s\n", d.ScenarioID)
	if d.Environment != "" {
		fmt.Printf("  environment: %s\n", d.Environment)
	}
	fmt.Printf("  status:      %s\n", d.Status)
}

func runList(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment list", flag.ContinueOnError)
	status := fs.String("status", "", "Filter by status")
	scenario := fs.String("scenario", "", "Filter by scenario id")
	environment := fs.String("environment", "", "Filter by environment")
	pageSize := fs.Int("page-size", 0, "Maximum records (server default when 0)")
	pageToken := fs.String("page-token", "", "Continuation token from a previous page")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	resp, err := client.List(context.Background(), &deploymentsv1.ListDeploymentsRequest{
		ScenarioId: *scenario, Environment: *environment, Status: *status, PageSize: int32(*pageSize), PageToken: *pageToken,
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(resp)
	}
	if len(resp.GetDeployments()) == 0 {
		fmt.Println("No deployments found.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEPLOYMENT\tSCENARIO\tENVIRONMENT\tTARGET\tSTATUS\tDOMAIN\tCREATED")
	for _, d := range resp.GetDeployments() {
		ref := d.GetRef()
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ref.GetId(), ref.GetScenarioId(), ref.GetEnvironment(), selector.TargetKey(ref), d.GetStatus(), d.GetDomain(), formatTimestamp(d.GetCreatedAt().AsTime(), d.GetCreatedAt() != nil))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if resp.GetNextPageToken() != "" {
		fmt.Printf("More: --page-token %s\n", resp.GetNextPageToken())
	}
	return nil
}

func formatTimestamp(t time.Time, ok bool) string {
	if !ok {
		return "n/a"
	}
	return t.UTC().Format(time.RFC3339)
}

func runGet(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment get", flag.ContinueOnError)
	sel := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	resp, err := client.Get(context.Background(), ref.GetId())
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(resp)
	}
	d := resp.GetDeployment()
	fmt.Println(selector.Identity(d.GetRef()))
	fmt.Printf("  name:           %s\n", d.GetName())
	fmt.Printf("  status:         %s\n", d.GetStatus())
	fmt.Printf("  fence:          %d\n", d.GetFence())
	if d.GetDomain() != "" {
		fmt.Printf("  domain:         %s\n", d.GetDomain())
	}
	if d.GetBundleSha256() != "" {
		fmt.Printf("  release digest: sha256:%s\n", d.GetBundleSha256())
	}
	if d.GetProgressStep() != "" {
		fmt.Printf("  progress step:  %s\n", d.GetProgressStep())
	}
	if d.GetErrorMessage() != "" {
		fmt.Printf("  error:          %s\n", d.GetErrorMessage())
	}
	fmt.Printf("  created:        %s\n", formatTimestamp(d.GetCreatedAt().AsTime(), d.GetCreatedAt() != nil))
	fmt.Printf("  last deployed:  %s\n", formatTimestamp(d.GetLastDeployedAt().AsTime(), d.GetLastDeployedAt() != nil))
	return nil
}

func runResolve(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment resolve", flag.ContinueOnError)
	sel := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	chosen, err := sel.Selector(fs.Args())
	if err != nil {
		return err
	}
	ref, err := client.Resolve(context.Background(), chosen)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(&deploymentsv1.ResolveDeploymentResponse{SchemaVersion: "1", Ref: ref})
	}
	fmt.Printf("Selector: %s\n", chosen)
	fmt.Println(selector.Identity(ref))
	if loc := ref.GetTarget().GetLocator(); loc != nil && loc.GetHost() != "" {
		fmt.Printf("  locator: %s@%s:%d %s\n", loc.GetUser(), loc.GetHost(), loc.GetPort(), loc.GetWorkdir())
	}
	return nil
}

func runDelete(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment delete", flag.ContinueOnError)
	sel := selector.Register(fs)
	stop := fs.Bool("stop", false, "Stop the workload on the target before deleting")
	cleanup := fs.Bool("cleanup", false, "Clean up bundle files")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	body, _, err := client.Delete(ref.GetId(), DeleteOptions{Stop: *stop, Cleanup: *cleanup})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(selector.Identity(ref))
	fmt.Println("Deployment record deleted.")
	return nil
}

func runStop(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment stop", flag.ContinueOnError)
	sel := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	body, resp, err := client.Stop(ref.GetId())
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(selector.Identity(ref))
	if !resp.Success {
		return fmt.Errorf("deployment stop failed: %s", resp.Error)
	}
	fmt.Println("Deployment stopped.")
	return nil
}

func runHistory(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment history", flag.ContinueOnError)
	sel := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	body, resp, err := client.History(ref.GetId())
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(selector.Identity(ref))
	if len(resp.Events) == 0 {
		fmt.Println("No history events found.")
		return nil
	}
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
		fmt.Printf("%s  %s%s\n", e.Timestamp.UTC().Format("2006-01-02 15:04:05"), e.Message, status)
		if e.Details != "" {
			fmt.Printf("           %s\n", e.Details)
		}
	}
	return nil
}
