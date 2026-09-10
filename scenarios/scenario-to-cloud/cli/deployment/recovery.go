package deployment

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/selector"
)

func runRecoveryPoints(client *Client, args []string) error {
	if len(args) == 0 {
		return printRecoveryPointsUsage()
	}
	switch args[0] {
	case "list":
		return runRecoveryPointsList(client, args[1:])
	case "capture":
		return runRecoveryPointsCapture(client, args[1:])
	case "verify":
		return runRecoveryPointsVerify(client, args[1:])
	case "restore":
		return runRecoveryPointsRestore(client, args[1:])
	case "help", "-h", "--help":
		return printRecoveryPointsUsage()
	default:
		return fmt.Errorf("unknown recovery-points subcommand: %s", args[0])
	}
}

func printRecoveryPointsUsage() error {
	fmt.Println(`Usage: scenario-to-cloud deployment recovery-points <command> <selector> [flags]

Commands:
  list <selector>                                   Recovery points of the deployment
  capture <selector> [--retention <policy>]         Capture a consistent recovery point
  verify <selector> --recovery-point <id> [--open]  Verify artifacts (and the recovery key with --open)
  restore <selector> --recovery-point <id> --target-ref <ref> [--into binding=locator ...] [--dry-run]
                                                    Restore onto a replacement target; --dry-run verifies
                                                    the point and prints what would be restored, calling no restore

Selector: ` + selector.Usage)
	return nil
}

func runRecoveryPointsList(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment recovery-points list", flag.ContinueOnError)
	sel := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	body, resp, err := client.ListRecoveryPoints(ref.GetId())
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(selector.Identity(ref))
	if len(resp.RecoveryPoints) == 0 {
		fmt.Println("No recovery points.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RECOVERY POINT\tCAPTURED\tRELEASE DIGEST\tSCHEMA\tPROVIDER\tPROTECTED")
	for _, rp := range resp.RecoveryPoints {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%t\n", rp.ID, rp.CapturedAt.UTC().Format(time.RFC3339), rp.ReleaseDigest, rp.SchemaVersion, rp.Provider, rp.Protected)
	}
	return w.Flush()
}

func runRecoveryPointsCapture(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment recovery-points capture", flag.ContinueOnError)
	sel := selector.Register(fs)
	retention := fs.String("retention", "", "Retention policy name")
	pointID := fs.String("recovery-point", "", "Explicit recovery point id (server derives one when empty)")
	releaseDigest := fs.String("release-digest", "", "Release digest the point is bound to (defaults to the recorded release)")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	body, resp, err := client.CaptureRecoveryPoint(ref.GetId(), RecoveryPointCaptureRequest{RetentionPolicy: *retention, RecoveryPointID: *pointID, ReleaseDigest: *releaseDigest})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(selector.Identity(ref))
	if rp := resp.RecoveryPoint; rp != nil {
		fmt.Printf("recovery point: %s\n", rp.ID)
		fmt.Printf("  captured:       %s\n", rp.CapturedAt.UTC().Format(time.RFC3339))
		fmt.Printf("  release digest: %s\n", rp.ReleaseDigest)
		fmt.Printf("  schema:         %s\n", rp.SchemaVersion)
		fmt.Printf("  provider:       %s  encrypted: %t  key ref: %s\n", rp.Provider, rp.Encrypted, rp.RecoveryKeyRef)
	}
	return nil
}

func runRecoveryPointsVerify(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment recovery-points verify", flag.ContinueOnError)
	sel := selector.Register(fs)
	pointID := fs.String("recovery-point", "", "Recovery point id (required)")
	open := fs.Bool("open", false, "Also open the sealed artifacts to prove the recovery key resolves")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*pointID) == "" {
		return apierr.Refused("--recovery-point is required")
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	body, resp, err := client.VerifyRecoveryPoint(ref.GetId(), *pointID, *open)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(selector.Identity(ref))
	printVerifyReport(*pointID, resp.Report)
	if resp.Report != nil && resp.Report.Outcome != "" && resp.Report.Outcome != "verified" && resp.Report.Outcome != "succeeded" {
		return apierr.Failed("recovery point %s verification outcome: %s", *pointID, resp.Report.Outcome)
	}
	return nil
}

func printVerifyReport(pointID string, report *RecoveryPointVerifyReport) {
	if report == nil {
		fmt.Printf("recovery point %s: no report returned\n", pointID)
		return
	}
	fmt.Printf("recovery point: %s\n", pointID)
	fmt.Printf("  outcome:          %s\n", report.Outcome)
	fmt.Printf("  artifacts intact: %t  key resolved: %t  artifacts opened: %t\n", report.ArtifactsIntact, report.KeyResolved, report.ArtifactsOpened)
	for _, inv := range report.Invariants {
		mark := "ok"
		if !inv.Passed {
			mark = "FAILED"
		}
		fmt.Printf("  %s %s/%s expected=%s observed=%s\n", mark, inv.Binding, inv.Check, inv.Expected, inv.Observed)
	}
}

func runRecoveryPointsRestore(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment recovery-points restore", flag.ContinueOnError)
	sel := selector.Register(fs)
	pointID := fs.String("recovery-point", "", "Recovery point id (required)")
	targetRef := fs.String("target-ref", "", "Replacement target reference (required)")
	dryRun := fs.Bool("dry-run", false, "Verify the point and print what would be restored; call no restore")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	var into intoFlag
	fs.Var(&into, "into", "binding=locator on the replacement host (repeatable)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*pointID) == "" {
		return apierr.Refused("--recovery-point is required")
	}
	if strings.TrimSpace(*targetRef) == "" && !*dryRun {
		return apierr.Refused("--target-ref is required (or use --dry-run)")
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	if *dryRun {
		body, resp, err := client.VerifyRecoveryPoint(ref.GetId(), *pointID, true)
		if err != nil {
			return err
		}
		if *jsonOutput {
			cliutil.PrintJSON(body)
			return nil
		}
		fmt.Println(selector.Identity(ref))
		fmt.Println("dry run: no restore was performed")
		printVerifyReport(*pointID, resp.Report)
		if resp.Report != nil && resp.Report.RecoveryPoint != nil {
			fmt.Printf("would restore bindings: %s\n", strings.Join(resp.Report.RecoveryPoint.BindingIDs, ", "))
		}
		if *targetRef != "" {
			fmt.Printf("onto target: %s\n", *targetRef)
		}
		return nil
	}
	body, resp, err := client.RestoreRecoveryPoint(ref.GetId(), *pointID, RecoveryPointRestoreRequest{TargetRef: *targetRef, Into: into.values})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(selector.Identity(ref))
	receipt := resp.Receipt
	if receipt == nil {
		return apierr.Failed("restore returned no receipt")
	}
	fmt.Printf("restore receipt: %s\n", receipt.ID)
	fmt.Printf("  recovery point: %s  target: %s\n", receipt.RecoveryPointID, receipt.TargetRef)
	fmt.Printf("  outcome:        %s  within budgets: %t\n", receipt.Outcome, receipt.WithinBudgets)
	for _, inv := range receipt.InvariantResults {
		mark := "ok"
		if !inv.Passed {
			mark = "FAILED"
		}
		fmt.Printf("  %s %s/%s expected=%s observed=%s\n", mark, inv.Binding, inv.Check, inv.Expected, inv.Observed)
	}
	if receipt.Outcome != "succeeded" {
		return apierr.Failed("restore %s: %s (%s)", receipt.Outcome, receipt.ErrorMessage, receipt.ErrorCode)
	}
	return nil
}

// intoFlag collects repeated binding=locator pairs.
type intoFlag struct{ values map[string]string }

func (f *intoFlag) String() string { return fmt.Sprint(f.values) }

func (f *intoFlag) Set(v string) error {
	key, value, ok := strings.Cut(v, "=")
	if !ok || strings.TrimSpace(key) == "" {
		return fmt.Errorf("--into expects binding=locator, got %q", v)
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	return nil
}

// runRollback drives the governed cloud recovery: --dry-run issues the
// preview_ref; --confirm executes with that preview_ref. Both are the
// server's decisions; the CLI never infers a rollback target.
func runRollback(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment rollback", flag.ContinueOnError)
	sel := selector.Register(fs)
	dryRun := fs.Bool("dry-run", false, "Preview the rollback and print the preview reference")
	confirm := fs.Bool("confirm", false, "Execute the rollback previewed by --dry-run")
	previewRef := fs.String("preview-ref", "", "Preview reference issued by --dry-run (required with --confirm)")
	reviewRef := fs.String("review-ref", "", "Review reference of the governed publication (required)")
	releaseDigest := fs.String("release-digest", "", "Exact predecessor release digest (defaults to the published predecessor)")
	repairSHA := fs.String("repair-bundle-sha256", "", "Retained predecessor bundle sha256")
	expectedSHA := fs.String("expected-bundle-sha256", "", "Bundle sha256 currently recorded on the deployment")
	dataCompat := fs.String("data-compatibility", "", "Declared data compatibility of the rollback")
	requestKey := fs.String("request-key", "", "Idempotency key for the recovery operation")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *dryRun == *confirm {
		return apierr.Refused("choose exactly one of --dry-run or --confirm")
	}
	if *confirm && strings.TrimSpace(*previewRef) == "" {
		return apierr.Refused("--confirm requires --preview-ref from a previous --dry-run")
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	body, resp, err := client.Recovery(ref.GetId(), RecoveryRequest{
		Action: "rollback", DryRun: *dryRun, PreviewRef: *previewRef, ReviewRef: *reviewRef, ReleaseDigest: *releaseDigest,
		RepairBundleSHA: *repairSHA, ExpectedBundleSHA: *expectedSHA, DataCompatibility: *dataCompat, IdempotencyKey: *requestKey,
		Confirmation: confirmation(*confirm, ref.GetId()),
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(selector.Identity(ref))
	r := resp.Receipt
	if r == nil {
		return apierr.Failed("recovery returned no receipt")
	}
	fmt.Printf("rollback %s: outcome %s  health %s  route %s\n", map[bool]string{true: "preview", false: "execution"}[r.DryRun], r.Outcome, r.Health, r.RouteKind)
	if r.ReleaseDigest != "" {
		fmt.Printf("  release digest: %s\n", r.ReleaseDigest)
	}
	if r.OperationID != "" {
		fmt.Printf("  operation: %s\n", r.OperationID)
	}
	if r.DryRun {
		fmt.Printf("  preview ref: %s\n", r.PreviewRef)
		fmt.Printf("  execute with: scenario-to-cloud deployment rollback --deployment %s --confirm --preview-ref %s --review-ref %s\n", ref.GetId(), r.PreviewRef, *reviewRef)
		return nil
	}
	if r.Error != "" {
		return apierr.Failed("rollback %s: %s", r.Outcome, r.Error)
	}
	return nil
}

// confirmation is the explicit operator confirmation string a recovery
// execution carries; the server decides whether it is required.
func confirmation(confirm bool, deploymentID string) string {
	if !confirm {
		return ""
	}
	return "ROLLBACK " + deploymentID
}
